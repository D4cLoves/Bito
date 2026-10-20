package ws

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bito/internal/game"
	"bito/internal/model"

	"github.com/google/uuid"
)

func waitHolds(t *testing.T, room *Room, id string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !room.Holds(id) {
		if time.Now().After(deadline) {
			t.Fatalf("%s never took a seat", id)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func companyTable(t *testing.T, h *Hub, creator string, seats int) *Room {
	t.Helper()
	room, err := h.CreateTable(RoomSettings{Mode: game.ModePodkidnoy, Deck: game.Deck36, MaxPlayers: seats, Stake: 100, CreatedBy: creator})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(room.Close)
	return room
}

func TestRoom_InviteHoldsSeatWithoutClaimingPlayer(t *testing.T) {
	h := NewHub(nil, nil, nil)
	room := companyTable(t, h, "alice", 2)

	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if room.Occupancy() != 2 {
		t.Fatalf("occupancy = %d, want 2 (alice reserved, bob invited)", room.Occupancy())
	}
	if h.SeatedElsewhere("bob", "") {
		t.Fatal("an invite must not block bob from other tables")
	}
}

func TestRoom_StrangerCannotTakeInvitedSeat(t *testing.T) {
	h := NewHub(nil, nil, nil)
	room := companyTable(t, h, "alice", 2)
	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
		t.Fatal(err)
	}

	alice := createTestClient("alice", "Alice", room)
	room.RegisterClient(alice)
	waitHolds(t, room, "alice")

	eve := createTestClient("eve", "Eve", room)
	room.RegisterClient(eve)
	deadline := time.Now().Add(time.Second)
	for {
		if _, ok := lastOfType[ErrorPayload](t, eve, TypeError); ok {
			break
		}
		if room.Holds("eve") || time.Now().After(deadline) {
			t.Fatal("eve took the seat kept for bob")
		}
		time.Sleep(5 * time.Millisecond)
	}

	bob := createTestClient("bob", "Bob", room)
	room.RegisterClient(bob)
	waitHolds(t, room, "bob")
}

func TestHub_InviteRules(t *testing.T) {
	h := NewHub(nil, nil, nil)
	room := companyTable(t, h, "alice", 3)
	other := companyTable(t, h, "carol", 3)

	cases := []struct {
		name   string
		roomID string
		from   string
		to     string
		want   error
	}{
		{"unknown table", "nope", "alice", "bob", model.ErrTableNotFound},
		{"not my table", room.ID(), "mallory", "bob", model.ErrNotAtTable},
		{"friend owns another table", room.ID(), "alice", "carol", model.ErrFriendBusy},
		{"ok", room.ID(), "alice", "bob", nil},
		{"twice", room.ID(), "alice", "bob", model.ErrAlreadyInvited},
		{"last seat", room.ID(), "alice", "dave", nil},
		{"no seats left", room.ID(), "alice", "erin", model.ErrTableFull},
	}
	for _, tc := range cases {
		_, _, err := h.Invite(tc.roomID, tc.from, tc.to)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
	_ = other
}

func TestHub_DeclineFreesSeatAndTellsTheTable(t *testing.T) {
	h := NewHub(nil, nil, nil)
	room := companyTable(t, h, "alice", 2)
	alice := createTestClient("alice", "Alice", room)
	room.RegisterClient(alice)
	waitHolds(t, room, "alice")

	if _, _, err := h.Invite(room.ID(), "alice", "bob"); err != nil {
		t.Fatal(err)
	}
	if !h.Decline(room.ID(), "bob", "Bob") {
		t.Fatal("decline of a live invite returned false")
	}
	if h.Decline(room.ID(), "bob", "Bob") {
		t.Fatal("second decline should find nothing")
	}
	if room.Occupancy() != 1 {
		t.Fatalf("occupancy after decline = %d, want 1", room.Occupancy())
	}

	deadline := time.Now().Add(time.Second)
	for {
		if msg, ok := lastOfType[InviteDeclinedPayload](t, alice, TypeInviteDeclined); ok {
			if msg.Payload.Username != "Bob" {
				t.Fatalf("declined by %q", msg.Payload.Username)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("alice was not told about the decline")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHub_SendInviteReachesLobby(t *testing.T) {
	h := NewHub(nil, nil, nil)
	bobID := uuid.New()
	tab := lobbyClient(bobID.String(), h.lobby)
	h.lobby.Register(tab)
	h.lobby.Online()
	for len(tab.send) > 0 {
		<-tab.send
	}

	h.SendInvite(bobID, model.TableInvite{Table: model.InviteTable{ID: "t1", Stake: 100}, FromName: "Alice", ExpiresAt: time.Now()})
	h.lobby.Online()

	select {
	case data := <-tab.send:
		var msg Message[TableInvitePayload]
		if err := json.Unmarshal(data, &msg); err != nil || msg.Type != TypeTableInvite || msg.Payload.TableID != "t1" || msg.Payload.FromName != "Alice" {
			t.Fatalf("got %s (%v)", data, err)
		}
	default:
		t.Fatal("no table_invite in the lobby")
	}
}
