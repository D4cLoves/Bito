package service

import (
	"context"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
)

type InviteTables interface {
	InviteTable(roomID string) (model.InviteTable, bool)
	Invite(roomID, from, to string) (model.InviteTable, time.Time, error)
	Decline(roomID, playerID, username string) bool
}

type InviteNotifier interface {
	SendInvite(to uuid.UUID, inv model.TableInvite)
}

type BalanceSource interface {
	Balance(ctx context.Context, userID string) (int64, error)
}

type InviteService struct {
	tables    InviteTables
	relations RelationSource
	presence  PresenceSource
	users     PlayerUsers
	balances  BalanceSource
	notify    InviteNotifier
}

func NewInviteService(tables InviteTables, relations RelationSource, presence PresenceSource, users PlayerUsers, balances BalanceSource, notify InviteNotifier) *InviteService {
	return &InviteService{
		tables:    tables,
		relations: relations,
		presence:  presence,
		users:     users,
		balances:  balances,
		notify:    notify,
	}
}

func (s *InviteService) Invite(ctx context.Context, me uuid.UUID, roomID string, friend uuid.UUID) (model.TableInvite, error) {
	relation, err := s.relations.Between(ctx, me, friend)
	if err != nil {
		return model.TableInvite{}, err
	}
	if relation != model.RelationFriend {
		return model.TableInvite{}, model.ErrNotFriends
	}
	if s.presence.Presence([]uuid.UUID{friend})[friend] != model.PresenceOnline {
		return model.TableInvite{}, model.ErrFriendNotInLobby
	}

	table, ok := s.tables.InviteTable(roomID)
	if !ok {
		return model.TableInvite{}, model.ErrTableNotFound
	}
	balance, err := s.balances.Balance(ctx, friend.String())
	if err != nil {
		return model.TableInvite{}, err
	}
	if balance < table.Stake {
		return model.TableInvite{}, model.ErrFriendNoFunds
	}

	inviter, err := s.users.GetByID(ctx, me)
	if err != nil {
		return model.TableInvite{}, err
	}

	table, expiresAt, err := s.tables.Invite(roomID, me.String(), friend.String())
	if err != nil {
		return model.TableInvite{}, err
	}

	inv := model.TableInvite{Table: table, FromID: me, FromName: inviter.Name, ExpiresAt: expiresAt}
	s.notify.SendInvite(friend, inv)
	return inv, nil
}

func (s *InviteService) Decline(ctx context.Context, me uuid.UUID, roomID string) error {
	user, err := s.users.GetByID(ctx, me)
	if err != nil {
		return err
	}
	if !s.tables.Decline(roomID, me.String(), user.Name) {
		return model.ErrInviteNotFound
	}
	return nil
}
