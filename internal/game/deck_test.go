package game

import (
	"errors"
	"testing"
)

func TestNewDeck_SizesAndUniqueness(t *testing.T) {
	deckTypes := []DeckType{Deck24, Deck36, Deck52}

	for _, dt := range deckTypes {
		deck, err := NewDeck(dt)
		if err != nil {
			t.Fatalf("NewDeck(%d) unexpected error: %v", dt, err)
		}

		if deck.Type() != dt {
			t.Errorf("deck.Type() = %d, want %d", deck.Type(), dt)
		}

		if deck.CardsLeft() != int(dt) {
			t.Errorf("deck.CardsLeft() = %d, want %d", deck.CardsLeft(), dt)
		}

		cards := deck.Cards()
		if len(cards) != int(dt) {
			t.Errorf("len(deck.Cards()) = %d, want %d", len(cards), dt)
		}

		if deck.Trump() != cards[len(cards)-1] {
			t.Errorf("deck.Trump() = %v, want last card %v", deck.Trump(), cards[len(cards)-1])
		}

		seen := make(map[Card]bool)
		for _, card := range cards {
			if seen[card] {
				t.Errorf("duplicate card found in deck %d: %v", dt, card)
			}
			seen[card] = true
		}

		if len(seen) != int(dt) {
			t.Errorf("unique cards count = %d, want %d", len(seen), dt)
		}
	}
}

func TestNewDeck_InvalidType(t *testing.T) {
	_, err := NewDeck(DeckType(99))
	if !errors.Is(err, ErrInvalidDeckType) {
		t.Errorf("NewDeck(99) err = %v, want %v", err, ErrInvalidDeckType)
	}
}

func TestDeck_Draw(t *testing.T) {
	deck, err := NewDeck(Deck36)
	if err != nil {
		t.Fatalf("NewDeck(Deck36) error: %v", err)
	}

	total := deck.CardsLeft()
	for i := 0; i < total; i++ {
		card, err := deck.Draw()
		if err != nil {
			t.Fatalf("unexpected error on draw %d: %v", i+1, err)
		}
		if card.Rank == RankInvalid {
			t.Errorf("drawn invalid card: %v", card)
		}
		if deck.CardsLeft() != total-i-1 {
			t.Errorf("cards left = %d, want %d", deck.CardsLeft(), total-i-1)
		}
	}

	card, err := deck.Draw()
	if !errors.Is(err, ErrEmptyDeck) {
		t.Errorf("Draw() on empty deck err = %v, want %v", err, ErrEmptyDeck)
	}
	if card != (Card{}) {
		t.Errorf("Draw() on empty deck returned non-empty card: %v", card)
	}
}

func TestDeck_DrawMany(t *testing.T) {
	t.Run("normal draw", func(t *testing.T) {
		deck, err := NewDeck(Deck36)
		if err != nil {
			t.Fatalf("NewDeck error: %v", err)
		}

		cards, err := deck.DrawMany(6)
		if err != nil {
			t.Fatalf("DrawMany(6) error: %v", err)
		}
		if len(cards) != 6 {
			t.Errorf("len(cards) = %d, want 6", len(cards))
		}
		if deck.CardsLeft() != 30 {
			t.Errorf("deck.CardsLeft() = %d, want 30", deck.CardsLeft())
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		deck, _ := NewDeck(Deck36)

		cards, err := deck.DrawMany(0)
		if !errors.Is(err, ErrInvalidCount) || cards != nil {
			t.Errorf("DrawMany(0) = (%v, %v), want (nil, %v)", cards, err, ErrInvalidCount)
		}

		cards, err = deck.DrawMany(-5)
		if !errors.Is(err, ErrInvalidCount) || cards != nil {
			t.Errorf("DrawMany(-5) = (%v, %v), want (nil, %v)", cards, err, ErrInvalidCount)
		}
	})

	t.Run("draw more than remaining", func(t *testing.T) {
		deck, _ := NewDeck(Deck24)

		_, err := deck.DrawMany(20)
		if err != nil {
			t.Fatalf("DrawMany(20) error: %v", err)
		}

		cards, err := deck.DrawMany(10)
		if err != nil {
			t.Fatalf("DrawMany(10) error: %v", err)
		}
		if len(cards) != 4 {
			t.Errorf("len(cards) = %d, want 4", len(cards))
		}
		if deck.CardsLeft() != 0 {
			t.Errorf("deck.CardsLeft() = %d, want 0", deck.CardsLeft())
		}

		emptyCards, err := deck.DrawMany(1)
		if !errors.Is(err, ErrEmptyDeck) || emptyCards != nil {
			t.Errorf("DrawMany(1) on empty deck = (%v, %v), want (nil, %v)", emptyCards, err, ErrEmptyDeck)
		}
	})
}

func TestDeck_Shuffle_EmptyDeck(t *testing.T) {
	deck := &Deck{}
	if err := deck.Shuffle(); !errors.Is(err, ErrEmptyDeck) {
		t.Errorf("Shuffle() on empty deck err = %v, want %v", err, ErrEmptyDeck)
	}
}
