package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManager_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour
	tm := NewTokenManager(secret, accessTTL, refreshTTL)

	userID := uuid.New()
	username := "vladislav"

	pair, err := tm.GenerateTokenPair(userID, username)
	if err != nil {
		t.Fatalf("unexpected error generating token pair: %v", err)
	}

	// Validate Access
	accessClaims, err := tm.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error validating access token: %v", err)
	}
	if accessClaims.UserID != userID || accessClaims.Username != username {
		t.Errorf("access claims mismatch: %v", accessClaims)
	}

	// Validate Refresh
	refreshClaims, err := tm.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error validating refresh token: %v", err)
	}
	if refreshClaims.UserID != userID || refreshClaims.Username != username {
		t.Errorf("refresh claims mismatch: %v", refreshClaims)
	}

	// Cross validation must fail (cannot use refresh as access, or vice versa)
	if _, err := tm.ValidateAccessToken(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken when validating refresh as access, got %v", err)
	}
	if _, err := tm.ValidateRefreshToken(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken when validating access as refresh, got %v", err)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	tm := NewTokenManager(secret, -1*time.Minute, -1*time.Minute)

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
	tm1 := NewTokenManager("secret-1", time.Hour, 24*time.Hour)
	tm2 := NewTokenManager("secret-2", time.Hour, 24*time.Hour)

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
