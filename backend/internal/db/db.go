// Package db opens and configures the Postgres connection pool used by the
// whole app. We use database/sql directly with lib/pq rather than a full
// ORM — keeps the dependency tree small and every query explicit.
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/yourorg/autograph-backend/internal/config"
)

func Connect(cfg config.DBConfig) (*sql.DB, error) {
	conn, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}

	return conn, nil
}
