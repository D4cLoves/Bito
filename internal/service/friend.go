package service

import (
	"context"
	"strings"

	"bito/internal/model"

	"github.com/google/uuid"
)

type FriendRepository interface {
	Request(ctx context.Context, from, to uuid.UUID) (model.FriendState, error)
	Accept(ctx context.Context, me, from uuid.UUID) error
	Remove(ctx context.Context, me, other uuid.UUID) error
	List(ctx context.Context, me uuid.UUID) ([]model.FriendLink, error)
}

type UserFinder interface {
	GetByName(ctx context.Context, name string) (*model.User, error)
}

type PresenceSource interface {
	Presence(ids []uuid.UUID) map[uuid.UUID]model.Presence
}

type FriendNotifier interface {
	FriendsChanged(ids ...uuid.UUID)
}

type FriendService struct {
	repo     FriendRepository
	users    UserFinder
	presence PresenceSource
	notify   FriendNotifier
}

func NewFriendService(repo FriendRepository, users UserFinder, presence PresenceSource, notify FriendNotifier) *FriendService {
	return &FriendService{repo: repo, users: users, presence: presence, notify: notify}
}

func (s *FriendService) Overview(ctx context.Context, me uuid.UUID) (model.FriendsOverview, error) {
	links, err := s.repo.List(ctx, me)
	if err != nil {
		return model.FriendsOverview{}, err
	}

	out := model.FriendsOverview{
		Friends:  []model.FriendLink{},
		Incoming: []model.FriendLink{},
		Outgoing: []model.FriendLink{},
	}
	ids := make([]uuid.UUID, 0, len(links))
	for _, l := range links {
		switch {
		case l.State == model.FriendAccepted:
			out.Friends = append(out.Friends, l)
			ids = append(ids, l.UserID)
		case l.Outgoing:
			out.Outgoing = append(out.Outgoing, l)
		default:
			out.Incoming = append(out.Incoming, l)
		}
	}

	statuses := s.presence.Presence(ids)
	for i := range out.Friends {
		out.Friends[i].Presence = statuses[out.Friends[i].UserID]
		if out.Friends[i].Presence == "" {
			out.Friends[i].Presence = model.PresenceOffline
		}
	}
	return out, nil
}

func (s *FriendService) Request(ctx context.Context, me uuid.UUID, username string) (model.FriendState, error) {
	name := strings.TrimSpace(username)
	if name == "" {
		return "", model.ErrUserNotFound
	}
	target, err := s.users.GetByName(ctx, name)
	if err != nil {
		return "", err
	}
	if target.ID == me {
		return "", model.ErrFriendSelf
	}

	state, err := s.repo.Request(ctx, me, target.ID)
	if err != nil {
		return "", err
	}
	s.notify.FriendsChanged(target.ID)
	return state, nil
}

func (s *FriendService) Accept(ctx context.Context, me, from uuid.UUID) error {
	if err := s.repo.Accept(ctx, me, from); err != nil {
		return err
	}
	s.notify.FriendsChanged(from)
	return nil
}

func (s *FriendService) Remove(ctx context.Context, me, other uuid.UUID) error {
	if err := s.repo.Remove(ctx, me, other); err != nil {
		return err
	}
	s.notify.FriendsChanged(other)
	return nil
}
