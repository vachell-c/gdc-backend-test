package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a new pgxpool connection pool from the given database URL,
// instrumented with OpenTelemetry tracing via otelpgx.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}
	config.ConnConfig.Tracer = otelpgx.NewTracer()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	return pool, nil
}

// Ping checks the database connection by pinging the pool.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}

// RunMigrations reads and executes all SQL migration files from the
// internal/db/migrations directory.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := "internal/db/migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		migrationsDir = "migrations"
		entries, err = os.ReadDir(migrationsDir)
		if err != nil {
			return fmt.Errorf("failed to read migrations directory: %w", err)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		migrationPath := filepath.Join(migrationsDir, entry.Name())
		sql, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationPath, err)
		}
	}

	return nil
}
