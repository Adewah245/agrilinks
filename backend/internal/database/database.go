package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultDBMaxOpenConns    = 20
	defaultDBMaxIdleConns    = 10
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBConnMaxIdleTime = 5 * time.Minute
	defaultDBPingTimeout     = 10 * time.Second
)

// ConnectDB opens a PostgreSQL connection pool using the supplied database URL.
//
// The caller owns the returned pool and must close it when the application
// stops. Schema setup deliberately lives in the numbered files under
// backend/migrations and is handled by RunMigration.
func ConnectDB(databaseURL string) (*sql.DB, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL must not be empty")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}

	// Bound the pool so one process cannot exhaust a shared PostgreSQL or Neon
	// connection limit. These are configurable for deployment sizing.
	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns, 1, 200))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns, 1, 100))
	db.SetConnMaxLifetime(envDuration("DB_CONN_MAX_LIFETIME", defaultDBConnMaxLifetime))
	db.SetConnMaxIdleTime(envDuration("DB_CONN_MAX_IDLE_TIME", defaultDBConnMaxIdleTime))

	pingCtx, cancel := context.WithTimeout(context.Background(), defaultDBPingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return db, nil
}

func envInt(name string, fallback, minimum, maximum int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum {
		log.Printf("invalid %s=%q; using %d", name, raw, fallback)
		return fallback
	}
	if value > maximum {
		return maximum
	}

	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		log.Printf("invalid %s=%q; using %s", name, raw, fallback)
		return fallback
	}

	return value
}
