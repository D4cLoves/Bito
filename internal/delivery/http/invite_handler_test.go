package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/model"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

type stubInvites struct {
	err      error
	gotRoom  string
	gotGuest uuid.UUID
}

func (s *stubInvites) Invite(_ context.Context, _ uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error) {
	s.gotRoom, s.gotGuest = roomID, friend
	if s.err != nil {
		return model.TableInvite{}, s.err
	}
	return model.TableInvite{Table: model.InviteTable{ID: roomID}, ExpiresAt: time.Now()}, nil
}

func (s *stubInvites) Decline(_ context.Context, _ uuid.UUID, roomID string) error {
	s.gotRoom = roomID
	return s.err
}

func newInvitesRouter(t *testing.T, stub *stubInvites) (http.Handler, string) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	hub := ws.NewHub(tokenManager, nil, nil)
	server := NewServer(":0", hub, Handlers{Table: NewTableHandler(hub, nil), Invite: NewInviteHandler(stub)}, tokenManager)
	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	return server.httpServer.Handler, token
}

func TestInvites_PassTableAndGuest(t *testing.T) {
	stub := &stubInvites{}
	router, token := newInvitesRouter(t, stub)
	bob := uuid.New()

	w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"`+bob.String()+`"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	if stub.gotRoom != "t1" || stub.gotGuest != bob {
		t.Fatalf("service got %q, %s", stub.gotRoom, stub.gotGuest)
	}
	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"nope"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("bad user id status = %d, want 400", w.Code)
	}
}

func TestInvites_ErrorStatuses(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{model.ErrTableNotFound, http.StatusNotFound},
		{model.ErrNotFriends, http.StatusForbidden},
		{model.ErrNotAtTable, http.StatusForbidden},
		{model.ErrTableStarted, http.StatusConflict},
		{model.ErrTableFull, http.StatusConflict},
		{model.ErrAlreadyInvited, http.StatusConflict},
		{model.ErrFriendBusy, http.StatusConflict},
		{model.ErrFriendNotInLobby, http.StatusConflict},
		{model.ErrFriendNoFunds, http.StatusConflict},
	}
	for _, tc := range cases {
		router, token := newInvitesRouter(t, &stubInvites{err: tc.err})
		w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites", token, `{"userId":"`+uuid.NewString()+`"}`)
		if w.Code != tc.want {
			t.Errorf("%v: status = %d, want %d", tc.err, w.Code, tc.want)
		}
	}
}

func TestInvites_Decline(t *testing.T) {
	stub := &stubInvites{}
	router, token := newInvitesRouter(t, stub)
	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites/decline", token, ""); w.Code != http.StatusNoContent || stub.gotRoom != "t1" {
		t.Fatalf("decline status = %d, room = %q", w.Code, stub.gotRoom)
	}
	stub.err = model.ErrInviteNotFound
	if w := doUsersRequest(router, http.MethodPost, "/api/v1/tables/t1/invites/decline", token, ""); w.Code != http.StatusNotFound {
		t.Fatalf("stale decline status = %d, want 404", w.Code)
	}
}
