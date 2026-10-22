//go:build integration

package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("BITO_TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		if os.Getenv("BITO_REQUIRE_INTEGRATION") == "1" {
			t.Fatalf("redis is not reachable: %v", err)
		}
		t.Skipf("redis is not reachable (%v); start docker compose to run integration tests", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func savePending(t *testing.T, repo *AuthRepository, email, code string) {
	t.Helper()
	err := repo.SavePending(context.Background(), &model.PendingRegistration{Email: email, Code: code, Name: "p"}, time.Minute)
	if err != nil {
		t.Fatalf("save pending: %v", err)
	}
}

func TestAuth_CodeBurnsAfterFiveWrongAttempts(t *testing.T) {
	repo := NewAuthRepository(setupRedis(t))
	ctx := context.Background()
	email := uuid.NewString() + "@bito.test"
	savePending(t, repo, email, "111111")

	for i := 1; i < MaxCodeAttempts; i++ {
		if _, err := repo.VerifyAndGetPending(ctx, email, "000000"); !errors.Is(err, model.ErrCodeMismatch) {
			t.Fatalf("wrong attempt %d: err = %v, want mismatch", i, err)
		}
	}
	if _, err := repo.VerifyAndGetPending(ctx, email, "000000"); !errors.Is(err, model.ErrCodeAttempts) {
		t.Fatalf("fifth wrong attempt: err = %v, want ErrCodeAttempts", err)
	}
	if _, err := repo.VerifyAndGetPending(ctx, email, "111111"); !errors.Is(err, model.ErrPendingNotFound) {
		t.Errorf("right code after the burn: err = %v, want ErrPendingNotFound", err)
	}
}

func TestAuth_NewCodeResetsAttempts(t *testing.T) {
	repo := NewAuthRepository(setupRedis(t))
	ctx := context.Background()
	email := uuid.NewString() + "@bito.test"
	savePending(t, repo, email, "111111")

	for i := 1; i < MaxCodeAttempts; i++ {
		_, _ = repo.VerifyAndGetPending(ctx, email, "000000")
	}
	savePending(t, repo, email, "222222")

	for i := 1; i < MaxCodeAttempts; i++ {
		if _, err := repo.VerifyAndGetPending(ctx, email, "000000"); !errors.Is(err, model.ErrCodeMismatch) {
			t.Fatalf("wrong attempt %d on the new code: err = %v, want mismatch", i, err)
		}
	}
	got, err := repo.VerifyAndGetPending(ctx, email, "222222")
	if err != nil || got.Email != email {
		t.Fatalf("right code: %+v, %v", got, err)
	}
	if _, err = repo.VerifyAndGetPending(ctx, email, "222222"); !errors.Is(err, model.ErrPendingNotFound) {
		t.Errorf("code must be single-use, err = %v", err)
	}
}

func TestRateLimiter_Window(t *testing.T) {
	limiter := NewRateLimiter(setupRedis(t))
	ctx := context.Background()
	key := "test:" + uuid.NewString()

	for i := 1; i <= 3; i++ {
		ok, _, err := limiter.Allow(ctx, key, 3, time.Minute)
		if err != nil || !ok {
			t.Fatalf("hit %d: ok = %v, err = %v", i, ok, err)
		}
	}
	ok, retryAfter, err := limiter.Allow(ctx, key, 3, time.Minute)
	if err != nil || ok {
		t.Fatalf("hit over the limit: ok = %v, err = %v", ok, err)
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Errorf("retryAfter = %v, want within the window", retryAfter)
	}

	if ok, _, _ = limiter.Allow(ctx, "test:"+uuid.NewString(), 3, time.Minute); !ok {
		t.Error("another key has its own counter")
	}
}
