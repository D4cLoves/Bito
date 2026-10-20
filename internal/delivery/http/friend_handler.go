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

type FriendService interface {
	Overview(ctx context.Context, me uuid.UUID) (model.FriendsOverview, error)
	Request(ctx context.Context, me uuid.UUID, username string) (model.FriendState, error)
	Accept(ctx context.Context, me, from uuid.UUID) error
	Remove(ctx context.Context, me, other uuid.UUID) error
}

type FriendResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Status   string `json:"status"`
	Since    string `json:"since"`
}

type FriendRequestResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	At       string `json:"at"`
}

type FriendsResponse struct {
	Friends  []FriendResponse        `json:"friends"`
	Incoming []FriendRequestResponse `json:"incoming"`
	Outgoing []FriendRequestResponse `json:"outgoing"`
}

type AddFriendRequest struct {
	Username string `json:"username"`
}

type AddFriendResponse struct {
	Status string `json:"status"`
}

type FriendHandler struct {
	friends FriendService
}

func NewFriendHandler(friends FriendService) *FriendHandler {
	return &FriendHandler{friends: friends}
}

func (fh *FriendHandler) List(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	overview, err := fh.friends.Overview(r.Context(), me)
	if err != nil {
		respondFriendError(w, me, err)
		return
	}
	respondJSON(w, http.StatusOK, toFriendsResponse(overview))
}

func (fh *FriendHandler) Request(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddFriendRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	state, err := fh.friends.Request(r.Context(), me, req.Username)
	if err != nil {
		respondFriendError(w, me, err)
		return
	}

	status := http.StatusCreated
	if state == model.FriendAccepted {
		status = http.StatusOK
	}
	respondJSON(w, status, AddFriendResponse{Status: string(state)})
}

func (fh *FriendHandler) Accept(w http.ResponseWriter, r *http.Request) {
	me, other, ok := fh.pair(w, r)
	if !ok {
		return
	}
	if err := fh.friends.Accept(r.Context(), me, other); err != nil {
		respondFriendError(w, me, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (fh *FriendHandler) Remove(w http.ResponseWriter, r *http.Request) {
	me, other, ok := fh.pair(w, r)
	if !ok {
		return
	}
	if err := fh.friends.Remove(r.Context(), me, other); err != nil {
		respondFriendError(w, me, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (fh *FriendHandler) pair(w http.ResponseWriter, r *http.Request) (me, other uuid.UUID, ok bool) {
	me, ok = UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	other, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный id игрока")
		return uuid.Nil, uuid.Nil, false
	}
	return me, other, true
}

func respondFriendError(w http.ResponseWriter, me uuid.UUID, err error) {
	switch {
	case errors.Is(err, model.ErrUserNotFound):
		respondError(w, http.StatusNotFound, "игрок с таким именем не найден")
	case errors.Is(err, model.ErrFriendSelf):
		respondError(w, http.StatusBadRequest, "нельзя добавить в друзья самого себя")
	case errors.Is(err, model.ErrAlreadyFriends):
		respondError(w, http.StatusConflict, "вы уже друзья")
	case errors.Is(err, model.ErrFriendRequestExists):
		respondError(w, http.StatusConflict, "заявка уже отправлена")
	case errors.Is(err, model.ErrFriendRequestNotFound):
		respondError(w, http.StatusNotFound, "заявка не найдена")
	default:
		log.Printf("friends: %s: %v", me, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

func toFriendsResponse(o model.FriendsOverview) FriendsResponse {
	resp := FriendsResponse{
		Friends:  make([]FriendResponse, 0, len(o.Friends)),
		Incoming: make([]FriendRequestResponse, 0, len(o.Incoming)),
		Outgoing: make([]FriendRequestResponse, 0, len(o.Outgoing)),
	}
	for _, f := range o.Friends {
		resp.Friends = append(resp.Friends, FriendResponse{
			ID:       f.UserID.String(),
			Username: f.Username,
			Status:   string(f.Presence),
			Since:    f.Since.Format(time.RFC3339),
		})
	}
	for _, l := range o.Incoming {
		resp.Incoming = append(resp.Incoming, toFriendRequestResponse(l))
	}
	for _, l := range o.Outgoing {
		resp.Outgoing = append(resp.Outgoing, toFriendRequestResponse(l))
	}
	return resp
}

func toFriendRequestResponse(l model.FriendLink) FriendRequestResponse {
	return FriendRequestResponse{ID: l.UserID.String(), Username: l.Username, At: l.Since.Format(time.RFC3339)}
}
