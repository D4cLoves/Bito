package game

import (
	"errors"
	"sort"
)

var ErrCardNotInHand = errors.New("card not in hand")

type PlayerID string
type PlayerState uint8

const (
	PlayerActive PlayerState = iota
	PlayerFinished
	PlayerSurrendered
)

func (s PlayerState) String() string {
	switch s {
	case PlayerActive:
		return "active"
	case PlayerFinished:
		return "finished"
	case PlayerSurrendered:
		return "surrendered"
	default:
		return "unknown"
	}
}

type Player struct {
	id    PlayerID
	name  string
	hand  []Card
	state PlayerState
}

func NewPlayer(id, name string) *Player {
	return &Player{
		id:    PlayerID(id),
		name:  name,
		hand:  make([]Card, 0),
		state: PlayerActive,
	}
}

func (p *Player) ID() PlayerID {
	return p.id
}

func (p *Player) Name() string {
	return p.name
}

func (p *Player) State() PlayerState {
	return p.state
}

func (p *Player) SetState(state PlayerState) {
	p.state = state
}

func (p *Player) Hand() []Card {
	handCopy := make([]Card, len(p.hand))
	copy(handCopy, p.hand)
	return handCopy
}

func (p *Player) HandCount() int {
	return len(p.hand)
}

func (p *Player) HasCard(c Card) bool {
	for _, card := range p.hand {
		if card == c {
			return true
		}
	}
	return false
}

func (p *Player) HasRank(rank Rank) bool {
	for _, card := range p.hand {
		if card.Rank == rank {
			return true
		}
	}
	return false
}

func (p *Player) LowestTrump(trump Suit) (Card, bool) {
	return FindLowestTrump(p.hand, trump)
}

func (p *Player) AddCard(c Card) {
	p.hand = append(p.hand, c)
}

func (p *Player) AddCards(cards []Card) {
	p.hand = append(p.hand, cards...)
}

func (p *Player) RemoveCard(c Card) error {
	for i, card := range p.hand {
		if card == c {
			p.hand = append(p.hand[:i], p.hand[i+1:]...)
			return nil
		}
	}
	return ErrCardNotInHand
}

func (p *Player) ClearHand() {
	p.hand = p.hand[:0]
}

func (p *Player) IsActive() bool {
	return p.state == PlayerActive
}

func (p *Player) SortHand(trump Suit) {
	sort.Slice(p.hand, func(i, j int) bool {
		c1, c2 := p.hand[i], p.hand[j]
		t1 := c1.Suit == trump
		t2 := c2.Suit == trump

		if t1 != t2 {
			return !t1
		}

		if c1.Rank != c2.Rank {
			return c1.Rank < c2.Rank
		}

		return c1.Suit < c2.Suit
	})
}
