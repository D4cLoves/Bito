package postgres

import (
	"context"
	"fmt"
	"time"

	"bito/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns        = int32(25)
	defaultMinConns        = int32(5)
	defaultMaxConnLifetime = 5 * time.Minute
	defaultMaxConnIdleTime = 1 * time.Minute
	defaultHealthCheckWait = 5 * time.Second
)

func NewDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres parse config: %w", err)
	}

	poolConfig.MaxConns = defaultMaxConns
	poolConfig.MinConns = defaultMinConns
	poolConfig.MaxConnLifetime = defaultMaxConnLifetime
	poolConfig.MaxConnIdleTime = defaultMaxConnIdleTime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, defaultHealthCheckWait)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return pool, nil
}
