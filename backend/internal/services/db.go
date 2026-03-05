package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a new database connection pool
func NewDB(databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnIdleTime = 30 * time.Second
	config.MaxConnLifetime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	// Set statement timeout via connection parameters
	config.ConnConfig.RuntimeParams["statement_timeout"] = "30000"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// InitSchema creates tables if they don't exist and runs cleanup
func (db *DB) InitSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS apps (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name        TEXT UNIQUE NOT NULL,
			repo_url    TEXT NOT NULL,
			branch      TEXT NOT NULL DEFAULT 'main',
			port        INTEGER UNIQUE NOT NULL,
			domain      TEXT,
			ssl_enabled BOOLEAN NOT NULL DEFAULT false,
			env_vars    JSONB NOT NULL DEFAULT '{}',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS managed_databases (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name       TEXT UNIQUE NOT NULL,
			db_user    TEXT UNIQUE NOT NULL,
			password   TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS audit_log (
			id          BIGSERIAL PRIMARY KEY,
			username    TEXT NOT NULL,
			ip          TEXT NOT NULL,
			method      TEXT NOT NULL,
			path        TEXT NOT NULL,
			status_code INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			body        JSONB DEFAULT '{}',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		DELETE FROM audit_log WHERE created_at < NOW() - INTERVAL '90 days';
	`

	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	log.Println("Database schema initialized")
	return nil
}

// Query executes a query and returns rows
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

// QueryRow executes a query expecting a single row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}

// Exec executes a statement
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(ctx, sql, args...)
}

// Close closes the database pool
func (db *DB) Close() {
	db.Pool.Close()
}
