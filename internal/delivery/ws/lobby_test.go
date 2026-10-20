package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"bito/internal/game"
	"bito/internal/model"

	"github.com/google/uuid"
)

type fakeTables struct {
	mu   sync.Mutex
	list []RoomInfo
}

func (f *fakeTables) ListOpenRooms() []RoomInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RoomInfo(nil), f.list...)
}

func (f *fakeTables) set(list ...RoomInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.list = list
}

func startLobby(t *testing.T) (*Lobby, <-chan struct{}) {
	t.Helper()
	return startLobbyWith(t, &fakeTables{}, time.Hour)
}

func startLobbyWith(t *testing.T, tables tableSource, interval time.Duration) (*Lobby, <-chan struct{}) {
	t.Helper()
	l := NewLobby(tables, interval)
	done := make(chan struct{})
	go func() {
		l.Run()
		close(done)
	}()
	t.Cleanup(func() {
		l.Close()
		<-done
	})
	return l, done
}

func lobbyClient(id string, l *Lobby) *Client {
	return &Client{
		id:    id,
		name:  id,
		owner: l,
		send:  make(chan []byte, 16),
	}
}

func isClosed(c *Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func TestLobby_OnlineCountsPlayersNotTabs(t *testing.T) {
	l, _ := startLobby(t)

	tab1 := lobbyClient("alice", l)
	tab2 := lobbyClient("alice", l)
	bob := lobbyClient("bob", l)
	for _, c := range []*Client{tab1, tab2, bob} {
		if !l.Register(c) {
			t.Fatalf("register %s failed", c.ID())
		}
	}
	if got := l.Online(); got != 2 {
		t.Fatalf("online after 3 tabs of 2 players = %d, want 2", got)
	}

	l.leave(tab1)
	if got := l.Online(); got != 2 {
		t.Fatalf("online after alice closed one tab = %d, want 2", got)
	}

	l.leave(tab2)
	if got := l.Online(); got != 1 {
		t.Fatalf("online after alice closed both tabs = %d, want 1", got)
	}
}

func TestLobby_LeaveClosesClient(t *testing.T) {
	l, _ := startLobby(t)
	c := lobbyClient("alice", l)
	l.Register(c)

	l.leave(c)
	l.Online()

	if !isClosed(c) {
		t.Fatal("client is still open after leaving the lobby")
	}
}

func TestLobby_LeaveTwiceIsSafe(t *testing.T) {
	l, _ := startLobby(t)
	c := lobbyClient("alice", l)
	l.Register(c)

	l.leave(c)
	l.leave(c)

	if got := l.Online(); got != 0 {
		t.Fatalf("online = %d, want 0", got)
	}
}

func TestLobby_CloseClosesEveryClient(t *testing.T) {
	l, done := startLobby(t)
	a := lobbyClient("alice", l)
	b := lobbyClient("bob", l)
	l.Register(a)
	l.Register(b)
	l.Online()

	l.Close()
	<-done

	if !isClosed(a) || !isClosed(b) {
		t.Fatalf("clients left open after lobby closed: alice=%v bob=%v", isClosed(a), isClosed(b))
	}
}

func TestLobby_CallsAfterCloseDoNotHang(t *testing.T) {
	l, done := startLobby(t)
	l.Close()
	<-done

	c := lobbyClient("alice", l)
	if l.Register(c) {
		t.Fatal("register succeeded on a closed lobby")
	}
	if got := l.Online(); got != 0 {
		t.Fatalf("online on a closed lobby = %d, want 0", got)
	}
	l.leave(c)
	if l.deliver(IncomingMessage{Client: c}) {
		t.Fatal("deliver reports a closed lobby as alive")
	}
}

func nextLobbyState(t *testing.T, c *Client) LobbyStatePayload {
	t.Helper()
	select {
	case data := <-c.send:
		var msg Message[LobbyStatePayload]
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("bad message %s: %v", data, err)
		}
		if msg.Type != TypeLobbyState {
			t.Fatalf("message type = %q, want %q", msg.Type, TypeLobbyState)
		}
		return msg.Payload
	case <-time.After(time.Second):
		t.Fatal("no lobby_state arrived")
		return LobbyStatePayload{}
	}
}

func expectSilence(t *testing.T, c *Client, d time.Duration) {
	t.Helper()
	select {
	case data := <-c.send:
		t.Fatalf("unexpected message: %s", data)
	case <-time.After(d):
	}
}

func TestLobby_NewClientGetsCurrentState(t *testing.T) {
	tables := &fakeTables{}
	tables.set(RoomInfo{ID: "t1", Players: 1, MaxPlayers: 3})
	l, _ := startLobbyWith(t, tables, time.Hour)

	c := lobbyClient("alice", l)
	l.Register(c)

	state := nextLobbyState(t, c)
	if state.Online != 1 || len(state.Tables) != 1 || state.Tables[0].ID != "t1" {
		t.Fatalf("state = %+v, want online 1 and table t1", state)
	}
}

