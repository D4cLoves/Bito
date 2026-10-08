package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bito/internal/model"
	"bito/internal/pkg/jwt"
	"bito/internal/service"

	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20

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

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (ah *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !decodeJSON(w, r, &req) {
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
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
	})

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// decodeJSON limits the body size and rejects unknown fields; on failure it writes 400 and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (ah *AuthHandler) writeSession(w http.ResponseWriter, status int, s *service.Session) {
	ah.setTokenCookies(w, s.Tokens)
	respondJSON(w, status, AuthResponse{
		AccessToken:  s.Tokens.AccessToken,
		RefreshToken: s.Tokens.RefreshToken,
		User: UserResponse{
			ID:        s.User.ID,
			Email:     s.User.Email,
			Username:  s.User.Name,
			CreatedAt: s.User.CreatedAt.Format(time.RFC3339),
		},
	})
}

func (ah *AuthHandler) setTokenCookies(w http.ResponseWriter, tokens *jwt.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    tokens.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}
