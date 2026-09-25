package http

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/game"

	"github.com/gorilla/websocket"
)

func TestE2E_DurakWebSocketGame(t *testing.T) {
	// 1. Создаем хаб и поднимаем тестовый HTTP-сервер
	hub := ws.NewHub()
	server := NewServer(":0", hub)
	ts := httptest.NewServer(server.httpServer.Handler)
	defer ts.Close()

	// Преобразуем http:// в ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// 2. Подключаем первого игрока (Алиса)
	aliceURL := wsURL + "?room_id=test_table&player_id=p1&player_name=Alice"
	wsAlice, _, err := websocket.DefaultDialer.Dial(aliceURL, nil)
	if err != nil {
		t.Fatalf("Alice failed to connect: %v", err)
	}
	defer wsAlice.Close()

	// 3. Подключаем второго игрока (Боб) -> комната наполнилась (2 игрока) и стартует!
	bobURL := wsURL + "?room_id=test_table&player_id=p2&player_name=Bob"
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

	if aliceState.Payload.AttackerID == "p1" {
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