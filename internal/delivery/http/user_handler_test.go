package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/model"
	"bito/internal/pkg/jwt"
	"bito/internal/service"

	"github.com/google/uuid"
)

type memUserRepo struct {
	users map[uuid.UUID]*model.User
}

func (m *memUserRepo) CreateUser(ctx context.Context, user *model.User, stats *model.UserStats, wallet *model.UserWallet) error {
	return nil
}

func (m *memUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, model.ErrUserNotFound
}

func (m *memUserRepo) Update(ctx context.Context, user *model.User) error {
	for id, u := range m.users {
		if id != user.ID && strings.EqualFold(u.Name, user.Name) {
			return model.ErrUserAlreadyExists
		}
	}
	stored, ok := m.users[user.ID]
	if !ok {
		return model.ErrUserNotFound
	}
	stored.Name = user.Name
	stored.PasswordHash = user.PasswordHash
	return nil
}

func (m *memUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.users[id]; !ok {
		return model.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

const testPassword = "correct-horse-1"

func newTestUsersRouter(t *testing.T) (http.Handler, string, *model.User) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)

	hash, err := service.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	alice := &model.User{
		ID:           uuid.New(),
		Email:        "alice@bito.local",
		Name:         "Alice",
		PasswordHash: hash,
		CreatedAt:    time.Now(),
		Stats:        &model.UserStats{Elo: 1042, Wins: 3, Losses: 1, TotalGames: 4},
		Wallet:       &model.UserWallet{Balance: 2500},
	}
	bob := &model.User{ID: uuid.New(), Name: "Bob", CreatedAt: time.Now()}
	repo := &memUserRepo{users: map[uuid.UUID]*model.User{alice.ID: alice, bob.ID: bob}}

	hub := ws.NewHub(tokenManager, nil)
	server := NewServer(":0", hub, nil, nil, NewUserHandler(service.NewUserService(repo)), tokenManager)

	token, err := tokenManager.GenerateAccessToken(alice.ID, alice.Name)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return server.httpServer.Handler, token, alice
}

func doUsersRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUsers_RequireAuth(t *testing.T) {
	router, _, _ := newTestUsersRouter(t)

	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodPatch, "/api/v1/users/me"},
		{http.MethodDelete, "/api/v1/users/me"},
		{http.MethodPost, "/api/v1/users/me/password"},
	}
	for _, tc := range cases {
		if w := doUsersRequest(router, tc.method, tc.path, "", "{}"); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: status = %d, want 401", tc.method, tc.path, w.Code)
		}
	}
}

func TestUsers_Profile(t *testing.T) {
	router, token, alice := newTestUsersRouter(t)

	w := doUsersRequest(router, http.MethodGet, "/api/v1/users/me", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "password") {
		t.Errorf("profile must not expose password data: %s", w.Body.String())
	}

	var got ProfileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode profile: %v", err)
	}
	if got.ID != alice.ID || got.Username != "Alice" || got.Email != alice.Email {
		t.Errorf("unexpected user fields: %+v", got.UserResponse)
	}
	if got.Stats != (StatsResponse{Elo: 1042, Wins: 3, Losses: 1, TotalGames: 4}) {
		t.Errorf("stats = %+v", got.Stats)
	}
	if got.Wallet.Balance != 2500 {
		t.Errorf("balance = %d, want 2500", got.Wallet.Balance)
	}
}

func TestUsers_ChangeUsername(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"cyrillic 22 chars", `{"username":"Владислав Великолепный"}`, http.StatusOK},
		{"too short", `{"username":"Al"}`, http.StatusBadRequest},
		{"taken", `{"username":"bob"}`, http.StatusConflict},
		{"id in body is rejected", `{"username":"Alice2","id":"x"}`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, token, _ := newTestUsersRouter(t)
			w := doUsersRequest(router, http.MethodPatch, "/api/v1/users/me", token, tc.body)
			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestUsers_ChangePassword(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"wrong old password", `{"oldPassword":"nope-nope-1","newPassword":"brand-new-pass"}`, http.StatusForbidden},
		{"too short", `{"oldPassword":"` + testPassword + `","newPassword":"short"}`, http.StatusBadRequest},
		{"same password", `{"oldPassword":"` + testPassword + `","newPassword":"` + testPassword + `"}`, http.StatusBadRequest},
		{"ok", `{"oldPassword":"` + testPassword + `","newPassword":"brand-new-pass"}`, http.StatusNoContent},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, token, _ := newTestUsersRouter(t)
			w := doUsersRequest(router, http.MethodPost, "/api/v1/users/me/password", token, tc.body)
			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestUsers_DeleteAccount(t *testing.T) {
	router, token, _ := newTestUsersRouter(t)

	if w := doUsersRequest(router, http.MethodDelete, "/api/v1/users/me", token, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", w.Code)
	}
	if w := doUsersRequest(router, http.MethodGet, "/api/v1/users/me", token, ""); w.Code != http.StatusNotFound {
		t.Fatalf("profile after delete status = %d, want 404", w.Code)
	}
}
