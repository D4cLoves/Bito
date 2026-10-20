package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
)

type fakeTables struct {
	table    model.InviteTable
	found    bool
	err      error
	invited  []string
	declined bool
}

func (f *fakeTables) InviteTable(string) (model.InviteTable, bool) { return f.table, f.found }

func (f *fakeTables) Invite(_, _, to string) (model.InviteTable, time.Time, error) {
	if f.err != nil {
		return model.InviteTable{}, time.Time{}, f.err
	}
	f.invited = append(f.invited, to)
	return f.table, time.Now().Add(time.Minute), nil
}

func (f *fakeTables) Decline(string, string, string) bool { return f.declined }

type fixedBalance int64

func (b fixedBalance) Balance(context.Context, string) (int64, error) { return int64(b), nil }

type fakeInviteNotifier struct{ sent []model.TableInvite }

func (f *fakeInviteNotifier) SendInvite(_ uuid.UUID, inv model.TableInvite) {
	f.sent = append(f.sent, inv)
}

type inviteSetup struct {
	me, bob  uuid.UUID
	tables   *fakeTables
	relation *fixedRelation
	presence fakePresence
	balance  fixedBalance
	notify   *fakeInviteNotifier
}

func newInviteSetup() *inviteSetup {
	me, bob := uuid.New(), uuid.New()
	return &inviteSetup{
		me:       me,
		bob:      bob,
		tables:   &fakeTables{table: model.InviteTable{ID: "t1", Stake: 500}, found: true},
		relation: &fixedRelation{relation: model.RelationFriend},
		presence: fakePresence{bob: model.PresenceOnline},
		balance:  2500,
		notify:   &fakeInviteNotifier{},
	}
}

func (s *inviteSetup) service() *InviteService {
	users := usersByID{s.me: {ID: s.me, Name: "Alice"}, s.bob: {ID: s.bob, Name: "Bob"}}
	return NewInviteService(s.tables, s.relation, s.presence, users, s.balance, s.notify)
}

func TestInviteService_SendsInviteToFriendInLobby(t *testing.T) {
	s := newInviteSetup()
	inv, err := s.service().Invite(context.Background(), s.me, "t1", s.bob)
	if err != nil {
		t.Fatal(err)
	}
	if inv.FromName != "Alice" || inv.Table.ID != "t1" {
		t.Fatalf("invite = %+v", inv)
	}
	if len(s.tables.invited) != 1 || s.tables.invited[0] != s.bob.String() {
		t.Fatalf("seat kept for %v", s.tables.invited)
	}
	if len(s.notify.sent) != 1 {
		t.Fatalf("notices sent = %d, want 1", len(s.notify.sent))
	}
}

func TestInviteService_Refusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*inviteSetup)
		want  error
	}{
		{"not friends", func(s *inviteSetup) { s.relation.relation = model.RelationOutgoing }, model.ErrNotFriends},
		{"friend at a table", func(s *inviteSetup) { s.presence[s.bob] = model.PresencePlaying }, model.ErrFriendNotInLobby},
		{"friend offline", func(s *inviteSetup) { delete(s.presence, s.bob) }, model.ErrFriendNotInLobby},
		{"no table", func(s *inviteSetup) { s.tables.found = false }, model.ErrTableNotFound},
		{"poor friend", func(s *inviteSetup) { s.balance = 100 }, model.ErrFriendNoFunds},
		{"table refuses", func(s *inviteSetup) { s.tables.err = model.ErrTableFull }, model.ErrTableFull},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newInviteSetup()
			tc.setup(s)
			if _, err := s.service().Invite(context.Background(), s.me, "t1", s.bob); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(s.notify.sent) != 0 {
				t.Fatal("refused invite was still delivered")
			}
		})
	}
}

func TestInviteService_Decline(t *testing.T) {
	s := newInviteSetup()
	if err := s.service().Decline(context.Background(), s.bob, "t1"); !errors.Is(err, model.ErrInviteNotFound) {
		t.Fatalf("decline without invite: err = %v", err)
	}
	s.tables.declined = true
	if err := s.service().Decline(context.Background(), s.bob, "t1"); err != nil {
		t.Fatalf("decline: %v", err)
	}
}
