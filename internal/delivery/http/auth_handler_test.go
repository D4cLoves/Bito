package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bito/internal/model"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

type mockAuthService struct {
	user        *model.User
	err         error
	activeTokens map[string]bool
}

func newMockAuthService(user *model.User) *mockAuthService {
	return &mockAuthService{
		user:         user,
		activeTokens: make(map[string]bool),
	}
}

func (m *mockAuthService) SendCode(ctx context.Context, email, username, password string) error {
	return m.err
}

func (m *mockAuthService) VerifyAndRegister(ctx context.Context, email, code string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockAuthService) LoginUser(ctx context.Context, email, password string) (*model.User, error) {
	return m.user, m.err
}

func (m *mockAuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockAuthService) SaveSession(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error {
	if m.activeTokens == nil {
		m.activeTokens = make(map[string]bool)
	}
	m.activeTokens[tokenID] = true
	return nil
}

func (m *mockAuthService) ValidateAndRevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error {
	if m.activeTokens != nil && m.activeTokens[tokenID] {
		delete(m.activeTokens, tokenID)
		return nil
	}
	return model.ErrSessionExpired
}

func (m *mockAuthService) RevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error {
	if m.activeTokens != nil {
		delete(m.activeTokens, tokenID)
	}
	return nil
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	refreshTTL := 7 * 24 * time.Hour
	tokenManager := jwt.NewTokenManager(secret, 15*time.Minute, refreshTTL)

	userID := uuid.New()
	user := &model.User{
		ID:        userID,
		Email:     "player@bito.local",
		Name:      "Player777",
		CreatedAt: time.Now(),
	}

	authService := newMockAuthService(user)
	handler := NewAuthHandler(authService, tokenManager, refreshTTL)

	// 1. Generate valid refresh token and save its session
	pair, err := tokenManager.GenerateTokenPair(userID, user.Name)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}
	_ = authService.SaveSession(context.Background(), userID, pair.TokenID, refreshTTL)

	// Case 1: Success with JSON body
	body, _ := json.Marshal(RefreshRequest{RefreshToken: pair.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Errorf("expected both access and refresh tokens, got %+v", resp)
	}
	if resp.User.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, resp.User.Email)
	}

	// Case 2: Success with Cookie on rotated token
	reqCookie := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	reqCookie.AddCookie(&http.Cookie{
		Name:  "refreshToken",
		Value: resp.RefreshToken, // use newly issued token
	})
	wCookie := httptest.NewRecorder()

	handler.RefreshToken(wCookie, reqCookie)

	if wCookie.Code != http.StatusOK {
		t.Fatalf("expected status 200 with cookie, got %d: %s", wCookie.Code, wCookie.Body.String())
	}

	// Case 3: Replay attack (using old revoked token) -> 401 Unauthorized
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	wReplay := httptest.NewRecorder()
	handler.RefreshToken(wReplay, reqReplay)
	if wReplay.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for replay attack with revoked token, got %d", wReplay.Code)
	}

	// Case 4: Missing token -> 400 Bad Request
	reqMissing := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	wMissing := httptest.NewRecorder()
	handler.RefreshToken(wMissing, reqMissing)
	if wMissing.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing token, got %d", wMissing.Code)
	}

	// Case 5: Invalid token -> 401 Unauthorized
	bodyInvalid, _ := json.Marshal(RefreshRequest{RefreshToken: "invalid-token-string"})
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyInvalid))
	wInvalid := httptest.NewRecorder()
	handler.RefreshToken(wInvalid, reqInvalid)
	if wInvalid.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for invalid token, got %d", wInvalid.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	secret := "test-secret-key-super-secure-12345"
	refreshTTL := 7 * 24 * time.Hour
	tokenManager := jwt.NewTokenManager(secret, 15*time.Minute, refreshTTL)

	userID := uuid.New()
	user := &model.User{
		ID:        userID,
		Email:     "player@bito.local",
		Name:      "Player777",
		CreatedAt: time.Now(),
	}

	authService := newMockAuthService(user)
	handler := NewAuthHandler(authService, tokenManager, refreshTTL)

	pair, _ := tokenManager.GenerateTokenPair(userID, user.Name)
	_ = authService.SaveSession(context.Background(), userID, pair.TokenID, refreshTTL)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refreshToken",
		Value: pair.RefreshToken,
	})
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify token was revoked
	if authService.activeTokens[pair.TokenID] {
		t.Errorf("token should have been revoked from active sessions")
	}
}
