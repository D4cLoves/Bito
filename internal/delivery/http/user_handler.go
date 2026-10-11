package http

import (
	"context"
	"errors"
	"net/http"

	"bito/internal/model"
	"bito/internal/service"

	"github.com/google/uuid"
)

type StatsResponse struct {
	Elo        int `json:"elo"`
	Wins       int `json:"wins"`
	Losses     int `json:"losses"`
	TotalGames int `json:"totalGames"`
}

type WalletResponse struct {
	Balance int64 `json:"balance"`
}

type ProfileResponse struct {
	UserResponse
	Stats  StatsResponse  `json:"stats"`
	Wallet WalletResponse `json:"wallet"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type UserService interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (*model.User, error)
	UpdateUsername(ctx context.Context, userID uuid.UUID, newUsername string) (*model.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	DeleteAccount(ctx context.Context, userID uuid.UUID) error
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (uh *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := uh.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		respondUserError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toProfileResponse(user))
}

func (uh *UserHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateUsernameRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := uh.userService.UpdateUsername(r.Context(), userID, req.Username)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidUsername):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrUserAlreadyExists):
			respondError(w, http.StatusConflict, "username is already taken")
		default:
			respondUserError(w, err)
		}
		return
	}

	respondJSON(w, http.StatusOK, toUserResponse(user))
}

func (uh *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ChangePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	err := uh.userService.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordTooShort),
			errors.Is(err, service.ErrPasswordTooLong),
			errors.Is(err, service.ErrSamePassword):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			respondError(w, http.StatusForbidden, "old password is incorrect")
		default:
			respondUserError(w, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (uh *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := uh.userService.DeleteAccount(r.Context(), userID); err != nil {
		respondUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondUserError(w http.ResponseWriter, err error) {
	if errors.Is(err, model.ErrUserNotFound) {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respondError(w, http.StatusInternalServerError, "internal server error")
}

func toProfileResponse(u *model.User) ProfileResponse {
	resp := ProfileResponse{UserResponse: toUserResponse(u)}
	if u.Stats != nil {
		resp.Stats = StatsResponse{
			Elo:        u.Stats.Elo,
			Wins:       u.Stats.Wins,
			Losses:     u.Stats.Losses,
			TotalGames: u.Stats.TotalGames,
		}
	}
	if u.Wallet != nil {
		resp.Wallet = WalletResponse{Balance: u.Wallet.Balance}
	}
	return resp
}
