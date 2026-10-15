package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitPrefix = "ratelimit:"

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	fullKey := rateLimitPrefix + key

	var hits *redis.IntCmd
	var ttl *redis.DurationCmd
	_, err := l.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		hits = pipe.Incr(ctx, fullKey)
		pipe.ExpireNX(ctx, fullKey, window)
		ttl = pipe.PTTL(ctx, fullKey)
		return nil
	})
	if err != nil {
		return false, 0, fmt.Errorf("redis: rate limit %s: %w", key, err)
	}

	if hits.Val() > int64(limit) {
		return false, ttl.Val(), nil
	}
	return true, 0, nil
}
