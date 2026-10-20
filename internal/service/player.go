package service

import (
	"context"

	"bito/internal/model"

	"github.com/google/uuid"
)

const profileGames = 10

type PlayerUsers interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type RelationSource interface {
	Between(ctx context.Context, me, other uuid.UUID) (model.Relation, error)
}

type PlayerService struct {
	users     PlayerUsers
	relations RelationSource
	games     GameRepository
	presence  PresenceSource
}

func NewPlayerService(users PlayerUsers, relations RelationSource, games GameRepository, presence PresenceSource) *PlayerService {
	return &PlayerService{users: users, relations: relations, games: games, presence: presence}
}

func (s *PlayerService) Profile(ctx context.Context, me, id uuid.UUID) (model.PlayerProfile, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return model.PlayerProfile{}, err
	}

	relation := model.RelationSelf
	if id != me {
		if relation, err = s.relations.Between(ctx, me, id); err != nil {
			return model.PlayerProfile{}, err
		}
	}

	games, err := s.games.History(ctx, id, profileGames)
	if err != nil {
		return model.PlayerProfile{}, err
	}

	p := model.PlayerProfile{
		ID:        user.ID,
		Username:  user.Name,
		CreatedAt: user.CreatedAt,
		Relation:  relation,
		Games:     games,
	}
	if user.Stats != nil {
		p.Stats = *user.Stats
	}
	if relation == model.RelationFriend {
		p.Presence = s.presence.Presence([]uuid.UUID{id})[id]
		if p.Presence == "" {
			p.Presence = model.PresenceOffline
		}
	}
	return p, nil
}
