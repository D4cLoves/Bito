package ws

import (
	"encoding/json"
	"testing"
	"time"

	"bito/internal/game"
)

// Helper to create a test client with a monitored send channel.
func createTestClient(id, name string, room *Room) *Client {
	return &Client{
		id:   id,
		name: name,
		room: room,
		send: make(chan []byte, 100),
	}
}

func TestRoom_StartGameAndPersonalizedState(t *testing.T) {
	room := NewRoom("room-1", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)

	// Register player 1
	room.RegisterClient(c1)

	// Game should not start with 1 player
	time.Sleep(20 * time.Millisecond)
	if len(c1.send) != 0 {
		t.Fatalf("expected 0 messages before game start, got %d", len(c1.send))
	}

	// Register player 2 -> triggers game start
	room.RegisterClient(c2)

	// Wait for game start and state dispatch
	time.Sleep(50 * time.Millisecond)

	if len(c1.send) == 0 || len(c2.send) == 0 {
		t.Fatalf("expected both players to receive game state")
	}

	// Verify player 1 state
	raw1 := <-c1.send
	var msg1 Message[GameStatePayload]
	if err := json.Unmarshal(raw1, &msg1); err != nil {
		t.Fatalf("failed to unmarshal msg1: %v", err)
	}

	if msg1.Type != TypeGameState {
		t.Errorf("expected type %s, got %s", TypeGameState, msg1.Type)
	}
	if len(msg1.Payload.MyHand) != 6 {
		t.Errorf("expected 6 cards in MyHand, got %d", len(msg1.Payload.MyHand))
	}
	if len(msg1.Payload.Players) != 2 {
		t.Errorf("expected 2 players in view, got %d", len(msg1.Payload.Players))
	}

	// Verify opponent's cards are hidden (Zero-Trust rule)
	for _, p := range msg1.Payload.Players {
		if p.ID == "p2" && p.HandCount != 6 {
			t.Errorf("expected opponent to have 6 cards count, got %d", p.HandCount)
		}
	}
}

func TestRoom_HandleIncomingAction(t *testing.T) {
	room := NewRoom("room-2", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)

	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(50 * time.Millisecond)

	// Drain initial game state messages
	raw1 := <-c1.send
	raw2 := <-c2.send

	var state1, state2 Message[GameStatePayload]
	_ = json.Unmarshal(raw1, &state1)
	_ = json.Unmarshal(raw2, &state2)

	// Find who is attacker and get card from their hand
	var attackerClient *Client
	var cardToPlay game.Card
	if state1.Payload.AttackerID == "p1" {
		attackerClient = c1
		cardToPlay = state1.Payload.MyHand[0]
	} else {
		attackerClient = c2
		cardToPlay = state2.Payload.MyHand[0]
	}

	attackMsg, _ := NewMessage(TypeAttack, AttackPayload{AttackCard: cardToPlay})
	room.incoming <- IncomingMessage{
		Client: attackerClient,
		Data:   attackMsg,
	}

	time.Sleep(50 * time.Millisecond)

	// Both should receive updated game state with 1 pair on the table
	if len(attackerClient.send) == 0 {
		t.Fatalf("expected new state broadcast after attack")
	}

	rawUpdate := <-attackerClient.send
	var updated Message[GameStatePayload]
	_ = json.Unmarshal(rawUpdate, &updated)

	if len(updated.Payload.Table) != 1 {
		t.Errorf("expected 1 card on table, got %d", len(updated.Payload.Table))
	}
	if updated.Payload.Table[0].Attack != cardToPlay {
		t.Errorf("expected card %v on table, got %v", cardToPlay, updated.Payload.Table[0].Attack)
	}
}

func TestRoom_ChatBroadcast(t *testing.T) {
	room := NewRoom("room-chat", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)

	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(50 * time.Millisecond)

	// Drain initial states
	<-c1.send
	<-c2.send

	// Alice sends a chat message
	chatPayload, _ := NewMessage(TypeChat, ChatPayload{Message: "Привет, удачной игры!"})
	room.incoming <- IncomingMessage{
		Client: c1,
		Data:   chatPayload,
	}

	time.Sleep(50 * time.Millisecond)

	if len(c2.send) == 0 {
		t.Fatalf("expected Bob to receive chat message")
	}

	var chatMsg Message[ChatPayload]
	_ = json.Unmarshal(<-c2.send, &chatMsg)

	if chatMsg.Type != TypeChat {
		t.Errorf("expected type %s, got %s", TypeChat, chatMsg.Type)
	}
	if chatMsg.Payload.SenderName != "Alice" {
		t.Errorf("expected sender Alice, got %s", chatMsg.Payload.SenderName)
	}
	if chatMsg.Payload.Message != "Привет, удачной игры!" {
		t.Errorf("expected message 'Привет, удачной игры!', got %s", chatMsg.Payload.Message)
	}
}

