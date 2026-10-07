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
	VerifyAndRegister(ctx context.Context, email, code string) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	SaveSession(ctx context.Context, userID uuid.UUID, tokenID string, ttl time.Duration) error
	ValidateAndRevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error
	RevokeSession(ctx context.Context, userID uuid.UUID, tokenID string) error
}

type TokenProvider interface {
	GenerateTokenPair(userID uuid.UUID, username string) (*jwt.TokenPair, error)
	ValidateRefreshToken(tokenString string) (*jwt.Claims, error)
}

type AuthHandler struct {
	authService   AuthService
	tokenProvider TokenProvider
	refreshTTL    time.Duration
}

func NewAuthHandler(authService AuthService, tokenProvider TokenProvider, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		tokenProvider: tokenProvider,
		refreshTTL:    refreshTTL,
	}
}

// SendCode - Шаг 1 регистрации: валидация, сохранение черновика в Redis и отправка кода на почту
func (ah *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
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

// VerifyCode - Шаг 2 регистрации: проверка кода из Redis, создание профиля и выдача токенов с сохранением в Redis
func (ah *AuthHandler) VerifyCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req VerifyCodeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := ah.authService.VerifyAndRegister(r.Context(), req.Email, req.Code)
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

	tokens, err := ah.tokenProvider.GenerateTokenPair(user.ID, user.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Сохраняем сессию в Redis
	if err := ah.authService.SaveSession(r.Context(), user.ID, tokens.TokenID, ah.refreshTTL); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save session")
		return
	}

	ah.setTokenCookies(w, tokens)

	respondJSON(w, http.StatusCreated, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}

// Login - Вход по email и паролю с выдачей пары токенов и записью сессии в Redis
func (ah *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := ah.authService.LoginUser(r.Context(), req.Email, req.Password)
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

	tokens, err := ah.tokenProvider.GenerateTokenPair(user.ID, user.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Сохраняем сессию в Redis
	if err := ah.authService.SaveSession(r.Context(), user.ID, tokens.TokenID, ah.refreshTTL); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save session")
		return
	}

	ah.setTokenCookies(w, tokens)

	respondJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}

// RefreshToken - Ротация токенов с проверкой сессии в Redis (Token Rotation)
func (ah *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshToken string

	// 1. Пытаемся прочитать из тела JSON
	if r.Body != nil {
		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	// 2. Если в теле нет, берем из HttpOnly Cookie
	if refreshToken == "" {
		if cookie, err := r.Cookie("refreshToken"); err == nil && cookie.Value != "" {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh token is required")
		return
	}

	claims, err := ah.tokenProvider.ValidateRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			respondError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Проверяем сессию в Redis и удаляем старый токен (строгая одноразовая ротация)
	if err := ah.authService.ValidateAndRevokeSession(r.Context(), claims.UserID, claims.ID); err != nil {
		if errors.Is(err, model.ErrSessionExpired) {
			respondError(w, http.StatusUnauthorized, "session expired or revoked")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to validate session")
		return
	}

	user, err := ah.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "user not found or deactivated")
		return
	}

	// Token Rotation: генерируем новую пару токенов
	tokens, err := ah.tokenProvider.GenerateTokenPair(user.ID, user.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Записываем новую сессию в Redis
	if err := ah.authService.SaveSession(r.Context(), user.ID, tokens.TokenID, ah.refreshTTL); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save session")
		return
	}

	ah.setTokenCookies(w, tokens)

	respondJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}

// Logout - Завершение сессии и удаление токена из Redis
func (ah *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var refreshToken string
	if cookie, err := r.Cookie("refreshToken"); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	}

	if refreshToken != "" {
		if claims, err := ah.tokenProvider.ValidateRefreshToken(refreshToken); err == nil {
			_ = ah.authService.RevokeSession(r.Context(), claims.UserID, claims.ID)
		}
	}

	// Инвалидируем cookies
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
