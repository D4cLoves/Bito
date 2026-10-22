package ws

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type rematchCall struct {
	settings RoomSettings
	players  []string
}

type fakeStarter struct {
	mu    sync.Mutex
	calls []rematchCall
	err   error
}

func (f *fakeStarter) start(s RoomSettings, players []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, rematchCall{settings: s, players: players})
	return "next-table", f.err
}

func (f *fakeStarter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func finishedDuel(t *testing.T, bank *fakeBank, starter *fakeStarter) (*Room, *Client, *Client) {
	t.Helper()
	room := NewRoom("room-rematch", stakedSettings(100), bank)
	room.settleRetryDelay = time.Millisecond
	room.rematch = starter.start
	go room.Run()
	t.Cleanup(room.Close)

	alice := createTestClient("p1", "Alice", room)
	bob := createTestClient("p2", "Bob", room)
	room.RegisterClient(alice)
	room.RegisterClient(bob)
	waitFor(t, func() bool { return room.Started() })

	send(room, alice, TypeSurrender)
	waitFor(t, func() bool { _, ok := lastOfType[GameOverPayload](t, bob, TypeGameOver); return ok })
	drain(alice)
	return room, alice, bob
}

func send(room *Room, c *Client, msgType string) {
	data, _ := json.Marshal(map[string]string{"type": msgType})
	room.incoming <- IncomingMessage{Client: c, Data: data}
}

func drain(c *Client) {
	for len(c.send) > 0 {
		<-c.send
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached in time")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func richBank() *fakeBank {
	return &fakeBank{balances: map[string]int64{"p1": 1000, "p2": 1000}}
}

func TestRematch_StartsWhenEveryoneAgrees(t *testing.T) {
	starter := &fakeStarter{}
	room, alice, bob := finishedDuel(t, richBank(), starter)

	send(room, alice, TypeRematch)
	waitFor(t, func() bool {
		st, ok := lastOfType[RematchStatePayload](t, bob, TypeRematchState)
		return ok && len(st.Payload.Votes) == 1 && st.Payload.Needed == 2
	})
	if starter.count() != 0 {
		t.Fatal("rematch started before the second vote")
	}

	send(room, bob, TypeRematch)
	waitFor(t, func() bool {
		start, ok := lastOfType[RematchStartPayload](t, alice, TypeRematchStart)
		return ok && start.Payload.TableID == "next-table"
	})

	call := starter.calls[0]
	if len(call.players) != 2 || call.settings.Stake != 100 || call.settings.MaxPlayers != 2 {
		t.Fatalf("rematch call = %+v", call)
	}
}

func TestRematch_DeclineCancelsForEveryone(t *testing.T) {
	starter := &fakeStarter{}
	room, alice, bob := finishedDuel(t, richBank(), starter)

	send(room, alice, TypeRematch)
	send(room, bob, TypeRematchNo)
	waitFor(t, func() bool {
		c, ok := lastOfType[RematchCancelledPayload](t, alice, TypeRematchCancelled)
		return ok && c.Payload.Reason == "Bob не хочет реванш"
	})

	send(room, alice, TypeRematch)
	waitFor(t, func() bool {
		e, ok := lastOfType[ErrorPayload](t, alice, TypeError)
		return ok && e.Payload.Message == errRematchClosed
	})
	if starter.count() != 0 {
		t.Fatal("a declined rematch still started")
	}
}

func TestRematch_OpponentLeavingCancels(t *testing.T) {
	starter := &fakeStarter{}
	room, alice, bob := finishedDuel(t, richBank(), starter)

	send(room, alice, TypeRematch)
	room.unregister <- bob
	waitFor(t, func() bool {
		_, ok := lastOfType[RematchCancelledPayload](t, alice, TypeRematchCancelled)
		return ok
	})
	if starter.count() != 0 {
		t.Fatal("rematch started with one player")
	}
}

func TestRematch_RefusedWhenSomeoneCannotPay(t *testing.T) {
	starter := &fakeStarter{}
	bank := &fakeBank{balances: map[string]int64{"p1": 1000, "p2": 50}}
	room, alice, bob := finishedDuel(t, bank, starter)

	send(room, alice, TypeRematch)
	send(room, bob, TypeRematch)
	waitFor(t, func() bool {
		c, ok := lastOfType[RematchCancelledPayload](t, alice, TypeRematchCancelled)
		return ok && c.Payload.Reason == "Bob: не хватает фишек на ставку"
	})
	if starter.count() != 0 {
		t.Fatal("rematch started although Bob cannot pay")
	}
}

func TestRematch_StarterFailureCancels(t *testing.T) {
	starter := &fakeStarter{err: errors.New("busy")}
	room, alice, bob := finishedDuel(t, richBank(), starter)

	send(room, alice, TypeRematch)
	send(room, bob, TypeRematch)
	waitFor(t, func() bool {
		c, ok := lastOfType[RematchCancelledPayload](t, bob, TypeRematchCancelled)
		return ok && c.Payload.Reason == errRematchFailed
	})
}

func TestRematch_NotBeforeTheEnd(t *testing.T) {
	room := NewRoom("room-early", stakedSettings(100), richBank())
	room.rematch = (&fakeStarter{}).start
	go room.Run()
	defer room.Close()

	alice := createTestClient("p1", "Alice", room)
	room.RegisterClient(alice)
	send(room, alice, TypeRematch)
	waitFor(t, func() bool {
		e, ok := lastOfType[ErrorPayload](t, alice, TypeError)
		return ok && e.Payload.Message == errRematchTooEarly
	})
}

func TestHub_RematchTableKeepsSeatsForPlayers(t *testing.T) {
	h := NewHub(nil, nil, nil)
	id, err := h.rematchTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 2, Stake: 100, CreatedBy: "a"}, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	room, ok := h.GetRoom(id)
	if !ok {
		t.Fatal("rematch table not registered in the hub")
	}
	defer room.Close()
	if !room.Reserved("a") || !room.Reserved("b") || room.Occupancy() != 2 {
		t.Fatalf("seats not kept: a=%v b=%v occupancy=%d", room.Reserved("a"), room.Reserved("b"), room.Occupancy())
	}

	if _, err = h.rematchTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 2, CreatedBy: "a"}, []string{"a", "c"}); !errors.Is(err, ErrSeatedElsewhere) {
		t.Fatalf("second rematch for a busy player: err = %v", err)
	}
}
