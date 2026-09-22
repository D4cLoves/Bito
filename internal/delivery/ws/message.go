package ws

import (
	"bito/internal/game"
	"encoding/json"
)

const (
	TypeAttack    = "attack"
	TypeDefend    = "defend"
	TypeTransfer  = "transfer"
	TypeTake      = "take"
	TypePass      = "pass"
	TypeSurrender = "surrender"
	TypeChat      = "chat"
)

const (
	TypeGameState          = "game_state"
	TypeGameOver           = "game_over"
	TypeError              = "error"
	TypeTimerTick          = "timer_tick"
	TypePlayerDisconnected = "player_disconnected"
	TypePlayerReconnected  = "player_reconnected"
)

type Message[T any] struct {
	Type    string `json:"type"`
	Payload T      `json:"payload,omitempty"`
}

type AttackPayload struct {
	AttackCard game.Card `json:"attackCard"`
}

type DefendPayload struct {
	AttackCard game.Card `json:"attackCard"`
	DefendCard game.Card `json:"defendCard"`
}

type TransferPayload struct {
	Card game.Card `json:"card"`
}

type ChatPayload struct {
	SenderID   string `json:"senderId,omitempty"`
	SenderName string `json:"senderName,omitempty"`
	Message    string `json:"message"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type PlayerStatusPayload struct {
	PlayerID string `json:"playerId"`
}

type TimerTickPayload struct {
	PlayerID    string `json:"playerId"`
	SecondsLeft int    `json:"secondsLeft"`
}

type GameOverPayload struct {
	WinnerID string `json:"winnerId,omitempty"`
	LoserID  string `json:"loserId,omitempty"`
	IsDraw   bool   `json:"isDraw"`
}

type PlayerView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	HandCount int    `json:"handCount"`
	IsActive  bool   `json:"isActive"`
}

type GameStatePayload struct {
	Phase       string          `json:"phase"`
	TrumpCard   game.Card       `json:"trumpCard"`
	TrumpSuit   game.Suit       `json:"trumpSuit"`
	CardsLeft   int             `json:"cardsLeft"`
	Table       []game.CardPair `json:"table"`
	DiscardSize int             `json:"discardSize"`
	AttackerID  string          `json:"attackerId"`
	DefenderID  string          `json:"defenderId"`
	MyHand      []game.Card     `json:"myHand"`
	Players     []PlayerView    `json:"players"`
}

func NewMessage(msgType string, payload any) ([]byte, error) {
	return json.Marshal(Message[any]{
		Type:    msgType,
		Payload: payload,
	})
}
