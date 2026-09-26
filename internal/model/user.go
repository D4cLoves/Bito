package model

import (
	"time"

	"github.com/google/uuid"
)

type OAuthProvider string

const (
	ProviderGoogle   OAuthProvider = "google"
	ProviderVK       OAuthProvider = "vk"
	ProviderTelegram OAuthProvider = "telegram"
)

type User struct {
	ID           uuid.UUID      `json:"id"`
	Email        string         `json:"email"`
	Name         string         `json:"name"`
	PasswordHash string         `json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Stats        *UserStats     `json:"stats,omitempty"`
	Wallet       *UserWallet    `json:"wallet,omitempty"`
	Accounts     []OAuthAccount `json:"accounts,omitempty"`
}

type UserStats struct {
	UserID     uuid.UUID `json:"-"`
	Elo        int       `json:"elo"`
	Wins       int       `json:"wins"`
	Losses     int       `json:"losses"`
	TotalGames int       `json:"totalGames"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type UserWallet struct {
	UserID    uuid.UUID `json:"-"`
	Balance   int64     `json:"balance"`
	Bonus     int64     `json:"bonus"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type OAuthAccount struct {
	ID             uuid.UUID     `json:"id"`
	UserID         uuid.UUID     `json:"-"`
	Provider       OAuthProvider `json:"provider"`
	ProviderUserID string        `json:"providerUserId"`
	CreatedAt      time.Time     `json:"createdAt"`
}
