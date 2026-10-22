package model

import "errors"

const TxPurchase TransactionKind = "purchase"

var (
	ErrUnknownItem  = errors.New("unknown shop item")
	ErrAlreadyOwned = errors.New("item already owned")
	ErrNotOwned     = errors.New("item not owned")
)

type Look struct {
	Theme string
	Felt  string
	Back  string
}
