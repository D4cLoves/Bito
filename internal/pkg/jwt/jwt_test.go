package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManager_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	ttl := 15 * time.Minute
	tm := NewTokenManager(secret, ttl)

	userID := uuid.New()
	username := "vladislav"

	tokenStr, err := tm.GenerateAccessToken(userID, username)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := tm.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %v, got %v", userID, claims.UserID)
	}

	if claims.Username != username {
		t.Errorf("expected username %q, got %q", username, claims.Username)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	ttl := -1 * time.Minute // expired in the past
	tm := NewTokenManager(secret, ttl)

	userID := uuid.New()
	username := "vladislav"

	tokenStr, err := tm.GenerateAccessToken(userID, username)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	_, err = tm.ValidateAccessToken(tokenStr)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestTokenManager_InvalidSecret(t *testing.T) {
	tm1 := NewTokenManager("secret-1", time.Hour)
	tm2 := NewTokenManager("secret-2", time.Hour)

	userID := uuid.New()
	username := "vladislav"

	tokenStr, err := tm1.GenerateAccessToken(userID, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tm2.ValidateAccessToken(tokenStr)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}
