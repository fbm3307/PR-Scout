// Package api provides the HTTP API server for pr-scout.
package api

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/codeready-toolchain/pr-scout/pkg/config"
	"github.com/codeready-toolchain/pr-scout/pkg/github"
	"github.com/codeready-toolchain/pr-scout/pkg/services"
)

// Server is the pr-scout HTTP API server.
type Server struct {
	echo          *echo.Echo
	httpServer    *http.Server
	cfg           *config.Config
	ghClient      *github.Client
	scanService   *services.ScanService
	prService     *services.PRService
	digestService *services.DigestService
	dashboardDir  string
}

// NewServer creates a new API server.
func NewServer(
	cfg *config.Config,
	ghClient *github.Client,
	scanService *services.ScanService,
	prService *services.PRService,
	digestService *services.DigestService,
) *Server {
	e := echo.New()

	s := &Server{
		echo:          e,
		cfg:           cfg,
		ghClient:      ghClient,
		scanService:   scanService,
		prService:     prService,
		digestService: digestService,
	}

	s.setupRoutes()
	return s
}

// SetDashboardDir configures static file serving for the dashboard build.
func (s *Server) SetDashboardDir(dir string) {
	s.dashboardDir = dir
	s.setupDashboardRoutes()
}

func (s *Server) corsAllowOrigins() []string {
	return []string{
		"http://localhost:5173",
		"http://localhost:8080",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:8080",
		s.cfg.Server.DashboardURL,
	}
}

func (s *Server) setupRoutes() {
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     s.corsAllowOrigins(),
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	s.echo.GET("/health", s.healthHandler)

	v1 := s.echo.Group("/api/v1")
	v1.POST("/scan", s.scanHandler)
	v1.GET("/prs", s.listPRsHandler)
	v1.GET("/prs/:id", s.getPRHandler)
	v1.GET("/my-reviews", s.myReviewsHandler)
	v1.GET("/digest", s.digestHandler)
}

func (s *Server) setupDashboardRoutes() {
	if s.dashboardDir == "" {
		return
	}

	indexPath := filepath.Join(s.dashboardDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		slog.Warn("Dashboard directory set but index.html not found", "dir", s.dashboardDir)
		return
	}

	slog.Info("Serving dashboard", "dir", s.dashboardDir)

	dashFS := os.DirFS(s.dashboardDir)

	// Vite hashed assets
	assetsFS, err := fs.Sub(dashFS, "assets")
	if err == nil {
		s.echo.GET("/assets/*", func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return c.FileFS(c.Param("*"), assetsFS)
		})
	}

	// SPA fallback
	s.echo.GET("/*", func(c *echo.Context) error {
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/health" {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		c.Response().Header().Set("Cache-Control", "no-cache")
		relPath := strings.TrimPrefix(path, "/")
		if relPath != "" {
			if info, statErr := fs.Stat(dashFS, relPath); statErr == nil && !info.IsDir() {
				return c.FileFS(relPath, dashFS)
			}
		}
		return c.FileFS("index.html", dashFS)
	})
}

// Start starts the HTTP server on the given address.
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.echo,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
