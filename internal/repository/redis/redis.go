package redis

import (
	"context"
	"fmt"
	"net"
	"time"

	"bito/internal/config"

	"github.com/redis/go-redis/v9"
)

const (
	defaultPoolSize        = 50
	defaultMinIdleConns    = 10
	defaultConnMaxIdleTime = 5 * time.Minute
	defaultDialTimeout     = 5 * time.Second
	defaultReadTimeout     = 3 * time.Second
	defaultWriteTimeout    = 3 * time.Second
	defaultPingTimeout     = 3 * time.Second
)

func New(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	addr := net.JoinHostPort(cfg.RedisHost, cfg.RedisPort)

	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        cfg.RedisPassword,
		DB:              cfg.RedisDB,
		PoolSize:        defaultPoolSize,
		MinIdleConns:    defaultMinIdleConns,
		ConnMaxIdleTime: defaultConnMaxIdleTime,
		DialTimeout:     defaultDialTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping %s: %w", addr, err)
	}

	return client, nil
}
