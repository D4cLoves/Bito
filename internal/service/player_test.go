package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
)

type usersByID map[uuid.UUID]*model.User

func (u usersByID) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if user, ok := u[id]; ok {
		return user, nil
	}
	return nil, model.ErrUserNotFound
}

type fixedRelation struct {
	relation model.Relation
	calls    int
}

func (f *fixedRelation) Between(context.Context, uuid.UUID, uuid.UUID) (model.Relation, error) {
	f.calls++
	return f.relation, nil
}

type fakeHistory struct {
	games []model.GameHistoryEntry
	limit int
}

func (f *fakeHistory) History(_ context.Context, _ uuid.UUID, limit int) ([]model.GameHistoryEntry, error) {
	f.limit = limit
	return f.games, nil
}

func TestPlayerService_FriendSeesPresence(t *testing.T) {
	me, bob := uuid.New(), uuid.New()
	users := usersByID{bob: {ID: bob, Name: "Bob", CreatedAt: time.Now(), Stats: &model.UserStats{Wins: 3, Losses: 1, TotalGames: 4}}}
	history := &fakeHistory{games: []model.GameHistoryEntry{{ID: uuid.New()}}}
	s := NewPlayerService(users, &fixedRelation{relation: model.RelationFriend}, history, fakePresence{bob: model.PresencePlaying})

	p, err := s.Profile(context.Background(), me, bob)
	if err != nil {
		t.Fatal(err)
	}
	if p.Username != "Bob" || p.Stats.Wins != 3 || p.Relation != model.RelationFriend || p.Presence != model.PresencePlaying {
		t.Fatalf("profile = %+v", p)
	}
	if len(p.Games) != 1 || history.limit != profileGames {
		t.Fatalf("games = %d, limit = %d", len(p.Games), history.limit)
	}
}

func TestPlayerService_StrangerHidesPresence(t *testing.T) {
	me, bob := uuid.New(), uuid.New()
	s := NewPlayerService(usersByID{bob: {ID: bob, Name: "Bob"}}, &fixedRelation{relation: model.RelationNone}, &fakeHistory{}, fakePresence{bob: model.PresenceOnline})

	p, err := s.Profile(context.Background(), me, bob)
	if err != nil {
		t.Fatal(err)
	}
	if p.Presence != "" || p.Relation != model.RelationNone {
		t.Fatalf("stranger profile leaks presence: %+v", p)
	}
}

func TestPlayerService_OwnProfileIsSelf(t *testing.T) {
	me := uuid.New()
	relations := &fixedRelation{relation: model.RelationFriend}
	s := NewPlayerService(usersByID{me: {ID: me, Name: "Me"}}, relations, &fakeHistory{}, fakePresence{})

	p, err := s.Profile(context.Background(), me, me)
	if err != nil {
		t.Fatal(err)
	}
	if p.Relation != model.RelationSelf || relations.calls != 0 {
		t.Fatalf("relation = %q, relation lookups = %d", p.Relation, relations.calls)
	}
}

func TestPlayerService_UnknownPlayer(t *testing.T) {
	s := NewPlayerService(usersByID{}, &fixedRelation{}, &fakeHistory{}, fakePresence{})
	if _, err := s.Profile(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, model.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}
