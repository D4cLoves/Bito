package game

import (
	"errors"
	"testing"
)

func TestCanBeat(t *testing.T) {
	trump := Hearts

	tests := []struct {
		name   string
		attack Card
		defend Card
		want   bool
	}{
		{
			name:   "invalid attack rank",
			attack: Card{Suit: Spades, Rank: RankInvalid},
			defend: Card{Suit: Spades, Rank: Rank10},
			want:   false,
		},
		{
			name:   "invalid defend rank",
			attack: Card{Suit: Spades, Rank: Rank9},
			defend: Card{Suit: Spades, Rank: RankInvalid},
			want:   false,
		},
		{
			name:   "invalid suit",
			attack: Card{Suit: 99, Rank: Rank9},
			defend: Card{Suit: Spades, Rank: Rank10},
			want:   false,
		},
		{
			name:   "same suit higher rank",
			attack: Card{Suit: Spades, Rank: Rank9},
			defend: Card{Suit: Spades, Rank: Rank10},
			want:   true,
		},
		{
			name:   "same suit lower rank",
			attack: Card{Suit: Spades, Rank: Rank10},
			defend: Card{Suit: Spades, Rank: Rank9},
			want:   false,
		},
		{
			name:   "same suit equal rank",
			attack: Card{Suit: Spades, Rank: Rank10},
			defend: Card{Suit: Spades, Rank: Rank10},
			want:   false,
		},
		{
			name:   "different non-trump suits",
			attack: Card{Suit: Spades, Rank: Rank9},
			defend: Card{Suit: Clubs, Rank: RankAce},
			want:   false,
		},
		{
			name:   "trump beats non-trump",
			attack: Card{Suit: Spades, Rank: RankAce},
			defend: Card{Suit: Hearts, Rank: Rank6},
			want:   true,
		},
		{
			name:   "non-trump cannot beat trump",
			attack: Card{Suit: Hearts, Rank: Rank6},
			defend: Card{Suit: Spades, Rank: RankAce},
			want:   false,
		},
		{
			name:   "higher trump beats lower trump",
			attack: Card{Suit: Hearts, Rank: Rank6},
			defend: Card{Suit: Hearts, Rank: Rank7},
			want:   true,
		},
		{
			name:   "lower trump cannot beat higher trump",
			attack: Card{Suit: Hearts, Rank: RankKing},
			defend: Card{Suit: Hearts, Rank: RankQueen},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanBeat(tt.attack, tt.defend, trump)
			if got != tt.want {
				t.Errorf("CanBeat(%v, %v) = %v, want %v", tt.attack, tt.defend, got, tt.want)
			}
		})
	}
}

func TestCanInitialAttack(t *testing.T) {
	table := NewTable()
	validCard := Card{Suit: Spades, Rank: Rank6}
	invalidRank := Card{Suit: Spades, Rank: RankInvalid}
	invalidSuit := Card{Suit: 99, Rank: Rank6}

	if !CanInitialAttack(table, validCard) {
		t.Error("CanInitialAttack() want true on empty table with valid card")
	}

	if CanInitialAttack(table, invalidRank) {
		t.Error("CanInitialAttack() want false with invalid rank")
	}
	if CanInitialAttack(table, invalidSuit) {
		t.Error("CanInitialAttack() want false with invalid suit")
	}

	table.AddAttack(validCard)
	if CanInitialAttack(table, validCard) {
		t.Error("CanInitialAttack() want false when table not empty")
	}
}

func TestCanToss(t *testing.T) {
	table := NewTable()
	c6 := Card{Suit: Spades, Rank: Rank6}
	d7 := Card{Suit: Hearts, Rank: Rank7}
	toss6 := Card{Suit: Clubs, Rank: Rank6}
	toss7 := Card{Suit: Diamonds, Rank: Rank7}
	toss8 := Card{Suit: Spades, Rank: Rank8}
	invalidCard := Card{Suit: Spades, Rank: RankInvalid}
	invalidSuit := Card{Suit: 99, Rank: Rank6}

	if CanToss(table, toss6) {
		t.Error("CanToss() on empty table want false")
	}

	table.AddAttack(c6)
	_ = table.AddDefense(c6, d7)

	if CanToss(table, invalidCard) {
		t.Error("CanToss() with invalid card want false")
	}
	if CanToss(table, invalidSuit) {
		t.Error("CanToss() with invalid suit want false")
	}
	if !CanToss(table, toss6) {
		t.Error("CanToss() rank 6 want true")
	}
	if !CanToss(table, toss7) {
		t.Error("CanToss() rank 7 want true")
	}
	if CanToss(table, toss8) {
		t.Error("CanToss() rank 8 want false")
	}
}

