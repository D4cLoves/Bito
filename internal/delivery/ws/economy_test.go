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
	mu             sync.Mutex
	balances       map[string]int64
	lockErr        error
	settleErr      error
	settleFailures int
	refundErr      error
	locks          []lockCall
	settles        []economy.Result
	refunds        []string
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
	if b.settleFailures > 0 {
		b.settleFailures--
		return economy.Settlement{}, errors.New("db is busy")
	}
	if b.settleErr != nil {
		return economy.Settlement{}, b.settleErr
	}
	return economy.Settle(stake, r), nil
}

func (b *fakeBank) RefundGame(_ context.Context, gameID string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.refundErr != nil {
		return false, b.refundErr
	}
	b.refunds = append(b.refunds, gameID)
	return true, nil
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

func TestHub_SeatedElsewhere(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	first := hub.CreateRoom(testSettings(3))
	defer first.Close()
	first.RegisterClient(createTestClient("p1", "Alice", first))
	time.Sleep(20 * time.Millisecond)

	if !hub.SeatedElsewhere("p1", "") {
		t.Error("p1 sits at the first table")
	}
	if hub.SeatedElsewhere("p1", first.ID()) {
		t.Error("the player's own table does not count as elsewhere")
	}
}

func TestHub_CreatorReservesSeat(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	s := testSettings(3)
	s.CreatedBy = "p1"
	table, err := hub.CreateTable(s)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	defer table.Close()

	if !hub.SeatedElsewhere("p1", "") {
		t.Error("the creator's reservation must count as a seat")
	}
	if info := table.Info(); info.Players != 1 {
		t.Errorf("players = %d, want the creator's reserved seat", info.Players)
	}
	if _, err = hub.CreateTable(s); !errors.Is(err, ErrSeatedElsewhere) {
		t.Errorf("second table err = %v, want ErrSeatedElsewhere", err)
	}
	if _, _, err = hub.MatchOrCreate(MatchQuery{Stake: 100}, "p1"); !errors.Is(err, ErrSeatedElsewhere) {
		t.Errorf("quick play err = %v, want ErrSeatedElsewhere", err)
	}

	other := hub.CreateRoom(testSettings(3))
	defer other.Close()
	if reason := hub.admit(other, "p1"); reason != errOtherTable {
		t.Errorf("joining another table: reason = %q, want %q", reason, errOtherTable)
	}
	if reason := hub.admit(table, "p1"); reason != "" {
		t.Errorf("joining own table: reason = %q, want none", reason)
	}
}

func TestHub_NewSearchReleasesOldDuelReservation(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	cheap, _, _ := hub.MatchOrCreate(MatchQuery{Stake: 100}, "p1")
	defer cheap.Close()

	rich, _, err := hub.MatchOrCreate(MatchQuery{Stake: 500}, "p1")
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	defer rich.Close()
	if cheap.Reserved("p1") {
		t.Error("a new search must release the previous duel reservation")
	}
	if !rich.Reserved("p1") {
		t.Error("the new duel must hold the reservation")
	}

	s := testSettings(3)
	s.CreatedBy = "p1"
	table, err := hub.CreateTable(s)
	if err != nil {
		t.Fatalf("a pending duel search must not block creating a table: %v", err)
	}
	defer table.Close()
	if rich.Reserved("p1") {
		t.Error("creating a table must release the duel reservation")
	}
}

func TestHub_TableWithoutPlayersLeavesList(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	s := testSettings(3)
	s.CreatedBy = "p1"
	table, err := hub.CreateTable(s)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if len(hub.ListOpenRooms()) != 1 {
		t.Fatal("a reserved table must be listed")
	}

	table.Unreserve("p1")
	if list := hub.ListOpenRooms(); len(list) != 0 {
		t.Errorf("open rooms = %+v, want none without players or reservations", list)
	}
}

func duelSettings(mode game.GameMode, deck game.DeckType, stake int64) RoomSettings {
	return RoomSettings{Mode: mode, Deck: deck, MaxPlayers: 2, Stake: stake, Matchmaking: true}
}

func TestHub_MatchOrCreate(t *testing.T) {
	cases := []struct {
		name  string
		query MatchQuery
		reuse bool
	}{
		{"exact settings", MatchQuery{Mode: game.ModePerevodnoy, Deck: game.Deck24, Stake: 500}, true},
		{"any mode and deck", MatchQuery{Stake: 500}, true},
		{"any deck", MatchQuery{Mode: game.ModePerevodnoy, Stake: 500}, true},
		{"other mode", MatchQuery{Mode: game.ModePodkidnoy, Stake: 500}, false},
		{"other deck", MatchQuery{Deck: game.Deck36, Stake: 500}, false},
		{"other stake", MatchQuery{Stake: 250}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hub := NewHub(nil, nil, nil)
			duel, created, _ := hub.MatchOrCreate(MatchQuery{Mode: game.ModePerevodnoy, Deck: game.Deck24, Stake: 500}, "p1")
			defer duel.Close()
			if !created {
				t.Fatal("the first search must open a table")
			}
			company := hub.CreateRoom(RoomSettings{Mode: game.ModePerevodnoy, Deck: game.Deck24, MaxPlayers: 3, Stake: 500})
			defer company.Close()

			room, created, _ := hub.MatchOrCreate(tc.query, "p2")
			defer room.Close()
			if tc.reuse && (created || room != duel) {
				t.Errorf("want the waiting duel, got created=%v", created)
			}
			if !tc.reuse && (!created || room == duel) {
				t.Errorf("want a new duel, got created=%v", created)
			}
			if room == company {
				t.Error("quick play must never seat a player at a company table")
			}
		})
	}
}

