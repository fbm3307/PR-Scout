// Package database provides a thin DB abstraction supporting SQLite and PostgreSQL.
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/codeready-toolchain/pr-scout/pkg/config"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

//go:embed migrations
var migrationsFS embed.FS

// Client wraps a *sql.DB and exposes the driver name for query dialect.
type Client struct {
	DB     *sql.DB
	Driver string // "sqlite3" or "pgx"
}

// NewClient opens a database connection based on config and runs migrations.
func NewClient(ctx context.Context, cfg config.DatabaseConfig) (*Client, error) {
	var (
		db         *sql.DB
		driverName string
		err        error
	)

	switch cfg.Driver {
	case "sqlite", "sqlite3":
		driverName = "sqlite3"
		db, err = sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_foreign_keys=on")
	case "postgres", "postgresql":
		driverName = "pgx"
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
		)
		db, err = sql.Open("pgx", dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %q (use sqlite or postgres)", cfg.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	client := &Client{DB: db, Driver: driverName}
	if err := client.runMigrations(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("Database connected", "driver", cfg.Driver)
	return client, nil
}

// Close shuts down the database connection.
func (c *Client) Close() error {
	return c.DB.Close()
}

func (c *Client) runMigrations() error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	var m *migrate.Migrate

	switch c.Driver {
	case "sqlite3":
		dbDriver, err := sqlite3.WithInstance(c.DB, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("create sqlite3 migration driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
		if err != nil {
			return fmt.Errorf("create migrate instance: %w", err)
		}
	case "pgx":
		dbDriver, err := postgres.WithInstance(c.DB, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("create postgres migration driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
		if err != nil {
			return fmt.Errorf("create migrate instance: %w", err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	// Only close the source driver — the database driver shares the *sql.DB
	if srcErr := sourceDriver.Close(); srcErr != nil {
		slog.Warn("Failed to close migration source", "error", srcErr)
	}

	return nil
}
