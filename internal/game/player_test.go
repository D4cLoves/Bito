package game

import (
	"errors"
	"testing"
)

func TestPlayerState_String(t *testing.T) {
	tests := []struct {
		state PlayerState
		want  string
	}{
		{PlayerActive, "active"},
		{PlayerFinished, "finished"},
		{PlayerSurrendered, "surrendered"},
		{PlayerState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("PlayerState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	if p == nil {
		t.Fatal("NewPlayer returned nil")
	}

	if p.ID() != "p1" {
		t.Errorf("ID() = %v, want p1", p.ID())
	}
	if p.Name() != "Alice" {
		t.Errorf("Name() = %v, want Alice", p.Name())
	}
	if p.State() != PlayerActive {
		t.Errorf("State() = %v, want PlayerActive", p.State())
	}
	if !p.IsActive() {
		t.Error("IsActive() want true")
	}
	if p.HandCount() != 0 {
		t.Errorf("HandCount() = %d, want 0", p.HandCount())
	}
	if len(p.Hand()) != 0 {
		t.Errorf("len(Hand()) = %d, want 0", len(p.Hand()))
	}
}

func TestPlayer_SetState(t *testing.T) {
	p := NewPlayer("p1", "Alice")

	p.SetState(PlayerFinished)
	if p.State() != PlayerFinished {
		t.Errorf("State() = %v, want PlayerFinished", p.State())
	}
	if p.IsActive() {
		t.Error("IsActive() on finished player want false")
	}

	p.SetState(PlayerSurrendered)
	if p.State() != PlayerSurrendered {
		t.Errorf("State() = %v, want PlayerSurrendered", p.State())
	}
	if p.IsActive() {
		t.Error("IsActive() on surrendered player want false")
	}
}

func TestPlayer_AddCard_AddCards(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	c1 := Card{Suit: Spades, Rank: Rank6}
	c2 := Card{Suit: Hearts, Rank: Rank7}
	c3 := Card{Suit: Diamonds, Rank: Rank8}

	p.AddCard(c1)
	if p.HandCount() != 1 {
		t.Fatalf("HandCount() = %d, want 1", p.HandCount())
	}
	if !p.HasCard(c1) {
		t.Error("HasCard(c1) want true")
	}
	if p.HasCard(c2) {
		t.Error("HasCard(c2) want false")
	}

	p.AddCards([]Card{c2, c3})
	if p.HandCount() != 3 {
		t.Fatalf("HandCount() = %d, want 3", p.HandCount())
	}
	if !p.HasCard(c2) || !p.HasCard(c3) {
		t.Error("HasCard() for added cards want true")
	}
}

func TestPlayer_HasRank(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	p.AddCard(Card{Suit: Spades, Rank: Rank6})
	p.AddCard(Card{Suit: Hearts, Rank: RankAce})

	if !p.HasRank(Rank6) {
		t.Error("HasRank(Rank6) want true")
	}
	if !p.HasRank(RankAce) {
		t.Error("HasRank(RankAce) want true")
	}
	if p.HasRank(Rank7) {
		t.Error("HasRank(Rank7) want false")
	}
}

func TestPlayer_LowestTrump(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	trump := Diamonds

	if _, found := p.LowestTrump(trump); found {
		t.Error("LowestTrump on empty hand want false")
	}

	p.AddCard(Card{Suit: Diamonds, Rank: Rank10})
	p.AddCard(Card{Suit: Diamonds, Rank: Rank7})
	p.AddCard(Card{Suit: Spades, Rank: Rank6})

	lowest, found := p.LowestTrump(trump)
	if !found {
		t.Fatal("LowestTrump want found true")
	}
	if lowest != (Card{Suit: Diamonds, Rank: Rank7}) {
		t.Errorf("LowestTrump = %v, want 7♦", lowest)
	}
}

func TestPlayer_Hand_Copy(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	c1 := Card{Suit: Spades, Rank: Rank6}
	p.AddCard(c1)

	hand := p.Hand()
	hand[0] = Card{Suit: Hearts, Rank: RankAce}

	if p.Hand()[0] != c1 {
		t.Error("Hand() slice mutation affected internal player hand")
	}
}

func TestPlayer_RemoveCard(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	c1 := Card{Suit: Spades, Rank: Rank6}
	c2 := Card{Suit: Hearts, Rank: Rank7}
	c3 := Card{Suit: Diamonds, Rank: Rank8}

	p.AddCards([]Card{c1, c2, c3})

	err := p.RemoveCard(Card{Suit: Clubs, Rank: RankAce})
	if !errors.Is(err, ErrCardNotInHand) {
		t.Errorf("RemoveCard with missing card err = %v, want %v", err, ErrCardNotInHand)
	}

	err = p.RemoveCard(c2)
	if err != nil {
		t.Fatalf("RemoveCard unexpected error: %v", err)
	}

	if p.HandCount() != 2 {
		t.Fatalf("HandCount() after removal = %d, want 2", p.HandCount())
	}
	if p.HasCard(c2) {
		t.Error("HasCard(c2) after removal want false")
	}

	hand := p.Hand()
	if hand[0] != c1 || hand[1] != c3 {
		t.Errorf("Hand order after middle removal mismatch: got %v", hand)
	}
}

func TestPlayer_ClearHand(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	p.AddCards([]Card{
		{Suit: Spades, Rank: Rank6},
		{Suit: Hearts, Rank: Rank7},
	})

	p.ClearHand()
	if p.HandCount() != 0 {
		t.Errorf("HandCount() after ClearHand = %d, want 0", p.HandCount())
	}
	if len(p.Hand()) != 0 {
		t.Errorf("len(Hand()) after ClearHand = %d, want 0", len(p.Hand()))
	}
}

func TestPlayer_SortHand(t *testing.T) {
	p := NewPlayer("p1", "Alice")
	trump := Diamonds

	cards := []Card{
		{Suit: Spades, Rank: Rank9},
		{Suit: Diamonds, Rank: Rank6},
		{Suit: Clubs, Rank: Rank7},
		{Suit: Spades, Rank: Rank7},
		{Suit: Diamonds, Rank: RankAce},
		{Suit: Hearts, Rank: RankKing},
	}
	p.AddCards(cards)

	p.SortHand(trump)

	want := []Card{
		{Suit: Spades, Rank: Rank7},
		{Suit: Clubs, Rank: Rank7},
		{Suit: Spades, Rank: Rank9},
		{Suit: Hearts, Rank: RankKing},
		{Suit: Diamonds, Rank: Rank6},
		{Suit: Diamonds, Rank: RankAce},
	}

	got := p.Hand()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SortHand[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
