package ws

import (
	"encoding/json"
	"testing"
)

func watcherOf(id string, room *Room) *Client {
	c := createTestClient(id, "Watcher "+id, room)
	c.watch = true
	return c
}

func TestWatch_SeesTheTableButNoHands(t *testing.T) {
	room := NewRoom("room-watch", testSettings(2), nil)
	go room.Run()
	defer room.Close()

	eve := watcherOf("w1", room)
	room.RegisterClient(eve)
	room.RegisterClient(createTestClient("p1", "Alice", room))
	room.RegisterClient(createTestClient("p2", "Bob", room))
	waitFor(t, func() bool { return room.Started() })

	var st Message[GameStatePayload]
	waitFor(t, func() bool {
		got, ok := lastOfType[GameStatePayload](t, eve, TypeGameState)
		st = got
		return ok
	})
	if len(st.Payload.MyHand) != 0 {
		t.Fatalf("watcher got a hand: %v", st.Payload.MyHand)
	}
	if len(st.Payload.Players) != 2 || st.Payload.Players[0].HandCount == 0 {
		t.Fatalf("watcher does not see the players: %+v", st.Payload.Players)
	}
	if room.PlayersCount() != 2 || room.Watchers() != 1 {
		t.Fatalf("players = %d, watchers = %d", room.PlayersCount(), room.Watchers())
	}
}

func TestWatch_CannotMoveOrChat(t *testing.T) {
	room := NewRoom("room-watch-act", testSettings(2), nil)
	go room.Run()
	defer room.Close()

	alice := createTestClient("p1", "Alice", room)
	eve := watcherOf("w1", room)
	room.RegisterClient(alice)
	room.RegisterClient(createTestClient("p2", "Bob", room))
	room.RegisterClient(eve)
	waitFor(t, func() bool { return room.Watchers() == 1 })
	drain(alice)

	send(room, eve, TypeTake)
	chat, _ := json.Marshal(map[string]any{"type": TypeChat, "payload": map[string]string{"message": "бей пиковой"}})
	room.incoming <- IncomingMessage{Client: eve, Data: chat}

	waitFor(t, func() bool {
		e, ok := lastOfType[ErrorPayload](t, eve, TypeError)
		return ok && e.Payload.Message == errWatcherAction
	})
	if _, ok := lastOfType[ChatPayload](t, alice, TypeChat); ok {
		t.Fatal("a watcher's message reached the players")
	}
}

func TestWatch_CountFollowsComingAndGoing(t *testing.T) {
	room := NewRoom("room-watch-count", testSettings(3), nil)
	go room.Run()
	defer room.Close()

	alice := createTestClient("p1", "Alice", room)
	room.RegisterClient(alice)
	eve := watcherOf("w1", room)
	room.RegisterClient(eve)
	waitFor(t, func() bool {
		w, ok := lastOfType[WatchersPayload](t, alice, TypeWatchers)
		return ok && w.Payload.Count == 1
	})
	if room.Info().Watchers != 1 {
		t.Fatalf("info watchers = %d", room.Info().Watchers)
	}

	room.unregister <- eve
	waitFor(t, func() bool {
		w, ok := lastOfType[WatchersPayload](t, alice, TypeWatchers)
		return ok && w.Payload.Count == 0
	})
	if room.Occupancy() != 1 {
		t.Fatalf("occupancy = %d, a watcher took a seat", room.Occupancy())
	}
}

func TestWatch_Limit(t *testing.T) {
	room := NewRoom("room-watch-limit", testSettings(2), nil)
	go room.Run()
	defer room.Close()

	for i := range maxWatchers {
		room.RegisterClient(watcherOf(string(rune('a'+i)), room))
	}
	late := watcherOf("late", room)
	room.RegisterClient(late)
	waitFor(t, func() bool {
		e, ok := lastOfType[ErrorPayload](t, late, TypeError)
		return ok && e.Payload.Message == errTooManyWatchers
	})
	if room.Watchers() != maxWatchers {
		t.Fatalf("watchers = %d", room.Watchers())
	}
}

func TestWatch_SecondTabReplacesTheFirst(t *testing.T) {
	room := NewRoom("room-watch-tabs", testSettings(2), nil)
	go room.Run()
	defer room.Close()

	first := watcherOf("w1", room)
	second := watcherOf("w1", room)
	room.RegisterClient(first)
	room.RegisterClient(second)
	waitFor(t, func() bool { return isClosed(first) })
	if room.Watchers() != 1 {
		t.Fatalf("watchers = %d", room.Watchers())
	}
}

func TestWatch_LateWatcherSeesTheResult(t *testing.T) {
	starter := &fakeStarter{}
	room, _, _ := finishedDuel(t, richBank(), starter)

	eve := watcherOf("w1", room)
	room.RegisterClient(eve)
	waitFor(t, func() bool {
		_, ok := lastOfType[GameOverPayload](t, eve, TypeGameOver)
		return ok
	})
}

func TestWatch_WatchersDoNotVoteForRematch(t *testing.T) {
	starter := &fakeStarter{}
	room, alice, bob := finishedDuel(t, richBank(), starter)
	room.RegisterClient(watcherOf("w1", room))
	waitFor(t, func() bool { return room.Watchers() == 1 })

	send(room, alice, TypeRematch)
	send(room, bob, TypeRematch)
	waitFor(t, func() bool { return starter.count() == 1 })
	if got := starter.calls[0].players; len(got) != 2 {
		t.Fatalf("rematch players = %v", got)
	}
}

func TestWatch_ClosingTheTableClosesWatchers(t *testing.T) {
	room := NewRoom("room-watch-close", testSettings(2), nil)
	go room.Run()

	alice := createTestClient("p1", "Alice", room)
	eve := watcherOf("w1", room)
	room.RegisterClient(alice)
	room.RegisterClient(eve)
	waitFor(t, func() bool { return room.Watchers() == 1 })

	room.unregister <- alice
	waitFor(t, func() bool { return isClosed(eve) })
}

func TestHub_LiveTablesAreStartedAndPublic(t *testing.T) {
	h := NewHub(nil, nil, nil)
	start := func(private bool, players ...string) *Room {
		room, err := h.CreateTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 2, CreatedBy: players[0], Private: private})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(room.Close)
		for _, id := range players {
			room.RegisterClient(createTestClient(id, id, room))
		}
		return room
	}

	live := start(false, "a", "b")
	hidden := start(true, "c", "d")
	waiting := start(false, "e")
	waitFor(t, func() bool { return live.Started() && hidden.Started() })

	list := h.ListLiveRooms()
	if len(list) != 1 || list[0].ID != live.ID() {
		t.Fatalf("live = %+v, want only %s (not private %s, not waiting %s)", list, live.ID(), hidden.ID(), waiting.ID())
	}
}
