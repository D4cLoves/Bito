package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/game"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func lobbyURL(wsURL string) string {
	return strings.TrimSuffix(wsURL, "/ws") + "/ws/lobby"
}

func TestLobbyWS_RejectsUnauthenticated(t *testing.T) {
	_, _, wsURL := newTestWSServer(t)

	conn, resp, err := websocket.DefaultDialer.Dial(lobbyURL(wsURL), nil)
	if err == nil {
		conn.Close()
		t.Fatal("expected handshake to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %v", resp)
	}
}

func TestLobbyWS_CookieHandshakeCountsOnlinePlayers(t *testing.T) {
	tokenManager, hub, wsURL := newTestWSServer(t)
	aliceToken, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	header := http.Header{}
	header.Set("Origin", testAllowedOrigin)
	header.Set("Cookie", "accessToken="+aliceToken)

	tab1, resp, err := websocket.DefaultDialer.Dial(lobbyURL(wsURL), header)
	if err != nil {
		t.Fatalf("cookie-only handshake failed: %v (%v)", err, resp)
	}
	tab2, _, err := websocket.DefaultDialer.Dial(lobbyURL(wsURL), header)
	if err != nil {
		t.Fatalf("second tab failed to connect: %v", err)
	}

	waitOnline(t, hub.LobbyOnline, 1)

	tab1.Close()
	waitOnline(t, hub.LobbyOnline, 1)

	tab2.Close()
	waitOnline(t, hub.LobbyOnline, 0)
}

func waitOnline(t *testing.T, online func() int, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := online(); got == want {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("online = %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLobbyWS_SendsOpenTablesOnConnect(t *testing.T) {
	tokenManager, hub, wsURL := newTestWSServer(t)
	creator := uuid.New()
	room, err := hub.CreateTable(ws.RoomSettings{Mode: game.ModePodkidnoy, Deck: game.Deck36, MaxPlayers: 3, CreatedBy: creator.String()})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	defer room.Close()

	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Bob")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	header := http.Header{}
	header.Set("Origin", testAllowedOrigin)
	header.Set("Cookie", "accessToken="+token)

	conn, _, err := websocket.DefaultDialer.Dial(lobbyURL(wsURL), header)
	if err != nil {
		t.Fatalf("lobby handshake failed: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg ws.Message[ws.LobbyStatePayload]
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read lobby state: %v", err)
	}
	if msg.Type != ws.TypeLobbyState {
		t.Fatalf("type = %q, want %q", msg.Type, ws.TypeLobbyState)
	}
	if msg.Payload.Online != 1 || len(msg.Payload.Tables) != 1 || msg.Payload.Tables[0].ID != room.ID() {
		t.Fatalf("payload = %+v, want online 1 and the created table", msg.Payload)
	}
}
