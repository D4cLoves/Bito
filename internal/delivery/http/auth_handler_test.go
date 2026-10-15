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
	"bito/internal/service"

	"github.com/google/uuid"
)

// fakeUserRepo serves a single user; only GetByID is exercised by refresh.
type fakeUserRepo struct {
	user *model.User
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, user *model.User, stats *model.UserStats, wallet *model.UserWallet) error {
	return nil
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, model.ErrUserNotFound
	}
	u := *f.user
	return &u, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, model.ErrUserNotFound
}

func (f *fakeUserRepo) Update(ctx context.Context, user *model.User) error { return nil }

func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

// fakeRedis keeps refresh sessions in memory, mirroring the Redis repository semantics.
type fakeRedis struct {
	activeTokens map[string]bool
	verifyErr    error
}

func (f *fakeRedis) SavePending(ctx context.Context, data *model.PendingRegistration, ttl time.Duration) error {
	return nil
}

func (f *fakeRedis) VerifyAndGetPending(ctx context.Context, email, code string) (*model.PendingRegistration, error) {
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	return nil, model.ErrPendingNotFound
}

func (f *fakeRedis) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error {
	f.activeTokens[tokenID] = true
	return nil
}

func (f *fakeRedis) ValidateAndRevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error) {
	if !f.activeTokens[tokenID] {
		return false, nil
	}
	delete(f.activeTokens, tokenID)
	return true, nil
}

func (f *fakeRedis) RevokeRefreshToken(ctx context.Context, userID uuid.UUID, tokenID string) error {
	delete(f.activeTokens, tokenID)
	return nil
}

func newTestAuthHandler(user *model.User, tokenManager *jwt.TokenManager, refreshTTL time.Duration) (*AuthHandler, *fakeRedis) {
	redis := &fakeRedis{activeTokens: make(map[string]bool)}
	authService := service.NewAuthService(&fakeUserRepo{user: user}, nil, nil, redis, tokenManager, refreshTTL)
	return NewAuthHandler(authService, nil, CookieConfig{AccessTTL: 15 * time.Minute, RefreshTTL: refreshTTL}), redis
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

	handler, redis := newTestAuthHandler(user, tokenManager, refreshTTL)

	// 1. Generate valid refresh token and save its session
	pair, err := tokenManager.GenerateTokenPair(userID, user.Name)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}
	_ = redis.SaveRefreshToken(context.Background(), userID, pair.TokenID, refreshTTL)

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

	handler, redis := newTestAuthHandler(user, tokenManager, refreshTTL)

	pair, _ := tokenManager.GenerateTokenPair(userID, user.Name)
	_ = redis.SaveRefreshToken(context.Background(), userID, pair.TokenID, refreshTTL)

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
	if redis.activeTokens[pair.TokenID] {
		t.Errorf("token should have been revoked from active sessions")
	}
}

type memoryLimiter struct {
	hits map[string]int
}

func (m *memoryLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	m.hits[key]++
	if m.hits[key] > limit {
		return false, window, nil
	}
	return true, 0, nil
}

func TestAuthHandler_LoginRateLimit(t *testing.T) {
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	authService := service.NewAuthService(&fakeUserRepo{}, nil, nil, &fakeRedis{activeTokens: map[string]bool{}}, tokenManager, time.Hour)
	limiter := &memoryLimiter{hits: map[string]int{}}
	handler := NewAuthHandler(authService, limiter, CookieConfig{})

	login := func(email, ip string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(LoginRequest{Email: email, Password: "wrong-password"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.RemoteAddr = ip + ":40000"
		w := httptest.NewRecorder()
		handler.Login(w, req)
		return w
	}

	for i := range authLimitPerEmail {
		if w := login("Victim@bito.local", "10.0.0.1"); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, w.Code)
		}
	}

	w := login("victim@bito.local", "10.0.0.2")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt over the email limit: status = %d, want 429", w.Code)
	}
	var resp ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != "Слишком много попыток. Попробуйте снова через 15 мин." {
		t.Errorf("message = %q", resp.Error)
	}
	if w.Header().Get("Retry-After") != "900" {
		t.Errorf("Retry-After = %q, want 900", w.Header().Get("Retry-After"))
	}

	if w := login("other@bito.local", "10.0.0.1"); w.Code != http.StatusUnauthorized {
		t.Errorf("another email from the same address: status = %d, want 401", w.Code)
	}

	for range authLimitPerIP {
		login("spray-"+uuid.NewString()+"@bito.local", "10.0.0.3")
	}
	if w := login("fresh@bito.local", "10.0.0.3"); w.Code != http.StatusTooManyRequests {
		t.Errorf("attempt over the address limit: status = %d, want 429", w.Code)
	}
}

func TestAuthHandler_VerifyCodeAttemptsExhausted(t *testing.T) {
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	redis := &fakeRedis{activeTokens: map[string]bool{}, verifyErr: model.ErrCodeAttempts}
	handler := NewAuthHandler(service.NewAuthService(&fakeUserRepo{}, nil, nil, redis, tokenManager, time.Hour), nil, CookieConfig{})

	body, _ := json.Marshal(VerifyCodeRequest{Email: "player@bito.local", Code: "123456"})
	w := httptest.NewRecorder()
	handler.VerifyCode(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-code", bytes.NewReader(body)))

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (%s)", w.Code, w.Body.String())
	}
}

func TestAuthHandler_CookiesOutliveBrowserSession(t *testing.T) {
	refreshTTL := 7 * 24 * time.Hour
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, refreshTTL)
	user := &model.User{ID: uuid.New(), Email: "player@bito.local", Name: "Player777", CreatedAt: time.Now()}
	handler, redis := newTestAuthHandler(user, tokenManager, refreshTTL)

	pair, _ := tokenManager.GenerateTokenPair(user.ID, user.Name)
	_ = redis.SaveRefreshToken(context.Background(), user.ID, pair.TokenID, refreshTTL)
	body, _ := json.Marshal(RefreshRequest{RefreshToken: pair.RefreshToken})
	w := httptest.NewRecorder()
	handler.RefreshToken(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body)))

	want := map[string]int{"accessToken": 15 * 60, "refreshToken": int(refreshTTL.Seconds())}
	for _, c := range w.Result().Cookies() {
		if c.MaxAge != want[c.Name] {
			t.Errorf("%s MaxAge = %d, want %d", c.Name, c.MaxAge, want[c.Name])
		}
		delete(want, c.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing cookies: %v", want)
	}
}