func TestRoom_GracePeriodReconnect(t *testing.T) {
	room := NewRoom("room-grace", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)

	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(50 * time.Millisecond)

	// Drain initial states
	<-c1.send
	<-c2.send

	// Alice disconnects
	room.unregister <- c1
	time.Sleep(20 * time.Millisecond)

	// Check that timer is running (thread-safe getter)
	if !room.HasGraceTimer("p1") {
		t.Fatalf("expected disconnect timer for p1")
	}

	// Bob should receive disconnect notification
	if len(c2.send) > 0 {
		var statusMsg Message[PlayerStatusPayload]
		_ = json.Unmarshal(<-c2.send, &statusMsg)
		if statusMsg.Type != TypePlayerDisconnected || statusMsg.Payload.PlayerID != "p1" {
			t.Errorf("expected player_disconnected message for p1, got %v", statusMsg)
		}
	}

	// Alice reconnects with a new client connection
	c1New := createTestClient("p1", "Alice", room)
	room.RegisterClient(c1New)
	time.Sleep(50 * time.Millisecond)

	// Disconnect timer should be stopped and cleared
	if room.HasGraceTimer("p1") {
		t.Errorf("expected disconnect timer to be cleared upon reconnect")
	}

	// Bob should receive reconnect notification
	if len(c2.send) > 0 {
		var statusMsg Message[PlayerStatusPayload]
		_ = json.Unmarshal(<-c2.send, &statusMsg)
		if statusMsg.Type != TypePlayerReconnected || statusMsg.Payload.PlayerID != "p1" {
			t.Errorf("expected player_reconnected message for p1, got %v", statusMsg)
		}
	}

	// Alice should receive immediate game state sync
	if len(c1New.send) == 0 {
		t.Fatalf("expected reconnected client to receive state sync")
	}
	var syncState Message[GameStatePayload]
	_ = json.Unmarshal(<-c1New.send, &syncState)
	if syncState.Type != TypeGameState {
		t.Errorf("expected type %s, got %s", TypeGameState, syncState.Type)
	}
}

func TestRoom_ChatBeforeGameStart(t *testing.T) {
	room := NewRoom("room-lobby-chat", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	room.RegisterClient(c1)
	time.Sleep(20 * time.Millisecond)

	// Send chat while alone in lobby (game == nil)
	chatPayload, _ := NewMessage(TypeChat, ChatPayload{Message: "Кто играть?"})
	room.incoming <- IncomingMessage{
		Client: c1,
		Data:   chatPayload,
	}

	time.Sleep(30 * time.Millisecond)

	if len(c1.send) == 0 {
		t.Fatalf("expected Alice to receive her own chat message in lobby")
	}

	var chatMsg Message[ChatPayload]
	_ = json.Unmarshal(<-c1.send, &chatMsg)
	if chatMsg.Type != TypeChat || chatMsg.Payload.Message != "Кто играть?" {
		t.Errorf("unexpected chat message: %v", chatMsg)
	}
}

func TestRoom_LobbyCapacityLimit(t *testing.T) {
	room := NewRoom("room-cap", game.ModePodkidnoy, game.Deck36, 2)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)
	c3 := createTestClient("p3", "Charlie", room)

	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(30 * time.Millisecond)

	// Now try to register Charlie when game has already started with 2 players
	room.RegisterClient(c3)
	time.Sleep(30 * time.Millisecond)

	// Charlie should receive an error and not be admitted
	if len(c3.send) == 0 {
		t.Fatalf("expected error for 3rd player trying to join full room")
	}

	var errMsg Message[ErrorPayload]
	_ = json.Unmarshal(<-c3.send, &errMsg)
	if errMsg.Type != TypeError {
		t.Errorf("expected error message type, got %s", errMsg.Type)
	}
}

func TestClient_SendOnClosedDoesNotPanic(t *testing.T) {
	c := &Client{
		id:   "p-test",
		name: "Test",
		send: make(chan []byte, 1),
	}

	// Close client
	c.Close()

	// Double close should not panic
	c.Close()

	// Send on closed should return false and NOT panic
	ok := c.Send([]byte("hello"))
	if ok {
		t.Errorf("expected Send to return false on closed client")
	}
}