func TestLobby_EmptyTablesAreAnArray(t *testing.T) {
	l, _ := startLobby(t)
	c := lobbyClient("alice", l)
	l.Register(c)

	data := <-c.send
	var raw struct {
		Payload struct {
			Tables json.RawMessage `json:"tables"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw.Payload.Tables) != "[]" {
		t.Fatalf("tables = %s, want []", raw.Payload.Tables)
	}
}

func TestLobby_SecondTabOfSamePlayerGetsStateWithoutBroadcast(t *testing.T) {
	l, _ := startLobby(t)
	tab1 := lobbyClient("alice", l)
	tab2 := lobbyClient("alice", l)
	l.Register(tab1)
	nextLobbyState(t, tab1)

	l.Register(tab2)
	if state := nextLobbyState(t, tab2); state.Online != 1 {
		t.Fatalf("second tab online = %d, want 1", state.Online)
	}
	expectSilence(t, tab1, 30*time.Millisecond)
}

func TestLobby_BroadcastsOnlineChanges(t *testing.T) {
	l, _ := startLobby(t)
	alice := lobbyClient("alice", l)
	l.Register(alice)
	nextLobbyState(t, alice)

	bob := lobbyClient("bob", l)
	l.Register(bob)
	if state := nextLobbyState(t, alice); state.Online != 2 {
		t.Fatalf("alice sees online %d after bob joined, want 2", state.Online)
	}
	nextLobbyState(t, bob)

	l.leave(bob)
	if state := nextLobbyState(t, alice); state.Online != 1 {
		t.Fatalf("alice sees online %d after bob left, want 1", state.Online)
	}
}

func TestLobby_BroadcastsOnlyWhenTablesChange(t *testing.T) {
	tables := &fakeTables{}
	l, _ := startLobbyWith(t, tables, 5*time.Millisecond)
	c := lobbyClient("alice", l)
	l.Register(c)
	nextLobbyState(t, c)

	expectSilence(t, c, 40*time.Millisecond)

	tables.set(RoomInfo{ID: "t1", Players: 1, MaxPlayers: 2})
	if state := nextLobbyState(t, c); len(state.Tables) != 1 {
		t.Fatalf("tables after change = %d, want 1", len(state.Tables))
	}
	expectSilence(t, c, 40*time.Millisecond)
}

func TestLobby_DropsClientThatDoesNotRead(t *testing.T) {
	tables := &fakeTables{}
	l, _ := startLobbyWith(t, tables, time.Millisecond)
	slow := &Client{id: "slow", name: "slow", owner: l, send: make(chan []byte, 1)}
	l.Register(slow)
	l.Online()

	tables.set(RoomInfo{ID: "t1"})
	deadline := time.Now().Add(time.Second)
	for !isClosed(slow) {
		if time.Now().After(deadline) {
			t.Fatal("slow client was never dropped")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := l.Online(); got != 0 {
		t.Fatalf("online = %d after dropping the only client, want 0", got)
	}
}

func TestLobby_OnlineOfReportsOnlyConnectedPlayers(t *testing.T) {
	l, _ := startLobby(t)
	l.Register(lobbyClient("alice", l))

	in := l.OnlineOf([]string{"alice", "bob"})
	if !in["alice"] || in["bob"] {
		t.Fatalf("online of = %v, want only alice", in)
	}
}

func TestLobby_NotifyReachesEveryTabOfOnePlayer(t *testing.T) {
	l, _ := startLobby(t)
	tab1, tab2 := lobbyClient("alice", l), lobbyClient("alice", l)
	bob := lobbyClient("bob", l)
	for _, c := range []*Client{tab1, tab2, bob} {
		l.Register(c)
	}
	l.Online()
	for _, c := range []*Client{tab1, tab2, bob} {
		for len(c.send) > 0 {
			<-c.send
		}
	}

	l.Notify("alice", []byte(`{"type":"friends_changed"}`))
	l.Online()

	for _, c := range []*Client{tab1, tab2} {
		select {
		case data := <-c.send:
			if string(data) != `{"type":"friends_changed"}` {
				t.Fatalf("got %s", data)
			}
		default:
			t.Fatal("a tab of alice got no notice")
		}
	}
	if len(bob.send) != 0 {
		t.Fatal("bob got alice's notice")
	}
}

func TestHub_PresenceLobbyTableOffline(t *testing.T) {
	h := NewHub(nil, nil, nil)
	inLobby, atTable, away := uuid.New(), uuid.New(), uuid.New()

	h.lobby.Register(lobbyClient(inLobby.String(), h.lobby))
	room := h.CreateRoom(RoomSettings{Mode: game.ModePodkidnoy, Deck: game.Deck36, MaxPlayers: 2, CreatedBy: "x"})
	defer room.Close()
	room.RegisterClient(createTestClient(atTable.String(), "T", room))
	deadline := time.Now().Add(time.Second)
	for !room.Holds(atTable.String()) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	got := h.Presence([]uuid.UUID{inLobby, atTable, away})
	want := map[uuid.UUID]model.Presence{inLobby: model.PresenceOnline, atTable: model.PresencePlaying, away: model.PresenceOffline}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("presence of %s = %q, want %q", id, got[id], p)
		}
	}
}
