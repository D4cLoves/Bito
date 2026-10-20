package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFriendSelf            = errors.New("cannot befriend yourself")
	ErrAlreadyFriends        = errors.New("already friends")
	ErrFriendRequestExists   = errors.New("friend request already sent")
	ErrFriendRequestNotFound = errors.New("friend request not found")
)

type FriendState string

const (
	FriendPending  FriendState = "pending"
	FriendAccepted FriendState = "accepted"
)

type Presence string

const (
	PresenceOffline Presence = "offline"
	PresenceOnline  Presence = "online"
	PresencePlaying Presence = "playing"
)

type FriendLink struct {
	UserID   uuid.UUID
	Username string
	State    FriendState
	Outgoing bool
	Since    time.Time
	Presence Presence
}

type FriendsOverview struct {
	Friends  []FriendLink
	Incoming []FriendLink
	Outgoing []FriendLink
}