func TestCanDefend(t *testing.T) {
	table := NewTable()
	trump := Hearts

	attackCard := Card{Suit: Spades, Rank: Rank6}
	higherSameSuit := Card{Suit: Spades, Rank: Rank10}
	lowerSameSuit := Card{Suit: Spades, Rank: Rank5}
	notOnTable := Card{Suit: Diamonds, Rank: RankAce}

	if CanDefend(table, attackCard, lowerSameSuit, trump) {
		t.Error("CanDefend() on card that cannot beat want false")
	}

	if CanDefend(table, notOnTable, higherSameSuit, trump) {
		t.Error("CanDefend() when attack card not on table want false")
	}

	table.AddAttack(attackCard)
	if !CanDefend(table, attackCard, higherSameSuit, trump) {
		t.Error("CanDefend() want true")
	}

	_ = table.AddDefense(attackCard, higherSameSuit)
	anotherDefend := Card{Suit: Hearts, Rank: RankAce}
	if CanDefend(table, attackCard, anotherDefend, trump) {
		t.Error("CanDefend() on already beaten card want false")
	}
}

func TestCanTransfer(t *testing.T) {
	table := NewTable()
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c6Clubs := Card{Suit: Clubs, Rank: Rank6}
	c7Hearts := Card{Suit: Hearts, Rank: Rank7}
	invalidCard := Card{Suit: Spades, Rank: RankInvalid}

	if CanTransfer(table, c6Clubs, 5) {
		t.Error("CanTransfer() on empty table want false")
	}

	table.AddAttack(c6Spades)

	if CanTransfer(table, invalidCard, 5) {
		t.Error("CanTransfer() with invalid card want false")
	}

	if CanTransfer(table, c6Clubs, 0) {
		t.Error("CanTransfer() with 0 defender cards want false")
	}

	if CanTransfer(table, c6Clubs, 1) {
		t.Error("CanTransfer() when pairs+1 > defenderCount want false")
	}

	if CanTransfer(table, c7Hearts, 2) {
		t.Error("CanTransfer() with wrong rank want false")
	}

	if !CanTransfer(table, c6Clubs, 2) {
		t.Error("CanTransfer() want true")
	}

	tMultiRank := NewTable()
	tMultiRank.AddAttack(c6Spades)
	tMultiRank.AddAttack(c7Hearts)
	if CanTransfer(tMultiRank, c6Clubs, 5) {
		t.Error("CanTransfer() when multiple different ranks on table want false")
	}

	_ = table.AddDefense(c6Spades, c7Hearts)
	if CanTransfer(table, c6Clubs, 5) {
		t.Error("CanTransfer() when card already beaten want false")
	}
}

func TestMaxRoundLimit(t *testing.T) {
	if got := MaxRoundLimit(true); got != 5 {
		t.Errorf("MaxRoundLimit(true) = %d, want 5", got)
	}
	if got := MaxRoundLimit(false); got != 6 {
		t.Errorf("MaxRoundLimit(false) = %d, want 6", got)
	}
}

func TestCanAddAttack(t *testing.T) {
	table := NewTable()

	if !CanAddAttack(table, 6, false) {
		t.Error("CanAddAttack() empty table want true")
	}

	if CanAddAttack(table, 0, false) {
		t.Error("CanAddAttack() 0 defender cards want false")
	}

	table.AddAttack(Card{Suit: Spades, Rank: Rank6})
	if CanAddAttack(table, 1, false) {
		t.Error("CanAddAttack() unbeaten == defenderCount want false")
	}
	if !CanAddAttack(table, 2, false) {
		t.Error("CanAddAttack() unbeaten < defenderCount want true")
	}

	tFirst := NewTable()
	for i := 0; i < 5; i++ {
		tFirst.AddAttack(Card{Suit: Spades, Rank: Rank6 + Rank(i)})
	}
	if CanAddAttack(tFirst, 6, true) {
		t.Error("CanAddAttack() first round with 5 pairs want false")
	}

	tNormal := NewTable()
	for i := 0; i < 6; i++ {
		tNormal.AddAttack(Card{Suit: Spades, Rank: Rank6 + Rank(i)})
	}
	if CanAddAttack(tNormal, 6, false) {
		t.Error("CanAddAttack() normal round with 6 pairs want false")
	}
}

