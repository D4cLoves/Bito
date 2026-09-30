package model

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrPendingNotFound   = errors.New("registration request not found or expired")
	ErrCodeMismatch      = errors.New("invalid verification code")
)
