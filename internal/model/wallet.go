package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionKind string

const (
	TxSignupBonus TransactionKind = "signup_bonus"
	TxOpening     TransactionKind = "opening"
	TxStakeLock   TransactionKind = "stake_lock"
	TxStakeRefund TransactionKind = "stake_refund"
	TxWinPayout   TransactionKind = "win_payout"
	TxTopUp       TransactionKind = "top_up"
)

type WalletTransaction struct {
	ID           int64           `json:"id"`
	Amount       int64           `json:"amount"`
	Kind         TransactionKind `json:"kind"`
	GameID       *uuid.UUID      `json:"gameId,omitempty"`
	BalanceAfter int64           `json:"balanceAfter"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type GameRecord struct {
	ID    uuid.UUID
	Mode  string
	Deck  int
	Stake int64
}

type GameHistoryEntry struct {
	ID         uuid.UUID `json:"id"`
	FinishedAt time.Time `json:"finishedAt"`
	Mode       string    `json:"mode"`
	Deck       int       `json:"deck"`
	Players    int       `json:"players"`
	Stake      int64     `json:"stake"`
	Place      int       `json:"place"`
	Payout     int64     `json:"payout"`
	Outcome    string    `json:"outcome"`
}
