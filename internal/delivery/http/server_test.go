package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/game"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const testAllowedOrigin = "http://localhost:5173"

// newTestWSServer starts the HTTP server with a real token manager and returns the hub and the /ws URL.
func newTestWSServer(t *testing.T) (*jwt.TokenManager, *ws.Hub, string) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	hub := ws.NewHub(tokenManager, []string{testAllowedOrigin})
	server := NewServer(":0", hub, nil, NewTableHandler(hub), tokenManager)
	ts := httptest.NewServer(server.httpServer.Handler)
	t.Cleanup(ts.Close)

	return tokenManager, hub, "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
}

func TestWS_RejectsUnauthenticated(t *testing.T) {
	tokenManager, _, wsURL := newTestWSServer(t)
	userID := uuid.New()

	refreshToken, _, err := tokenManager.GenerateRefreshToken(userID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	cases := []struct {
		name  string
		query string
	}{
		{"no token", "?room_id=t1"},
		{"garbage token", "?room_id=t1&token=garbage"},
		{"refresh token instead of access", "?room_id=t1&token=" + refreshToken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL+tc.query, nil)
			if err == nil {
				conn.Close()
				t.Fatal("expected handshake to fail")
			}
			if resp == nil || resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %v", resp)
			}
		})
	}
}

func TestWS_OriginCheck(t *testing.T) {
	tokenManager, hub, wsURL := newTestWSServer(t)
	room := hub.CreateRoom(game.ModePodkidnoy, game.Deck36, 2, "creator")
	accessToken, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	url := wsURL + "?room_id=" + room.ID() + "&token=" + accessToken

	foreign := http.Header{"Origin": []string{"http://evil.example"}}
	if conn, resp, err := websocket.DefaultDialer.Dial(url, foreign); err == nil {
		conn.Close()
		t.Fatal("expected foreign origin to be rejected")
	} else if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign origin, got %v", resp)
	}

	allowed := http.Header{"Origin": []string{testAllowedOrigin}}
	conn, _, err := websocket.DefaultDialer.Dial(url, allowed)
	if err != nil {
		t.Fatalf("allowed origin failed to connect: %v", err)
	}
	conn.Close()
}

