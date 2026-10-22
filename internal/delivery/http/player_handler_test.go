package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/model"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

type stubPlayers struct {
	profile model.PlayerProfile
	err     error
	gotID   uuid.UUID
}

func (s *stubPlayers) Profile(_ context.Context, _, id uuid.UUID) (model.PlayerProfile, error) {
	s.gotID = id
	return s.profile, s.err
}

func newPlayersRouter(t *testing.T, stub *stubPlayers) (http.Handler, string) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	server := NewServer(":0", ws.NewHub(tokenManager, nil, nil), Handlers{Player: NewPlayerHandler(stub)}, tokenManager)
	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	return server.httpServer.Handler, token
}

func TestPlayers_ProfileHidesMoney(t *testing.T) {
	bob := uuid.New()
	stub := &stubPlayers{profile: model.PlayerProfile{
		ID:       bob,
		Username: "Bob",
		Stats:    model.UserStats{Wins: 2, Losses: 1, TotalGames: 3},
		Relation: model.RelationFriend,
		Presence: model.PresenceOnline,
		Games:    []model.GameHistoryEntry{{ID: uuid.New(), Mode: "podkidnoy", Payout: -100, Stake: 100}},
	}}
	router, token := newPlayersRouter(t, stub)

	w := doUsersRequest(router, http.MethodGet, "/api/v1/players/"+bob.String(), token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body.String())
	}
	if stub.gotID != bob {
		t.Fatalf("service got %s", stub.gotID)
	}
	body := w.Body.String()
	for _, secret := range []string{"payout", "balance", "email"} {
		if strings.Contains(body, secret) {
			t.Fatalf("public profile exposes %q: %s", secret, body)
		}
	}
	var resp PlayerProfileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "online" || resp.Relation != "friend" || resp.Stats.Wins != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	if len(resp.Games) != 1 || resp.Games[0].Outcome != "lose" {
		t.Fatalf("games = %+v, want one lost game", resp.Games)
	}
}

func TestPlayers_StrangerHasNoStatus(t *testing.T) {
	stub := &stubPlayers{profile: model.PlayerProfile{ID: uuid.New(), Username: "Eve", Relation: model.RelationNone}}
	router, token := newPlayersRouter(t, stub)

	w := doUsersRequest(router, http.MethodGet, "/api/v1/players/"+stub.profile.ID.String(), token, "")
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"status"`) {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"games":[]`) {
		t.Fatalf("games must be [] not null: %s", w.Body.String())
	}
}

func TestPlayers_Errors(t *testing.T) {
	router, token := newPlayersRouter(t, &stubPlayers{err: model.ErrUserNotFound})

	if w := doUsersRequest(router, http.MethodGet, "/api/v1/players/"+uuid.NewString(), "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token status = %d, want 401", w.Code)
	}
	if w := doUsersRequest(router, http.MethodGet, "/api/v1/players/nope", token, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d, want 400", w.Code)
	}
	if w := doUsersRequest(router, http.MethodGet, "/api/v1/players/"+uuid.NewString(), token, ""); w.Code != http.StatusNotFound {
		t.Fatalf("unknown player status = %d, want 404", w.Code)
	}
}
