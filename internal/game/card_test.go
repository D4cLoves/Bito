package game

import "testing"

func TestSuit_String(t *testing.T) {
	tests := []struct {
		suit     Suit
		expected string
	}{
		{Spades, "♠"},
		{Clubs, "♣"},
		{Hearts, "♥"},
		{Diamonds, "♦"},
		{Suit(99), "?"},
	}

	for _, tt := range tests {
		if got := tt.suit.String(); got != tt.expected {
			t.Errorf("Suit(%d).String() = %v, want %v", tt.suit, got, tt.expected)
		}
	}
}

func TestRank_String(t *testing.T) {
	tests := []struct {
		rank     Rank
		expected string
	}{
		{Rank2, "2"},
		{Rank5, "5"},
		{Rank10, "10"},
		{RankJack, "J"},
		{RankQueen, "Q"},
		{RankKing, "K"},
		{RankAce, "A"},
		{RankInvalid, "?"},
		{Rank(99), "?"},
	}

	for _, tt := range tests {
		if got := tt.rank.String(); got != tt.expected {
			t.Errorf("Rank(%d).String() = %v, want %v", tt.rank, got, tt.expected)
		}
	}
}

func TestCard_String(t *testing.T) {
	c := Card{Suit: Spades, Rank: RankAce}
	if got := c.String(); got != "A♠" {
		t.Errorf("Card.String() = %v, want A♠", got)
	}

	c10 := Card{Suit: Hearts, Rank: Rank10}
	if got := c10.String(); got != "10♥" {
		t.Errorf("Card.String() = %v, want 10♥", got)
	}
}