func TestE2E_DurakWebSocketGame(t *testing.T) {
	// 1. Поднимаем тестовый сервер и выпускаем access-токены игрокам
	tokenManager, hub, wsURL := newTestWSServer(t)
	room := hub.CreateRoom(game.ModePodkidnoy, game.Deck36, 2, "creator")

	aliceID := uuid.New()
	aliceToken, err := tokenManager.GenerateAccessToken(aliceID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate Alice token: %v", err)
	}
	bobToken, err := tokenManager.GenerateAccessToken(uuid.New(), "Bob")
	if err != nil {
		t.Fatalf("failed to generate Bob token: %v", err)
	}

	// 2. Подключаем первого игрока (Алиса)
	aliceURL := wsURL + "?room_id=" + room.ID() + "&token=" + aliceToken
	wsAlice, _, err := websocket.DefaultDialer.Dial(aliceURL, nil)
	if err != nil {
		t.Fatalf("Alice failed to connect: %v", err)
	}
	defer wsAlice.Close()

	// 3. Подключаем второго игрока (Боб) -> комната наполнилась (2 игрока) и стартует!
	bobURL := wsURL + "?room_id=" + room.ID() + "&token=" + bobToken
	wsBob, _, err := websocket.DefaultDialer.Dial(bobURL, nil)
	if err != nil {
		t.Fatalf("Bob failed to connect: %v", err)
	}
	defer wsBob.Close()

	// 4. Читаем стартовое состояние игры у обоих игроков
	var aliceState ws.Message[ws.GameStatePayload]
	if err := wsAlice.ReadJSON(&aliceState); err != nil {
		t.Fatalf("Alice failed to read initial state: %v", err)
	}

	var bobState ws.Message[ws.GameStatePayload]
	if err := wsBob.ReadJSON(&bobState); err != nil {
		t.Fatalf("Bob failed to read initial state: %v", err)
	}

	// Проверяем: игра началась, у каждого по 6 карт в руке
	if aliceState.Payload.Phase != "battle" {
		t.Fatalf("expected phase 'battle', got '%s'", aliceState.Payload.Phase)
	}
	if len(aliceState.Payload.MyHand) != 6 {
		t.Fatalf("Alice expected 6 cards, got %d", len(aliceState.Payload.MyHand))
	}
	if len(bobState.Payload.MyHand) != 6 {
		t.Fatalf("Bob expected 6 cards, got %d", len(bobState.Payload.MyHand))
	}

	// 5. Определяем, кто ходит первым
	var attackerConn, defenderConn *websocket.Conn
	var cardToPlay game.Card

	if aliceState.Payload.AttackerID == aliceID.String() {
		t.Logf("Алиса ходит первой!")
		attackerConn = wsAlice
		defenderConn = wsBob
		cardToPlay = aliceState.Payload.MyHand[0]
	} else {
		t.Logf("Боб ходит первым!")
		attackerConn = wsBob
		defenderConn = wsAlice
		cardToPlay = bobState.Payload.MyHand[0]
	}

	// 6. Атакующий делает реальный ход: шлет в сокет TypeAttack
	attackAction, _ := ws.NewMessage(ws.TypeAttack, ws.AttackPayload{
		AttackCard: cardToPlay,
	})
	if err := attackerConn.WriteMessage(websocket.TextMessage, attackAction); err != nil {
		t.Fatalf("failed to send attack action: %v", err)
	}

	// 7. Проверяем, что защитник мгновенно получил по сокету обновление стола
	_ = defenderConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, rawMsg, err := defenderConn.ReadMessage()
	if err != nil {
		t.Fatalf("Defender did not receive table update: %v", err)
	}

	var updateMsg ws.Message[ws.GameStatePayload]
	if err := json.Unmarshal(rawMsg, &updateMsg); err != nil {
		t.Fatalf("failed to parse table update: %v", err)
	}

	// Проверяем, что на столе лежит ровно эта карта!
	if len(updateMsg.Payload.Table) != 1 {
		t.Fatalf("expected 1 card on table, got %d", len(updateMsg.Payload.Table))
	}
	if updateMsg.Payload.Table[0].Attack != cardToPlay {
		t.Fatalf("expected card %v on table, got %v", cardToPlay, updateMsg.Payload.Table[0].Attack)
	}

	t.Logf("УСПЕХ! Карта %v успешно легла на стол по сети через WebSocket!", cardToPlay)
}

func TestWS_UnknownRoomReturns404(t *testing.T) {
	tokenManager, _, wsURL := newTestWSServer(t)
	accessToken, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL+"?room_id=missing&token="+accessToken, nil)
	if err == nil {
		conn.Close()
		t.Fatal("expected handshake to fail for unknown room")
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", resp)
	}
}

func TestHub_ListOpenRooms(t *testing.T) {
	tokenManager, hub, wsURL := newTestWSServer(t)
	room := hub.CreateRoom(game.ModePerevodnoy, game.Deck24, 2, "creator")

	open := hub.ListOpenRooms()
	if len(open) != 1 || open[0].ID != room.ID() {
		t.Fatalf("expected the new room in open list, got %+v", open)
	}
	if open[0].Mode != "perevodnoy" || open[0].Deck != 24 || open[0].MaxPlayers != 2 || open[0].Players != 0 {
		t.Errorf("unexpected room info: %+v", open[0])
	}

	for _, name := range []string{"Alice", "Bob"} {
		token, err := tokenManager.GenerateAccessToken(uuid.New(), name)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?room_id="+room.ID()+"&token="+token, nil)
		if err != nil {
			t.Fatalf("%s failed to connect: %v", name, err)
		}
		defer conn.Close()
	}

	deadline := time.Now().Add(2 * time.Second)
	for !room.Started() {
		if time.Now().After(deadline) {
			t.Fatal("game did not start after table filled")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if open := hub.ListOpenRooms(); len(open) != 0 {
		t.Errorf("started room must leave the open list, got %+v", open)
	}
}
