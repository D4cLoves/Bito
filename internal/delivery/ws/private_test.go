package ws

import (
	"testing"
)

func privateTable(t *testing.T, h *Hub, creator string) *Room {
	t.Helper()
	room, err := h.CreateTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 4, CreatedBy: creator, Private: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(room.Close)
	return room
}

func TestPrivate_HiddenFromTheList(t *testing.T) {
	h := NewHub(nil, nil, nil)
	hidden := privateTable(t, h, "alice")
	open, err := h.CreateTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 3, CreatedBy: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	defer open.Close()

	list := h.ListOpenRooms()
	if len(list) != 1 || list[0].ID != open.ID() {
		t.Fatalf("list = %+v, want only the open table", list)
	}
	if !hidden.Info().Private {
		t.Fatal("info does not say the table is private")
	}
}

func TestPrivate_WhoMayEnter(t *testing.T) {
	h := NewHub(nil, nil, nil)
	room := privateTable(t, h, "alice")
	if _, _, err := h.Invite(room.ID(), "alice", "olga"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		player string
		key    string
		want   string
	}{
		{"creator keeps a seat", "alice", "", ""},
		{"invited friend", "olga", "", ""},
		{"stranger with the link", "eve", room.Key(), ""},
		{"stranger without a key", "mallory", "", errPrivateTable},
		{"stranger with a wrong key", "trudy", room.Key() + "x", errPrivateTable},
	}
	for _, tc := range cases {
		if got := h.admit(room, tc.player, tc.key); got != tc.want {
			t.Errorf("%s: admit = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPrivate_KeysAreLongAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		k := tableKey(true)
		if len(k) < 24 || seen[k] {
			t.Fatalf("weak or repeated key %q", k)
		}
		seen[k] = true
	}
	if tableKey(false) != "" {
		t.Fatal("an open table got a key")
	}
}

func TestPrivate_OpenTableNeedsNoKey(t *testing.T) {
	h := NewHub(nil, nil, nil)
	open, err := h.CreateTable(RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 3, CreatedBy: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	defer open.Close()
	if got := h.admit(open, "eve", ""); got != "" {
		t.Fatalf("open table refused a stranger: %q", got)
	}
}

func TestPrivate_RematchStaysPrivate(t *testing.T) {
	starter := &fakeStarter{}
	room := NewRoom("room-private-rematch", RoomSettings{Mode: defaultDuelMode, Deck: defaultDuelDeck, MaxPlayers: 2, Stake: 100, Private: true}, richBank())
	room.settleRetryDelay = 0
	room.rematch = starter.start
	go room.Run()
	defer room.Close()

	alice := createTestClient("p1", "Alice", room)
	bob := createTestClient("p2", "Bob", room)
	room.RegisterClient(alice)
	room.RegisterClient(bob)
	waitFor(t, func() bool { return room.Started() })
	send(room, alice, TypeSurrender)
	waitFor(t, func() bool { _, ok := lastOfType[GameOverPayload](t, bob, TypeGameOver); return ok })

	send(room, alice, TypeRematch)
	send(room, bob, TypeRematch)
	waitFor(t, func() bool { return starter.count() == 1 })
	if !starter.calls[0].settings.Private {
		t.Fatal("the rematch of a private table became public")
	}
}