func TestHub_MatchOrCreateFillsMissingSettings(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	room, _, _ := hub.MatchOrCreate(MatchQuery{Stake: 100}, "p1")
	defer room.Close()

	info := room.Info()
	if info.Mode != "podkidnoy" || info.Deck != 36 || info.MaxPlayers != 2 || !info.Matchmaking || info.CreatedBy != "p1" {
		t.Errorf("created duel = %+v, want podkidnoy, 36, 2 players, matchmaking", info)
	}
}

func TestHub_ReservationKeepsDuelForTwo(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	query := MatchQuery{Stake: 100}

	first, _, _ := hub.MatchOrCreate(query, "p1")
	defer first.Close()
	second, created, _ := hub.MatchOrCreate(query, "p2")
	if created || second != first {
		t.Fatal("the second player must get the waiting duel")
	}
	third, created, _ := hub.MatchOrCreate(query, "p3")
	defer third.Close()
	if !created || third == first {
		t.Error("two reserved seats fill a duel even before anyone connects; the third player needs a new table")
	}

	again, created, _ := hub.MatchOrCreate(query, "p1")
	if created || again != first {
		t.Error("a repeated search must return the table the player already reserved")
	}
}

func TestHub_SimultaneousSearchesMeet(t *testing.T) {
	for i := 0; i < 50; i++ {
		hub := NewHub(nil, nil, nil)
		rooms := make([]*Room, 2)
		var wg sync.WaitGroup
		for p := range rooms {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rooms[p], _, _ = hub.MatchOrCreate(MatchQuery{Stake: 250}, string(rune('a'+p)))
			}()
		}
		wg.Wait()
		if rooms[0] != rooms[1] {
			t.Fatalf("run %d: two players searching at once ended up at different tables", i)
		}
		rooms[0].Close()
	}
}

func TestHub_DuelTablesAreHiddenFromList(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	duel := hub.CreateRoom(duelSettings(game.ModePodkidnoy, game.Deck36, 100))
	defer duel.Close()
	company := hub.CreateRoom(testSettings(3))
	defer company.Close()
	company.Reserve("creator")

	list := hub.ListOpenRooms()
	if len(list) != 1 || list[0].ID != company.ID() {
		t.Errorf("open rooms = %+v, want only the company table", list)
	}
}

func finishBySurrender(t *testing.T, bank *fakeBank, id string) (*Room, *Client) {
	t.Helper()
	room := NewRoom(id, stakedSettings(100), bank)
	room.settleRetryDelay = time.Millisecond
	go room.Run()
	t.Cleanup(room.Close)

	c1 := createTestClient("p1", "Alice", room)
	c2 := createTestClient("p2", "Bob", room)
	room.RegisterClient(c1)
	room.RegisterClient(c2)
	time.Sleep(50 * time.Millisecond)

	surrender, _ := json.Marshal(map[string]string{"type": TypeSurrender})
	room.incoming <- IncomingMessage{Client: c1, Data: surrender}
	time.Sleep(50 * time.Millisecond)
	return room, c2
}

func TestRoom_SettleRetriesTransientFailure(t *testing.T) {
	bank := &fakeBank{settleFailures: 2}
	_, c2 := finishBySurrender(t, bank, "room-retry")

	if _, settles := bank.counts(); settles != settleAttempts {
		t.Fatalf("settle calls = %d, want %d", settles, settleAttempts)
	}
	if len(bank.refunds) != 0 {
		t.Errorf("refunds = %v, want none after a successful retry", bank.refunds)
	}
	over, ok := lastOfType[GameOverPayload](t, c2, TypeGameOver)
	if !ok || over.Payload.Payouts["p2"] != 95 || over.Payload.Payouts["p1"] != -100 {
		t.Errorf("payouts = %v, want the settled result", over.Payload.Payouts)
	}
}

func TestRoom_RefundsStakesWhenSettleKeepsFailing(t *testing.T) {
	bank := &fakeBank{settleErr: errors.New("db is down")}
	_, c2 := finishBySurrender(t, bank, "room-refund")

	if _, settles := bank.counts(); settles != settleAttempts {
		t.Fatalf("settle calls = %d, want %d", settles, settleAttempts)
	}
	if len(bank.refunds) != 1 || bank.refunds[0] != "room-refund" {
		t.Fatalf("refunds = %v, want exactly this game", bank.refunds)
	}

	var gotNotice bool
	var over Message[GameOverPayload]
	for len(c2.send) > 0 {
		raw := <-c2.send
		var probe Message[json.RawMessage]
		_ = json.Unmarshal(raw, &probe)
		switch probe.Type {
		case TypeError:
			var msg Message[ErrorPayload]
			_ = json.Unmarshal(raw, &msg)
			gotNotice = gotNotice || msg.Payload.Message == errStakesRefunded
		case TypeGameOver:
			_ = json.Unmarshal(raw, &over)
		}
	}
	if !gotNotice {
		t.Error("players must be told that stakes were returned")
	}
	if len(over.Payload.Payouts) != 2 || over.Payload.Payouts["p1"] != 0 || over.Payload.Payouts["p2"] != 0 {
		t.Errorf("payouts = %v, want zero for both players", over.Payload.Payouts)
	}
}

func TestRoom_FailedRefundStillFinishesGame(t *testing.T) {
	bank := &fakeBank{settleErr: errors.New("db is down"), refundErr: errors.New("still down")}
	room, c2 := finishBySurrender(t, bank, "room-refund-down")

	if _, ok := lastOfType[GameOverPayload](t, c2, TypeGameOver); !ok {
		t.Fatal("game_over must be sent even when the bank is unreachable")
	}
	if room.Holds("p1") {
		t.Error("a finished table must release seats")
	}
}
