package http

import (
	"context"
	"net/http"
	"strings"

	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

type TokenValidator interface {
	ValidateAccessToken(token string) (*jwt.Claims, error)
}

// ctxKey is unexported so no other package can read or overwrite the value.
type ctxKey struct{}

// AuthMiddleware rejects requests without a valid access token and puts the user ID into the request context.
func AuthMiddleware(tokens TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := accessTokenFromRequest(r)
			if token == "" {
				respondError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims, err := tokens.ValidateAccessToken(token)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the user ID stored by AuthMiddleware.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return id, ok
}

// accessTokenFromRequest prefers "Authorization: Bearer <token>" and falls back to the accessToken cookie.
func accessTokenFromRequest(r *http.Request) string {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && token != "" {
		return token
	}
	if cookie, err := r.Cookie("accessToken"); err == nil {
		return cookie.Value
	}
	return ""
}
