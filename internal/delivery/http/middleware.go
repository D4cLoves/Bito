package http

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"bito/internal/pkg/jwt"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func RedactingLogger(logger middleware.LoggerInterface) func(http.Handler) http.Handler {
	return middleware.RequestLogger(redactingFormatter{
		next: &middleware.DefaultLogFormatter{Logger: logger, NoColor: runtime.GOOS == "windows"},
	})
}

type redactingFormatter struct {
	next middleware.LogFormatter
}

func (f redactingFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	query := r.URL.Query()
	if !query.Has("token") && !query.Has("key") {
		return f.next.NewLogEntry(r)
	}
	query.Del("token")
	query.Del("key")
	safeURL := *r.URL
	safeURL.RawQuery = query.Encode()

	safe := r.WithContext(r.Context())
	safe.URL = &safeURL
	safe.RequestURI = safeURL.RequestURI()
	return f.next.NewLogEntry(safe)
}

type TokenValidator interface {
	ValidateAccessToken(token string) (*jwt.Claims, error)
}

type ctxKey struct{}

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

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return id, ok
}

func accessTokenFromRequest(r *http.Request) string {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && token != "" {
		return token
	}
	if cookie, err := r.Cookie("accessToken"); err == nil {
		return cookie.Value
	}
	return ""
}
