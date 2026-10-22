package http

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

func TestAuthMiddleware(t *testing.T) {
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	userID := uuid.New()

	accessToken, err := tokenManager.GenerateAccessToken(userID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	refreshToken, _, err := tokenManager.GenerateRefreshToken(userID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	var gotID uuid.UUID
	var gotOK bool
	handler := AuthMiddleware(tokenManager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, gotOK = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name     string
		setup    func(r *http.Request)
		wantCode int
	}{
		{"no token", func(r *http.Request) {}, http.StatusUnauthorized},
		{"garbage bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer garbage") }, http.StatusUnauthorized},
		{"refresh token as bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+refreshToken) }, http.StatusUnauthorized},
		{"valid bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+accessToken) }, http.StatusOK},
		{"valid cookie", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "accessToken", Value: accessToken}) }, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotOK = uuid.Nil, false
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tc.setup(req)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantCode)
			}
			if tc.wantCode == http.StatusOK && (!gotOK || gotID != userID) {
				t.Errorf("context user = (%v, %v), want (%v, true)", gotID, gotOK, userID)
			}
			if tc.wantCode != http.StatusOK && gotOK {
				t.Error("next handler must not run on rejected request")
			}
		})
	}
}

func TestRedactingLogger_DropsTokenFromLog(t *testing.T) {
	var out bytes.Buffer
	var seenToken string
	handler := RedactingLogger(log.New(&out, "", 0))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenToken = r.URL.Query().Get("token")
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws?room_id=abc&token=secret-jwt", nil))

	if seenToken != "secret-jwt" {
		t.Errorf("handler got token %q, want the original query untouched", seenToken)
	}
	logged := out.String()
	if strings.Contains(logged, "secret-jwt") || strings.Contains(logged, "token=") {
		t.Errorf("token leaked into the log: %s", logged)
	}
	if !strings.Contains(logged, "/ws?room_id=abc") {
		t.Errorf("log lost the rest of the address: %s", logged)
	}
}

func TestRedactingLogger_DropsTableKeyFromLog(t *testing.T) {
	var out bytes.Buffer
	handler := RedactingLogger(log.New(&out, "", 0))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws?room_id=abc&key=secret-table-key", nil))
	if strings.Contains(out.String(), "secret-table-key") {
		t.Errorf("table key leaked into the log: %s", out.String())
	}
}
