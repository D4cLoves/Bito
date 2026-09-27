package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrPendingNotFound = errors.New("registration request not found or expired")
	ErrCodeMismatch    = errors.New("invalid verification code")
)

const pendingRegistrationPrefix = "auth:pending:"

type PendingRegistration struct {
	Code         string `json:"code"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"passwordHash"`
}

type AuthRepository struct {
	client *redis.Client
}

func NewAuthRepository(client *redis.Client) *AuthRepository {
	return &AuthRepository{client: client}
}

// SavePending сериализует данные регистрации в JSON и сохраняет в Redis с TTL
func (r *AuthRepository) SavePending(ctx context.Context, data *PendingRegistration, ttl time.Duration) error {
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

// VerifyAndGetPending достает данные регистрации, сверяет код и при успехе удаляет запись из Redis
func (r *AuthRepository) VerifyAndGetPending(ctx context.Context, email, code string) (*PendingRegistration, error) {
	key := fmt.Sprintf("%s%s", pendingRegistrationPrefix, email)

	payload, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrPendingNotFound
		}
		return nil, fmt.Errorf("redis: get pending: %w", err)
	}

	var data PendingRegistration
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("redis: unmarshal pending: %w", err)
	}

	if data.Code != code {
		return nil, ErrCodeMismatch
	}

	_ = r.client.Del(ctx, key)

	return &data, nil
}
