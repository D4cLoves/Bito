package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
)

type fakeFriendRepo struct {
	links     []model.FriendLink
	requested [][2]uuid.UUID
	state     model.FriendState
	err       error
}

func (f *fakeFriendRepo) Request(_ context.Context, from, to uuid.UUID) (model.FriendState, error) {
	f.requested = append(f.requested, [2]uuid.UUID{from, to})
	return f.state, f.err
}

func (f *fakeFriendRepo) Accept(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
func (f *fakeFriendRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
func (f *fakeFriendRepo) List(context.Context, uuid.UUID) ([]model.FriendLink, error) {
	return f.links, f.err
}

type fakeUsers map[string]*model.User

func (f fakeUsers) GetByName(_ context.Context, name string) (*model.User, error) {
	if u, ok := f[strings.ToLower(name)]; ok {
		return u, nil
	}
	return nil, model.ErrUserNotFound
}

type fakePresence map[uuid.UUID]model.Presence

func (f fakePresence) Presence([]uuid.UUID) map[uuid.UUID]model.Presence { return f }

type fakeNotifier struct{ got []uuid.UUID }

func (f *fakeNotifier) FriendsChanged(ids ...uuid.UUID) { f.got = append(f.got, ids...) }

func TestFriendService_RequestFindsUserByName(t *testing.T) {
	me, bob := uuid.New(), uuid.New()
	repo := &fakeFriendRepo{state: model.FriendPending}
	notify := &fakeNotifier{}
	s := NewFriendService(repo, fakeUsers{"bob": {ID: bob, Name: "Bob"}}, fakePresence{}, notify)

	state, err := s.Request(context.Background(), me, "  BOB ")
	if err != nil || state != model.FriendPending {
		t.Fatalf("request = %q, %v", state, err)
	}
	if len(repo.requested) != 1 || repo.requested[0] != [2]uuid.UUID{me, bob} {
		t.Fatalf("repo got %v, want me→bob", repo.requested)
	}
	if len(notify.got) != 1 || notify.got[0] != bob {
		t.Fatalf("notified %v, want bob", notify.got)
	}
}

func TestFriendService_RequestRejectsSelfAndUnknown(t *testing.T) {
	me := uuid.New()
	repo := &fakeFriendRepo{}
	notify := &fakeNotifier{}
	s := NewFriendService(repo, fakeUsers{"me": {ID: me, Name: "Me"}}, fakePresence{}, notify)

	if _, err := s.Request(context.Background(), me, "me"); !errors.Is(err, model.ErrFriendSelf) {
		t.Fatalf("self err = %v", err)
	}
	if _, err := s.Request(context.Background(), me, "ghost"); !errors.Is(err, model.ErrUserNotFound) {
		t.Fatalf("unknown err = %v", err)
	}
	if _, err := s.Request(context.Background(), me, "   "); !errors.Is(err, model.ErrUserNotFound) {
		t.Fatalf("blank err = %v", err)
	}
	if len(repo.requested) != 0 || len(notify.got) != 0 {
		t.Fatalf("repo or notifier touched on a rejected request")
	}
}

func TestFriendService_FailedRequestDoesNotNotify(t *testing.T) {
	bob := uuid.New()
	notify := &fakeNotifier{}
	repo := &fakeFriendRepo{err: model.ErrFriendRequestExists}
	s := NewFriendService(repo, fakeUsers{"bob": {ID: bob}}, fakePresence{}, notify)

	if _, err := s.Request(context.Background(), uuid.New(), "bob"); !errors.Is(err, model.ErrFriendRequestExists) {
		t.Fatalf("err = %v", err)
	}
	if len(notify.got) != 0 {
		t.Fatalf("notified on failure: %v", notify.got)
	}
}

func TestFriendService_OverviewSplitsAndAddsPresence(t *testing.T) {
	friend, gone, in, out := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repo := &fakeFriendRepo{links: []model.FriendLink{
		{UserID: friend, State: model.FriendAccepted, Since: time.Now()},
		{UserID: gone, State: model.FriendAccepted},
		{UserID: in, State: model.FriendPending},
		{UserID: out, State: model.FriendPending, Outgoing: true},
	}}
	s := NewFriendService(repo, fakeUsers{}, fakePresence{friend: model.PresencePlaying}, &fakeNotifier{})

	o, err := s.Overview(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Friends) != 2 || len(o.Incoming) != 1 || len(o.Outgoing) != 1 {
		t.Fatalf("overview = %+v", o)
	}
	if o.Friends[0].Presence != model.PresencePlaying || o.Friends[1].Presence != model.PresenceOffline {
		t.Fatalf("presence = %q, %q", o.Friends[0].Presence, o.Friends[1].Presence)
	}
	if o.Incoming[0].UserID != in || o.Outgoing[0].UserID != out {
		t.Fatalf("requests split wrong: %+v", o)
	}
}
