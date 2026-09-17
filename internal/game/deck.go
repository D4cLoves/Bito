package game

import (
	"crypto/rand"
	"errors"
	"math/big"
)

var (
	ErrEmptyDeck       = errors.New("deck is empty")
	ErrInvalidDeckType = errors.New("invalid deck type")
	ErrInvalidCount    = errors.New("invalid count")
)

type DeckType uint8

const (
	Deck24 DeckType = 24
	Deck36 DeckType = 36
	Deck52 DeckType = 52
)

type Deck struct {
	cards    []Card
	deckType DeckType
	trump    Card
}

func NewDeck(deckType DeckType) (*Deck, error) {
	var startRank Rank

	switch deckType {
	case Deck24:
		startRank = Rank9
	case Deck36:
		startRank = Rank6
	case Deck52:
		startRank = Rank2
	default:
		return nil, ErrInvalidDeckType
	}

	cards := make([]Card, 0, deckType)

	for s := Spades; s <= Diamonds; s++ {
		for r := startRank; r <= RankAce; r++ {
			cards = append(cards, Card{Suit: s, Rank: r})
		}
	}

	deck := &Deck{
		cards:    cards,
		deckType: deckType,
	}

	if err := deck.Shuffle(); err != nil {
		return nil, err
	}

	deck.trump = deck.cards[len(deck.cards)-1]

	return deck, nil
}

func (d *Deck) Cards() []Card {
	cardsCopy := make([]Card, len(d.cards))
	copy(cardsCopy, d.cards)
	return cardsCopy
}

func (d *Deck) Type() DeckType {
	return d.deckType
}

func (d *Deck) Trump() Card {
	return d.trump
}

func (d *Deck) CardsLeft() int {
	return len(d.cards)
}

func (d *Deck) Shuffle() error {
	if len(d.cards) == 0 {
		return ErrEmptyDeck
	}

	for i := len(d.cards) - 1; i > 0; i-- {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}

		j := int(nBig.Int64())
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	}

	return nil
}

func (d *Deck) Draw() (Card, error) {
	if len(d.cards) == 0 {
		return Card{}, ErrEmptyDeck
	}

	card := d.cards[0]
	d.cards = d.cards[1:]
	return card, nil
}

func (d *Deck) DrawMany(count int) ([]Card, error) {
	if count <= 0 {
		return nil, ErrInvalidCount
	}
	if len(d.cards) == 0 {
		return nil, ErrEmptyDeck
	}
	if count > len(d.cards) {
		count = len(d.cards)
	}

	cards := make([]Card, count)
	copy(cards, d.cards[:count])
	d.cards = d.cards[count:]

	return cards, nil
}
