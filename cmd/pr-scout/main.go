// pr-scout — daily PR review agent for codeready-toolchain.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codeready-toolchain/pr-scout/pkg/api"
	"github.com/codeready-toolchain/pr-scout/pkg/config"
	"github.com/codeready-toolchain/pr-scout/pkg/database"
	"github.com/codeready-toolchain/pr-scout/pkg/github"
	"github.com/codeready-toolchain/pr-scout/pkg/llm"
	"github.com/codeready-toolchain/pr-scout/pkg/services"
)

func main() {
	configureLogging()

	configDir := flag.String("config-dir",
		getEnv("CONFIG_DIR", "./deploy/config"),
		"Path to configuration directory")
	dashboardDir := flag.String("dashboard-dir",
		getEnv("DASHBOARD_DIR", ""),
		"Path to dashboard build directory (empty = no static serving)")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configDir)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize database
	dbClient, err := database.NewClient(ctx, cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	// Initialize GitHub client
	ghToken := cfg.GitHub.Token()
	if ghToken == "" {
		slog.Error("GITHUB_TOKEN not set")
		os.Exit(1)
	}

	ghClient := github.NewClient(ghToken, cfg.GitHub.Org, cfg.GitHub.Username, slog.Default())
	if err := ghClient.ValidateCredentials(ctx); err != nil {
		slog.Error("GitHub authentication failed", "error", err)
		os.Exit(1)
	}
	slog.Info("GitHub authenticated", "org", cfg.GitHub.Org, "user", cfg.GitHub.Username)

	// Initialize LLM client (optional)
	llmClient := llm.NewClient(cfg.LLM, slog.Default())

	// Initialize services
	scanner := github.NewScanner(ghClient, slog.Default())
	tracker := github.NewTracker(ghClient, slog.Default())
	scanService := services.NewScanService(dbClient.DB, ghClient, scanner, tracker, llmClient, slog.Default(), cfg.GitHub.Repos, cfg.GitHub.MaxPRsPerRepo, cfg.LLM.MaxPRAgeDays)
	prService := services.NewPRService(dbClient.DB)
	digestService := services.NewDigestService(dbClient.DB)

	// Start background LLM worker
	scanService.StartLLMWorker(ctx)
	defer scanService.StopLLMWorker()

	// Create HTTP server
	httpServer := api.NewServer(cfg, ghClient, scanService, prService, digestService)

	if *dashboardDir != "" {
		httpServer.SetDashboardDir(*dashboardDir)
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", addr)
		if err := httpServer.Start(addr); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	slog.Info("pr-scout started",
		"port", cfg.Server.Port,
		"database", cfg.Database.Driver,
		"llm", cfg.LLM.Enabled)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		slog.Info("Shutdown signal received", "signal", sig)
	case err := <-errCh:
		slog.Error("Server error", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
	}
	slog.Info("Shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func configureLogging() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
