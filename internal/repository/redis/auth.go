package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bito/internal/model"

	"github.com/redis/go-redis/v9"
)

const pendingRegistrationPrefix = "auth:pending:"

type AuthRepository struct {
	client *redis.Client
}

func NewAuthRepository(client *redis.Client) *AuthRepository {
	return &AuthRepository{client: client}
}

// SavePending ??????????? ?????? ??????????? ? JSON ? ????????? ? Redis ? TTL
func (r *AuthRepository) SavePending(ctx context.Context, data *model.PendingRegistration, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s", pendingRegistrationPrefix, data.Email)

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("redis: marshal pending: %w", err)
	}

	if err := r.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("redis: save pending: %w", err)
	}

	return nil
}

// VerifyAndGetPending ??????? ?????? ???????????, ??????? ??? ? ??? ?????? ??????? ?????? ?? Redis
func (r *AuthRepository) VerifyAndGetPending(ctx context.Context, email, code string) (*model.PendingRegistration, error) {
	key := fmt.Sprintf("%s%s", pendingRegistrationPrefix, email)

	payload, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, model.ErrPendingNotFound
		}
		return nil, fmt.Errorf("redis: get pending: %w", err)
	}

	var data model.PendingRegistration
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("redis: unmarshal pending: %w", err)
	}

	if data.Code != code {
		return nil, model.ErrCodeMismatch
	}

	_ = r.client.Del(ctx, key)

	return &data, nil
}
