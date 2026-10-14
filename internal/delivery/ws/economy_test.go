package ws

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"bito/internal/economy"
	"bito/internal/game"
	"bito/internal/model"

	"github.com/google/uuid"
)

type lockCall struct {
	gameID  string
	stake   int64
	players []string
}

type fakeBank struct {
	mu        sync.Mutex
	balances  map[string]int64
	lockErr   error
	settleErr error
	locks     []lockCall
	settles   []economy.Result
}

func (b *fakeBank) Balance(_ context.Context, userID string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.balances[userID], nil
}

func (b *fakeBank) LockStakes(_ context.Context, gameID, _ string, _ int, stake int64, players []string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lockErr != nil {
		err := b.lockErr
		b.lockErr = nil
		return err
	}
	sorted := append([]string(nil), players...)
	sort.Strings(sorted)
	b.locks = append(b.locks, lockCall{gameID: gameID, stake: stake, players: sorted})
	return nil
}

func (b *fakeBank) Settle(_ context.Context, _ string, stake int64, r economy.Result) (economy.Settlement, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.settles = append(b.settles, r)
	if b.settleErr != nil {
		return economy.Settlement{}, b.settleErr
	}
	return economy.Settle(stake, r), nil
}

func (b *fakeBank) counts() (locks, settles int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.locks), len(b.settles)
}

func stakedSettings(stake int64) RoomSettings {
	s := testSettings(2)
	s.Stake = stake
	return s
}

func lastOfType[T any](t *testing.T, c *Client, msgType string) (Message[T], bool) {
	t.Helper()
	var found Message[T]
	ok := false
	for len(c.send) > 0 {
		raw := <-c.send
		var probe Message[json.RawMessage]
		_ = json.Unmarshal(raw, &probe)
		if probe.Type == msgType {
			_ = json.Unmarshal(raw, &found)
			ok = true
		}
	}
	return found, ok
}

func TestRoom_LocksStakesOnStart(t *testing.T) {
	bank := &fakeBank{}
	room := NewRoom("room-lock", stakedSettings(500), bank)
	go room.Run()
	defer room.Close()

	room.RegisterClient(createTestClient("p1", "Alice", room))
	time.Sleep(20 * time.Millisecond)
	if locks, _ := bank.counts(); locks != 0 {
		t.Fatalf("stakes locked with one player, want lock only at start")
	}

	room.RegisterClient(createTestClient("p2", "Bob", room))
	time.Sleep(50 * time.Millisecond)

	if locks, _ := bank.counts(); locks != 1 {
		t.Fatalf("locks = %d, want 1", locks)
	}
	call := bank.locks[0]
	if call.gameID != "room-lock" || call.stake != 500 || len(call.players) != 2 || call.players[0] != "p1" || call.players[1] != "p2" {
		t.Errorf("lock call = %+v, want room-lock / 500 / [p1 p2]", call)
	}
	if !room.Started() {
		t.Error("game must start after stakes are locked")
	}
}

func TestRoom_PlayerWithoutFundsIsRemoved(t *testing.T) {
	poor := uuid.New()
	bank := &fakeBank{lockErr: &model.InsufficientFundsError{UserID: poor}}
	room := NewRoom("room-poor", stakedSettings(500), bank)
	go room.Run()
	defer room.Close()

	rich := createTestClient("rich", "Alice", room)
	broke := createTestClient(poor.String(), "Bob", room)
	room.RegisterClient(rich)
	room.RegisterClient(broke)
	time.Sleep(50 * time.Millisecond)

	if room.Started() {
		t.Fatal("game must not start when a stake cannot be locked")
	}
	if room.Holds(poor.String()) {
		t.Error("player without funds must lose the seat")
	}
	if !room.Holds("rich") {
		t.Error("the other player keeps the seat and waits")
	}
	if msg, ok := lastOfType[ErrorPayload](t, broke, TypeError); !ok || msg.Payload.Message != errNoFunds {
		t.Errorf("error to the poor player = %+v, want %q", msg.Payload, errNoFunds)
	}

	room.RegisterClient(createTestClient("rich2", "Carol", room))
	time.Sleep(50 * time.Millisecond)
	if !room.Started() {
		t.Error("game must start once a player with funds sits down")
	}
}

