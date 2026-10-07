package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	pendingRegistrationPrefix = "auth:pending:"
	refreshTokenPrefix        = "auth:refresh:"
)

type AuthRepository struct {
	client *redis.Client
}

func NewAuthRepository(client *redis.Client) *AuthRepository {
	return &AuthRepository{client: client}
}

// SavePending сохраняет данные незавершенной регистрации в JSON в Redis с TTL
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

// VerifyAndGetPending считывает данные регистрации, сверяет код и при успехе удаляет ключ
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

// SaveRefreshToken сохраняет сессию refresh токена в Redis с TTL
func (r *AuthRepository) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s:%s", refreshTokenPrefix, userID.String(), tokenID)
	if err := r.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("redis: save refresh token: %w", err)
	}
	return nil
}

// ValidateAndRevokeRefreshToken атомарно удаляет старый токен и возвращает true, если токен был активен
func (r *AuthRepository) ValidateAndRevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error) {
	key := fmt.Sprintf("%s%s:%s", refreshTokenPrefix, userID.String(), tokenID)
	deleted, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis: validate and revoke refresh token: %w", err)
	}
	return deleted > 0, nil
}

// RevokeRefreshToken аннулирует токен при логауте
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) error {
	key := fmt.Sprintf("%s%s:%s", refreshTokenPrefix, userID.String(), tokenID)
	return r.client.Del(ctx, key).Err()
}
