package game

import "errors"

var (
	ErrAttackNotFound    = errors.New("attack card not found on table")
	ErrCardAlreadyBeaten = errors.New("card is already beaten")
)

type CardPair struct {
	Attack Card  `json:"attack"`
	Defend *Card `json:"defend,omitempty"`
}

type Table struct {
	pairs []CardPair
}

func NewTable() *Table {
	return &Table{
		pairs: make([]CardPair, 0),
	}
}

func (t *Table) Pairs() []CardPair {
	pairsCopy := make([]CardPair, len(t.pairs))
	copy(pairsCopy, t.pairs)
	return pairsCopy
}

func (t *Table) PairsCount() int {
	return len(t.pairs)
}

func (t *Table) FirstAttack() (Card, bool) {
	if len(t.pairs) == 0 {
		return Card{}, false
	}
	return t.pairs[0].Attack, true
}

func (t *Table) HasUnbeatenAttack(attackCard Card) bool {
	for _, p := range t.pairs {
		if p.Attack == attackCard && p.Defend == nil {
			return true
		}
	}
	return false
}

func (t *Table) AddAttack(card Card) {
	t.pairs = append(t.pairs, CardPair{
		Attack: card,
		Defend: nil,
	})
}

func (t *Table) AddDefense(attackCard, defendCard Card) error {
	found := false
	for i := range t.pairs {
		if t.pairs[i].Attack == attackCard {
			found = true
			if t.pairs[i].Defend == nil {
				defendCopy := defendCard
				t.pairs[i].Defend = &defendCopy
				return nil
			}
		}
	}
	if found {
		return ErrCardAlreadyBeaten
	}
	return ErrAttackNotFound
}

func (t *Table) IsAllBeaten() bool {
	if len(t.pairs) == 0 {
		return false
	}
	for _, p := range t.pairs {
		if p.Defend == nil {
			return false
		}
	}
	return true
}

func (t *Table) HasUnbeaten() bool {
	for _, p := range t.pairs {
		if p.Defend == nil {
			return true
		}
	}
	return false
}

func (t *Table) UnbeatenCount() int {
	count := 0
	for _, p := range t.pairs {
		if p.Defend == nil {
			count++
		}
	}
	return count
}

func (t *Table) AllCards() []Card {
	cards := make([]Card, 0, len(t.pairs)*2)
	for _, p := range t.pairs {
		cards = append(cards, p.Attack)
		if p.Defend != nil {
			cards = append(cards, *p.Defend)
		}
	}
	return cards
}

func (t *Table) RanksOnTable() map[Rank]bool {
	ranks := make(map[Rank]bool)
	for _, p := range t.pairs {
		ranks[p.Attack.Rank] = true
		if p.Defend != nil {
			ranks[p.Defend.Rank] = true
		}
	}
	return ranks
}

func (t *Table) Clear() {
	t.pairs = nil
}
