package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

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
