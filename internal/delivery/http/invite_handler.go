package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"bito/internal/model"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type InviteService interface {
	Invite(ctx context.Context, me uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error)
	Decline(ctx context.Context, me uuid.UUID, roomID string) error
}

type InviteRequest struct {
	UserID string `json:"userId"`
}

type InviteResponse struct {
	TableID   string `json:"tableId"`
	ExpiresAt string `json:"expiresAt"`
}

type InviteHandler struct {
	invites InviteService
}

func NewInviteHandler(invites InviteService) *InviteHandler {
	return &InviteHandler{invites: invites}
}

func (ih *InviteHandler) Invite(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req InviteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	friend, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный id игрока")
		return
	}

	inv, err := ih.invites.Invite(r.Context(), me, chi.URLParam(r, "id"), friend)
	if err != nil {
		respondInviteError(w, me, err)
		return
	}
	respondJSON(w, http.StatusCreated, InviteResponse{TableID: inv.Table.ID, ExpiresAt: inv.ExpiresAt.Format(time.RFC3339)})
}

func (ih *InviteHandler) Decline(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := ih.invites.Decline(r.Context(), me, chi.URLParam(r, "id")); err != nil {
		respondInviteError(w, me, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respondInviteError(w http.ResponseWriter, me uuid.UUID, err error) {
	switch {
	case errors.Is(err, model.ErrTableNotFound):
		respondError(w, http.StatusNotFound, "стол уже закрыт")
	case errors.Is(err, model.ErrInviteNotFound):
		respondError(w, http.StatusNotFound, "приглашение уже недействительно")
	case errors.Is(err, model.ErrNotFriends):
		respondError(w, http.StatusForbidden, "звать за стол можно только друзей")
	case errors.Is(err, model.ErrNotAtTable):
		respondError(w, http.StatusForbidden, "звать можно только за свой стол")
	case errors.Is(err, model.ErrTableStarted):
		respondError(w, http.StatusConflict, "партия уже началась")
	case errors.Is(err, model.ErrTableFull):
		respondError(w, http.StatusConflict, "за столом нет свободных мест")
	case errors.Is(err, model.ErrAlreadyInvited):
		respondError(w, http.StatusConflict, "друг уже приглашён")
	case errors.Is(err, model.ErrFriendBusy):
		respondError(w, http.StatusConflict, "друг сейчас за другим столом")
	case errors.Is(err, model.ErrFriendNotInLobby):
		respondError(w, http.StatusConflict, "друг сейчас не в лобби")
	case errors.Is(err, model.ErrFriendNoFunds):
		respondError(w, http.StatusConflict, "у друга не хватает фишек на эту ставку")
	default:
		log.Printf("invites: %s: %v", me, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
