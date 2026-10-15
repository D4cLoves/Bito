package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrPendingNotFound   = errors.New("registration request not found or expired")
	ErrCodeMismatch      = errors.New("invalid verification code")
	ErrCodeAttempts      = errors.New("too many invalid verification codes")
	ErrSessionExpired    = errors.New("session expired or revoked")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrTopUpNotAllowed   = errors.New("top-up not allowed")
)

type InsufficientFundsError struct {
	UserID uuid.UUID
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("%s: user %s", ErrInsufficientFunds, e.UserID)
}

func (e *InsufficientFundsError) Unwrap() error {
	return ErrInsufficientFunds
}

type TopUpCooldownError struct {
	NextAt time.Time
}

func (e *TopUpCooldownError) Error() string {
	return fmt.Sprintf("%s: next top-up at %s", ErrTopUpNotAllowed, e.NextAt.Format(time.RFC3339))
}

func (e *TopUpCooldownError) Unwrap() error {
	return ErrTopUpNotAllowed
}
