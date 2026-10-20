package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/model"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

type stubFriends struct {
	overview model.FriendsOverview
	state    model.FriendState
	err      error
	lastName string
	lastID   uuid.UUID
}

func (s *stubFriends) Overview(context.Context, uuid.UUID) (model.FriendsOverview, error) {
	return s.overview, s.err
}

func (s *stubFriends) Request(_ context.Context, _ uuid.UUID, username string) (model.FriendState, error) {
	s.lastName = username
	return s.state, s.err
}

func (s *stubFriends) Accept(_ context.Context, _, from uuid.UUID) error {
	s.lastID = from
	return s.err
}

func (s *stubFriends) Remove(_ context.Context, _, other uuid.UUID) error {
	s.lastID = other
	return s.err
}

func newFriendsRouter(t *testing.T, stub *stubFriends) (http.Handler, string) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	hub := ws.NewHub(tokenManager, nil, nil)
	server := NewServer(":0", hub, Handlers{Friend: NewFriendHandler(stub)}, tokenManager)
	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	return server.httpServer.Handler, token
}

func TestFriends_RequireAuth(t *testing.T) {
	router, _ := newFriendsRouter(t, &stubFriends{})
	if w := doUsersRequest(router, http.MethodGet, "/api/v1/friends", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestFriends_ListShape(t *testing.T) {
	bob := uuid.New()
	stub := &stubFriends{overview: model.FriendsOverview{
		Friends: []model.FriendLink{{UserID: bob, Username: "Bob", Presence: model.PresenceOnline}},
	}}
	router, token := newFriendsRouter(t, stub)

	w := doUsersRequest(router, http.MethodGet, "/api/v1/friends", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp FriendsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Friends) != 1 || resp.Friends[0].ID != bob.String() || resp.Friends[0].Status != "online" {
		t.Fatalf("friends = %+v", resp.Friends)
	}
	if resp.Incoming == nil || resp.Outgoing == nil {
		t.Fatalf("empty lists must be [] not null: %s", w.Body.String())
	}
}

func TestFriends_RequestStatusCodes(t *testing.T) {
	cases := []struct {
		name  string
		state model.FriendState
		err   error
		want  int
	}{
		{"new request", model.FriendPending, nil, http.StatusCreated},
		{"counter request", model.FriendAccepted, nil, http.StatusOK},
		{"unknown user", "", model.ErrUserNotFound, http.StatusNotFound},
		{"self", "", model.ErrFriendSelf, http.StatusBadRequest},
		{"already friends", "", model.ErrAlreadyFriends, http.StatusConflict},
		{"already sent", "", model.ErrFriendRequestExists, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubFriends{state: tc.state, err: tc.err}
			router, token := newFriendsRouter(t, stub)
			w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests", token, `{"username":"Bob"}`)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.want, w.Body.String())
			}
			if stub.lastName != "Bob" {
				t.Fatalf("service got username %q", stub.lastName)
			}
		})
	}
}

func TestFriends_AcceptAndRemoveUseIDFromPath(t *testing.T) {
	stub := &stubFriends{}
	router, token := newFriendsRouter(t, stub)
	bob := uuid.New()

	if w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests/"+bob.String()+"/accept", token, ""); w.Code != http.StatusNoContent {
		t.Fatalf("accept status = %d", w.Code)
	}
	if stub.lastID != bob {
		t.Fatalf("accept got %s, want %s", stub.lastID, bob)
	}
	if w := doUsersRequest(router, http.MethodDelete, "/api/v1/friends/"+bob.String(), token, ""); w.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d", w.Code)
	}
	if w := doUsersRequest(router, http.MethodDelete, "/api/v1/friends/not-a-uuid", token, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d, want 400", w.Code)
	}

	stub.err = model.ErrFriendRequestNotFound
	if w := doUsersRequest(router, http.MethodPost, "/api/v1/friends/requests/"+bob.String()+"/accept", token, ""); w.Code != http.StatusNotFound {
		t.Fatalf("missing request status = %d, want 404", w.Code)
	}
}
