package game

import (
	"errors"
	"testing"
)

func TestNewTable(t *testing.T) {
	table := NewTable()
	if table == nil {
		t.Fatal("NewTable() returned nil")
	}
	if table.PairsCount() != 0 {
		t.Errorf("PairsCount() = %d, want 0", table.PairsCount())
	}
	if table.UnbeatenCount() != 0 {
		t.Errorf("UnbeatenCount() = %d, want 0", table.UnbeatenCount())
	}
	if _, ok := table.FirstAttack(); ok {
		t.Error("FirstAttack() on empty table want false")
	}
	if table.HasUnbeatenAttack(Card{Suit: Spades, Rank: Rank6}) {
		t.Error("HasUnbeatenAttack() on empty table want false")
	}
	if table.IsAllBeaten() {
		t.Error("IsAllBeaten() on empty table want false")
	}
	if table.HasUnbeaten() {
		t.Error("HasUnbeaten() on empty table want false")
	}
	if len(table.AllCards()) != 0 {
		t.Errorf("len(AllCards()) = %d, want 0", len(table.AllCards()))
	}
	if len(table.RanksOnTable()) != 0 {
		t.Errorf("len(RanksOnTable()) = %d, want 0", len(table.RanksOnTable()))
	}
}

func TestTable_AddAttack(t *testing.T) {
	table := NewTable()
	c1 := Card{Suit: Spades, Rank: Rank6}

	table.AddAttack(c1)

	if table.PairsCount() != 1 {
		t.Fatalf("PairsCount() = %d, want 1", table.PairsCount())
	}
	if table.UnbeatenCount() != 1 {
		t.Fatalf("UnbeatenCount() = %d, want 1", table.UnbeatenCount())
	}

	first, ok := table.FirstAttack()
	if !ok || first != c1 {
		t.Errorf("FirstAttack() = (%v, %v), want (%v, true)", first, ok, c1)
	}

	if !table.HasUnbeatenAttack(c1) {
		t.Error("HasUnbeatenAttack() want true")
	}

	pairs := table.Pairs()
	if pairs[0].Attack != c1 {
		t.Errorf("pairs[0].Attack = %v, want %v", pairs[0].Attack, c1)
	}
	if pairs[0].Defend != nil {
		t.Errorf("pairs[0].Defend = %v, want nil", pairs[0].Defend)
	}
	if !table.HasUnbeaten() {
		t.Error("HasUnbeaten() want true")
	}
	if table.IsAllBeaten() {
		t.Error("IsAllBeaten() want false")
	}
}

func TestTable_AddDefense(t *testing.T) {
	table := NewTable()
	attack := Card{Suit: Spades, Rank: Rank6}
	defend := Card{Suit: Spades, Rank: Rank7}
	otherCard := Card{Suit: Hearts, Rank: RankKing}

	table.AddAttack(attack)

	err := table.AddDefense(otherCard, defend)
	if !errors.Is(err, ErrAttackNotFound) {
		t.Errorf("AddDefense with unknown attack err = %v, want %v", err, ErrAttackNotFound)
	}

	err = table.AddDefense(attack, defend)
	if err != nil {
		t.Fatalf("AddDefense() unexpected error: %v", err)
	}

	if table.HasUnbeatenAttack(attack) {
		t.Error("HasUnbeatenAttack() after beaten want false")
	}

	pairs := table.Pairs()
	if pairs[0].Defend == nil || *pairs[0].Defend != defend {
		t.Errorf("pairs[0].Defend = %v, want %v", pairs[0].Defend, defend)
	}

	err = table.AddDefense(attack, otherCard)
	if !errors.Is(err, ErrCardAlreadyBeaten) {
		t.Errorf("AddDefense on beaten card err = %v, want %v", err, ErrCardAlreadyBeaten)
	}
}

func TestTable_IsAllBeaten_HasUnbeaten(t *testing.T) {
	table := NewTable()
	c1 := Card{Suit: Spades, Rank: Rank6}
	d1 := Card{Suit: Spades, Rank: Rank7}
	c2 := Card{Suit: Hearts, Rank: Rank8}
	d2 := Card{Suit: Hearts, Rank: Rank9}

	if table.IsAllBeaten() || table.HasUnbeaten() || table.UnbeatenCount() != 0 {
		t.Error("empty table: IsAllBeaten and HasUnbeaten should be false, UnbeatenCount 0")
	}

	table.AddAttack(c1)
	if table.IsAllBeaten() || !table.HasUnbeaten() || table.UnbeatenCount() != 1 {
		t.Error("1 attack: IsAllBeaten false, HasUnbeaten true, UnbeatenCount 1")
	}

	_ = table.AddDefense(c1, d1)
	if !table.IsAllBeaten() || table.HasUnbeaten() || table.UnbeatenCount() != 0 {
		t.Error("1 attack beaten: IsAllBeaten true, HasUnbeaten false, UnbeatenCount 0")
	}

	table.AddAttack(c2)
	if table.IsAllBeaten() || !table.HasUnbeaten() || table.UnbeatenCount() != 1 {
		t.Error("2 attacks (1 beaten): IsAllBeaten false, HasUnbeaten true, UnbeatenCount 1")
	}

	_ = table.AddDefense(c2, d2)
	if !table.IsAllBeaten() || table.HasUnbeaten() || table.UnbeatenCount() != 0 {
		t.Error("2 attacks (all beaten): IsAllBeaten true, HasUnbeaten false, UnbeatenCount 0")
	}
}

func TestTable_AllCards(t *testing.T) {
	table := NewTable()
	c1 := Card{Suit: Spades, Rank: Rank6}
	d1 := Card{Suit: Spades, Rank: Rank7}
	c2 := Card{Suit: Hearts, Rank: Rank8}

	table.AddAttack(c1)
	_ = table.AddDefense(c1, d1)
	table.AddAttack(c2)

	all := table.AllCards()
	if len(all) != 3 {
		t.Fatalf("len(AllCards()) = %d, want 3", len(all))
	}
	if all[0] != c1 || all[1] != d1 || all[2] != c2 {
		t.Errorf("AllCards() order mismatch: got %v", all)
	}
}

func TestTable_RanksOnTable(t *testing.T) {
	table := NewTable()
	c1 := Card{Suit: Spades, Rank: Rank6}
	d1 := Card{Suit: Spades, Rank: Rank7}
	c2 := Card{Suit: Hearts, Rank: Rank8}

	table.AddAttack(c1)
	_ = table.AddDefense(c1, d1)
	table.AddAttack(c2)

	ranks := table.RanksOnTable()
	if len(ranks) != 3 {
		t.Errorf("len(ranks) = %d, want 3", len(ranks))
	}
	if !ranks[Rank6] || !ranks[Rank7] || !ranks[Rank8] {
		t.Errorf("expected ranks 6, 7, 8 in %v", ranks)
	}
	if ranks[Rank9] {
		t.Errorf("rank 9 should not be present in %v", ranks)
	}
}

func TestTable_Clear(t *testing.T) {
	table := NewTable()
	table.AddAttack(Card{Suit: Spades, Rank: Rank6})
	table.AddAttack(Card{Suit: Hearts, Rank: Rank7})

	table.Clear()

	if table.PairsCount() != 0 {
		t.Errorf("PairsCount() after Clear = %d, want 0", table.PairsCount())
	}
	if table.UnbeatenCount() != 0 {
		t.Errorf("UnbeatenCount() after Clear = %d, want 0", table.UnbeatenCount())
	}
	if len(table.AllCards()) != 0 {
		t.Errorf("len(AllCards()) after Clear = %d, want 0", len(table.AllCards()))
	}
}
