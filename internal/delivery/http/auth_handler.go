package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bito/internal/model"
	"bito/internal/pkg/jwt"
	"bito/internal/service"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type VerifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt string    `json:"createdAt"`
}

type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type AuthService interface {
	SendCode(ctx context.Context, email, username, password string) error
	VerifyAndRegister(ctx context.Context, email, code string) (*service.Session, error)
	Login(ctx context.Context, email, password string) (*service.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*service.Session, error)
	Logout(ctx context.Context, refreshToken string) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error)
}

const (
	authLimitWindow   = 15 * time.Minute
	authLimitPerEmail = 10
	authLimitPerIP    = 50
)

type CookieConfig struct {
	Secure     bool
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type AuthHandler struct {
	authService AuthService
	limiter     RateLimiter
	cookies     CookieConfig
}

func NewAuthHandler(authService AuthService, limiter RateLimiter, cookies CookieConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		limiter:     limiter,
		cookies:     cookies,
	}
}

func (ah *AuthHandler) allow(w http.ResponseWriter, r *http.Request, action, email string) bool {
	if ah.limiter == nil {
		return true
	}

	type rule struct {
		key   string
		limit int
	}
	rules := []rule{{action + ":ip:" + clientIP(r), authLimitPerIP}}
	if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
		rules = append(rules, rule{action + ":email:" + email, authLimitPerEmail})
	}

	for _, rl := range rules {
		ok, retryAfter, err := ah.limiter.Allow(r.Context(), rl.key, rl.limit, authLimitWindow)
		if err != nil {
			log.Printf("auth: rate limit %s: %v", action, err)
			continue
		}
		if !ok {
			respondTooManyAttempts(w, retryAfter)
			return false
		}
	}
	return true
}

func respondTooManyAttempts(w http.ResponseWriter, retryAfter time.Duration) {
	minutes := max(int(math.Ceil(retryAfter.Minutes())), 1)
	w.Header().Set("Retry-After", strconv.Itoa(minutes*60))
	respondError(w, http.StatusTooManyRequests, fmt.Sprintf("Слишком много попыток. Попробуйте снова через %d мин.", minutes))
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (ah *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !ah.allow(w, r, "send-code", req.Email) {
		return
	}

	if err := ah.authService.SendCode(r.Context(), req.Email, req.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, model.ErrUserAlreadyExists):
			respondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrInvalidEmail),
			errors.Is(err, service.ErrInvalidUsername),
			errors.Is(err, service.ErrPasswordTooShort),
			errors.Is(err, service.ErrPasswordTooLong):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to send verification code")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "verification code sent",
	})
}

func (ah *AuthHandler) VerifyCode(w http.ResponseWriter, r *http.Request) {
	var req VerifyCodeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	session, err := ah.authService.VerifyAndRegister(r.Context(), req.Email, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserAlreadyExists):
			respondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, model.ErrPendingNotFound):
			respondError(w, http.StatusGone, "код подтверждения истек или не найден")
		case errors.Is(err, model.ErrCodeAttempts):
			respondError(w, http.StatusTooManyRequests, "Слишком много неверных попыток. Код больше не действует, запросите новый.")
		case errors.Is(err, model.ErrCodeMismatch), errors.Is(err, service.ErrInvalidCode):
			respondError(w, http.StatusBadRequest, "неверный проверочный код")
		case errors.Is(err, service.ErrInvalidEmail):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to verify code and register")
		}
		return
	}

	ah.writeSession(w, http.StatusCreated, session)
}

func (ah *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !ah.allow(w, r, "login", req.Email) {
		return
	}

	session, err := ah.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials),
			errors.Is(err, model.ErrUserNotFound):
			respondError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	ah.writeSession(w, http.StatusOK, session)
}

func (ah *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshToken string

	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		if cookie, err := r.Cookie("refreshToken"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh token is required")
		return
	}

	session, err := ah.authService.Refresh(r.Context(), refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrExpiredToken):
			respondError(w, http.StatusUnauthorized, "refresh token has expired")
		case errors.Is(err, jwt.ErrInvalidToken):
			respondError(w, http.StatusUnauthorized, "invalid refresh token")
		case errors.Is(err, model.ErrSessionExpired):
			respondError(w, http.StatusUnauthorized, "session expired or revoked")
		case errors.Is(err, model.ErrUserNotFound):
			respondError(w, http.StatusUnauthorized, "user not found or deactivated")
		default:
			respondError(w, http.StatusInternalServerError, "failed to refresh session")
		}
		return
	}

	ah.writeSession(w, http.StatusOK, session)
}

func (ah *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refreshToken"); err == nil && cookie.Value != "" {
		_ = ah.authService.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   ah.cookies.Secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   ah.cookies.Secure,
	})

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

func (ah *AuthHandler) writeSession(w http.ResponseWriter, status int, s *service.Session) {
	ah.setTokenCookies(w, s.Tokens)
	respondJSON(w, status, AuthResponse{
		AccessToken:  s.Tokens.AccessToken,
		RefreshToken: s.Tokens.RefreshToken,
		User:         toUserResponse(s.User),
	})
}

func (ah *AuthHandler) setTokenCookies(w http.ResponseWriter, tokens *jwt.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    tokens.AccessToken,
		Path:     "/",
		MaxAge:   int(ah.cookies.AccessTTL.Seconds()),
		HttpOnly: true,
		Secure:   ah.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    tokens.RefreshToken,
		Path:     "/api/v1/auth",
		MaxAge:   int(ah.cookies.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   ah.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