func TestValidateAttack(t *testing.T) {
	table := NewTable()
	c6 := Card{Suit: Spades, Rank: Rank6}
	c7 := Card{Suit: Hearts, Rank: Rank7}
	invalidCard := Card{Suit: Spades, Rank: RankInvalid}

	if err := ValidateAttack(table, invalidCard, 6, false); !errors.Is(err, ErrInvalidCard) {
		t.Errorf("ValidateAttack invalid card err = %v, want %v", err, ErrInvalidCard)
	}

	if err := ValidateAttack(table, c6, 0, false); !errors.Is(err, ErrAttackLimitExceeded) {
		t.Errorf("ValidateAttack 0 defender err = %v, want %v", err, ErrAttackLimitExceeded)
	}

	if err := ValidateAttack(table, c6, 6, false); err != nil {
		t.Errorf("ValidateAttack initial want nil, got %v", err)
	}

	table.AddAttack(c6)

	if err := ValidateAttack(table, c7, 6, false); !errors.Is(err, ErrCannotToss) {
		t.Errorf("ValidateAttack toss wrong rank err = %v, want %v", err, ErrCannotToss)
	}

	c6Clubs := Card{Suit: Clubs, Rank: Rank6}
	if err := ValidateAttack(table, c6Clubs, 6, false); err != nil {
		t.Errorf("ValidateAttack valid toss want nil, got %v", err)
	}
}

func TestValidateDefense(t *testing.T) {
	table := NewTable()
	trump := Hearts

	attack := Card{Suit: Spades, Rank: Rank6}
	defendWin := Card{Suit: Spades, Rank: Rank10}
	defendLose := Card{Suit: Spades, Rank: Rank5}
	invalidCard := Card{Suit: Spades, Rank: RankInvalid}

	if err := ValidateDefense(table, invalidCard, defendWin, trump); !errors.Is(err, ErrInvalidCard) {
		t.Errorf("ValidateDefense invalid card err = %v, want %v", err, ErrInvalidCard)
	}

	if err := ValidateDefense(table, attack, defendWin, trump); !errors.Is(err, ErrAttackNotFound) {
		t.Errorf("ValidateDefense attack not found err = %v, want %v", err, ErrAttackNotFound)
	}

	table.AddAttack(attack)

	if err := ValidateDefense(table, attack, defendLose, trump); !errors.Is(err, ErrCannotBeat) {
		t.Errorf("ValidateDefense cannot beat err = %v, want %v", err, ErrCannotBeat)
	}

	if err := ValidateDefense(table, attack, defendWin, trump); err != nil {
		t.Errorf("ValidateDefense valid want nil, got %v", err)
	}

	_ = table.AddDefense(attack, defendWin)

	if err := ValidateDefense(table, attack, defendWin, trump); !errors.Is(err, ErrCardAlreadyBeaten) {
		t.Errorf("ValidateDefense already beaten err = %v, want %v", err, ErrCardAlreadyBeaten)
	}
}

