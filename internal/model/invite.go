package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTableNotFound    = errors.New("table not found")
	ErrNotAtTable       = errors.New("inviter does not sit at this table")
	ErrTableStarted     = errors.New("game already started")
	ErrTableFull        = errors.New("no free seat at the table")
	ErrAlreadyInvited   = errors.New("player already invited to this table")
	ErrFriendBusy       = errors.New("friend sits at another table")
	ErrNotFriends       = errors.New("players are not friends")
	ErrFriendNotInLobby = errors.New("friend is not in the lobby")
	ErrFriendNoFunds    = errors.New("friend cannot afford the stake")
	ErrInviteNotFound   = errors.New("invite not found")
)

type InviteTable struct {
	ID         string
	Mode       string
	Deck       int
	MaxPlayers int
	Stake      int64
}

type TableInvite struct {
	Table     InviteTable
	FromID    uuid.UUID
	FromName  string
	ExpiresAt time.Time
}
