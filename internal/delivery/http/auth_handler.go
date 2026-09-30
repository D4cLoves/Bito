package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"bito/internal/model"
	"bito/internal/service"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	AccessToken string       `json:"accessToken"`
	User        UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type AuthService interface {
	RegisterUser(ctx context.Context, email, username, password string) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (*model.User, error)
}

type TokenProvider interface {
	GenerateAccessToken(userID uuid.UUID, username string) (string, error)
}

type AuthHandler struct {
	authService   AuthService
	tokenProvider TokenProvider
}

func NewAuthHandler(authService AuthService, tokenProvider TokenProvider) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		tokenProvider: tokenProvider,
	}
}

func (ah *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := ah.authService.RegisterUser(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserAlreadyExists):
			respondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrInvalidEmail),
			errors.Is(err, service.ErrInvalidUsername),
			errors.Is(err, service.ErrPasswordTooShort),
			errors.Is(err, service.ErrPasswordTooLong):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	token, err := ah.tokenProvider.GenerateAccessToken(user.ID, user.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	respondJSON(w, http.StatusCreated, AuthResponse{
		AccessToken: token,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}

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

	token, err := ah.tokenProvider.GenerateAccessToken(user.ID, user.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	respondJSON(w, http.StatusOK, AuthResponse{
		AccessToken: token,
		User: UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	})
}