func TestValidateTransfer(t *testing.T) {
	table := NewTable()
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c6Clubs := Card{Suit: Clubs, Rank: Rank6}
	c7Hearts := Card{Suit: Hearts, Rank: Rank7}
	invalidCard := Card{Suit: Spades, Rank: RankInvalid}

	if err := ValidateTransfer(table, invalidCard, 5, false); !errors.Is(err, ErrInvalidCard) {
		t.Errorf("ValidateTransfer invalid card err = %v, want %v", err, ErrInvalidCard)
	}

	if err := ValidateTransfer(table, c6Clubs, 5, false); !errors.Is(err, ErrTransferNotAllowed) {
		t.Errorf("ValidateTransfer empty table err = %v, want %v", err, ErrTransferNotAllowed)
	}

	table.AddAttack(c6Spades)

	if err := ValidateTransfer(table, c7Hearts, 5, false); !errors.Is(err, ErrTransferNotAllowed) {
		t.Errorf("ValidateTransfer rank mismatch err = %v, want %v", err, ErrTransferNotAllowed)
	}

	if err := ValidateTransfer(table, c6Clubs, 1, false); !errors.Is(err, ErrTransferNotAllowed) {
		t.Errorf("ValidateTransfer hand too small err = %v, want %v", err, ErrTransferNotAllowed)
	}

	if err := ValidateTransfer(table, c6Clubs, 5, false); err != nil {
		t.Errorf("ValidateTransfer valid want nil, got %v", err)
	}

	tMax := NewTable()
	for i := 0; i < 5; i++ {
		tMax.AddAttack(c6Spades)
	}
	if err := ValidateTransfer(tMax, c6Clubs, 6, true); !errors.Is(err, ErrAttackLimitExceeded) {
		t.Errorf("ValidateTransfer max round err = %v, want %v", err, ErrAttackLimitExceeded)
	}

	_ = table.AddDefense(c6Spades, c7Hearts)
	if err := ValidateTransfer(table, c6Clubs, 5, false); !errors.Is(err, ErrCardAlreadyBeaten) {
		t.Errorf("ValidateTransfer already beaten err = %v, want %v", err, ErrCardAlreadyBeaten)
	}
}

func TestFindLowestTrump(t *testing.T) {
	trump := Hearts

	if _, found := FindLowestTrump(nil, 99); found {
		t.Error("FindLowestTrump() invalid trump want found false")
	}

	noTrumps := []Card{
		{Suit: Spades, Rank: RankAce},
		{Suit: Clubs, Rank: RankKing},
		{Suit: Hearts, Rank: RankInvalid},
	}
	if _, found := FindLowestTrump(noTrumps, trump); found {
		t.Error("FindLowestTrump() with no trumps want found false")
	}

	withTrumps := []Card{
		{Suit: Spades, Rank: RankAce},
		{Suit: Hearts, Rank: Rank10},
		{Suit: Hearts, Rank: Rank7},
		{Suit: Hearts, Rank: RankAce},
	}
	lowest, found := FindLowestTrump(withTrumps, trump)
	if !found {
		t.Fatal("FindLowestTrump() want found true")
	}
	if lowest != (Card{Suit: Hearts, Rank: Rank7}) {
		t.Errorf("FindLowestTrump() = %v, want 7♥", lowest)
	}
}

func TestDetermineFirstAttacker(t *testing.T) {
	trump := Hearts

	if player, _, found := DetermineFirstAttacker(nil, 99); found || player != "" {
		t.Error("DetermineFirstAttacker(invalid trump) want found false")
	}

	if player, _, found := DetermineFirstAttacker(nil, trump); found || player != "" {
		t.Error("DetermineFirstAttacker(nil) want found false")
	}

	handsNoTrumps := map[string][]Card{
		"p1": {{Suit: Spades, Rank: RankAce}},
		"p2": {{Suit: Clubs, Rank: RankKing}},
	}
	if player, _, found := DetermineFirstAttacker(handsNoTrumps, trump); found || player != "" {
		t.Error("DetermineFirstAttacker with no trumps want found false")
	}

	handsWithTrumps := map[string][]Card{
		"p1": {
			{Suit: Spades, Rank: RankAce},
			{Suit: Hearts, Rank: Rank10},
		},
		"p2": {
			{Suit: Clubs, Rank: RankKing},
			{Suit: Hearts, Rank: Rank6},
			{Suit: Hearts, Rank: RankAce},
		},
		"p3": {
			{Suit: Diamonds, Rank: Rank7},
		},
	}

	player, card, found := DetermineFirstAttacker(handsWithTrumps, trump)
	if !found {
		t.Fatal("DetermineFirstAttacker want found true")
	}
	if player != "p2" {
		t.Errorf("DetermineFirstAttacker player = %s, want p2", player)
	}
	if card != (Card{Suit: Hearts, Rank: Rank6}) {
		t.Errorf("DetermineFirstAttacker card = %v, want 6♥", card)
	}
}