func TestRoom_LockFailureClosesTable(t *testing.T) {
	bank := &fakeBank{lockErr: errors.New("db is down")}
	room := NewRoom("room-db-down", stakedSettings(100), bank)
	go room.Run()

	c1 := createTestClient("p1", "Alice", room)
	room.RegisterClient(c1)
	room.RegisterClient(createTestClient("p2", "Bob", room))
	time.Sleep(50 * time.Millisecond)

	if room.Started() {
		t.Fatal("game must not start when the bank fails")
	}
	if room.Holds("p1") {
		t.Error("a closed table must not hold seats")
	}
}

func TestRoom_SettlesOnceAndSendsPayouts(t *testing.T) {
	bank := &fakeBank{}
	room := NewRoom("room-settle", stakedSettings(100), bank)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)
	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(50 * time.Millisecond)

	surrender, _ := json.Marshal(map[string]string{"type": TypeSurrender})
	room.incoming <- IncomingMessage{Client: c1, Data: surrender}
	room.incoming <- IncomingMessage{Client: c2, Data: surrender}
	time.Sleep(50 * time.Millisecond)

	if _, settles := bank.counts(); settles != 1 {
		t.Fatalf("settles = %d, want exactly 1", settles)
	}
	got := bank.settles[0]
	if len(got.Places) != 2 || got.Places[0] != "p2" || got.Places[1] != "p1" || !got.Losers["p1"] || got.Losers["p2"] {
		t.Errorf("settle result = %+v, want places [p2 p1] and loser p1", got)
	}

	over, ok := lastOfType[GameOverPayload](t, c2, TypeGameOver)
	if !ok {
		t.Fatal("expected game_over")
	}
	if over.Payload.Payouts["p2"] != 95 || over.Payload.Payouts["p1"] != -100 || over.Payload.Stake != 100 {
		t.Errorf("payouts = %v stake = %d, want p2:+95 p1:-100 stake 100", over.Payload.Payouts, over.Payload.Stake)
	}
	if room.Holds("p1") || room.Holds("p2") {
		t.Error("a finished table must release seats")
	}
}

func TestRoom_SeatIsKeptWhileGameRuns(t *testing.T) {
	room := NewRoom("room-seat", testSettings(3), nil)
	go room.Run()
	defer room.Close()

	c1 := createTestClient("p1", "Alice", room)
	room.RegisterClient(c1)
	room.RegisterClient(createTestClient("p2", "Bob", room))
	time.Sleep(20 * time.Millisecond)
	if !room.Holds("p1") {
		t.Fatal("seat must be taken on join")
	}

	room.unregister <- c1
	time.Sleep(20 * time.Millisecond)
	if room.Holds("p1") {
		t.Fatal("leaving before the start must free the seat")
	}

	c1 = createTestClient("p1", "Alice", room)
	room.RegisterClient(c1)
	room.RegisterClient(createTestClient("p3", "Carol", room))
	time.Sleep(50 * time.Millisecond)
	if !room.Started() {
		t.Fatal("game must start with three players")
	}

	room.unregister <- c1
	time.Sleep(20 * time.Millisecond)
	if !room.Holds("p1") {
		t.Error("a disconnected player keeps the seat while the game runs")
	}
}

func TestHub_SeatedElsewhereAndQuickMatch(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	settings := testSettings(2)
	settings.Stake = 250

	first := hub.CreateRoom(settings)
	defer first.Close()
	first.RegisterClient(createTestClient("p1", "Alice", first))
	time.Sleep(20 * time.Millisecond)

	if !hub.SeatedElsewhere("p1", "") {
		t.Error("p1 sits at the first table")
	}
	if hub.SeatedElsewhere("p1", first.ID()) {
		t.Error("the player's own table does not count as elsewhere")
	}

	if room, ok := hub.FindOpenRoom(settings, "p2"); !ok || room != first {
		t.Error("quick match must find the open table with the same settings")
	}
	if room, ok := hub.FindOpenRoom(settings, "p1"); !ok || room != first {
		t.Error("quick match must return the table the player already sits at")
	}

	other := settings
	other.Stake = 500
	if _, ok := hub.FindOpenRoom(other, "p2"); ok {
		t.Error("a table with another stake must not match")
	}
	other = settings
	other.Mode = game.ModePerevodnoy
	if _, ok := hub.FindOpenRoom(other, "p2"); ok {
		t.Error("a table with another mode must not match")
	}
}
