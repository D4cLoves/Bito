package game

import "errors"

var (
	ErrInvalidCard         = errors.New("invalid card")
	ErrCannotBeat          = errors.New("defending card cannot beat attacking card")
	ErrCannotToss          = errors.New("card rank is not present on table")
	ErrTransferNotAllowed  = errors.New("transfer is not allowed")
	ErrAttackLimitExceeded = errors.New("attack limit exceeded")
)

func CanBeat(attack, defend Card, trump Suit) bool {
	if attack.Rank < Rank2 || attack.Rank > RankAce || defend.Rank < Rank2 || defend.Rank > RankAce {
		return false
	}
	if attack.Suit > Diamonds || defend.Suit > Diamonds || trump > Diamonds {
		return false
	}

	if defend.Suit == trump && attack.Suit != trump {
		return true
	}

	if defend.Suit == attack.Suit {
		return defend.Rank > attack.Rank
	}

	return false
}

func CanInitialAttack(table *Table, card Card) bool {
	return table.PairsCount() == 0 && card.Rank >= Rank2 && card.Rank <= RankAce && card.Suit <= Diamonds
}

func CanToss(table *Table, card Card) bool {
	if table.PairsCount() == 0 || card.Rank < Rank2 || card.Rank > RankAce || card.Suit > Diamonds {
		return false
	}
	return table.RanksOnTable()[card.Rank]
}

func CanDefend(table *Table, attackCard, defendCard Card, trump Suit) bool {
	if !CanBeat(attackCard, defendCard, trump) {
		return false
	}
	return table.HasUnbeatenAttack(attackCard)
}

func CanTransfer(table *Table, card Card, nextDefenderHandCount int) bool {
	if table.PairsCount() == 0 || card.Rank < Rank2 || card.Rank > RankAce || card.Suit > Diamonds {
		return false
	}

	if table.UnbeatenCount() != table.PairsCount() {
		return false
	}

	ranks := table.RanksOnTable()
	if len(ranks) != 1 || !ranks[card.Rank] {
		return false
	}

	if table.PairsCount()+1 > nextDefenderHandCount {
		return false
	}

	return true
}

func MaxRoundLimit(isFirstRound bool) int {
	if isFirstRound {
		return 5
	}
	return 6
}

func CanAddAttack(table *Table, defenderHandCount int, isFirstRound bool) bool {
	if defenderHandCount <= 0 {
		return false
	}
	if table.PairsCount() >= MaxRoundLimit(isFirstRound) {
		return false
	}
	if table.UnbeatenCount() >= defenderHandCount {
		return false
	}
	return true
}

func ValidateAttack(table *Table, card Card, defenderHandCount int, isFirstRound bool) error {
	if card.Rank < Rank2 || card.Rank > RankAce || card.Suit > Diamonds {
		return ErrInvalidCard
	}

	if !CanAddAttack(table, defenderHandCount, isFirstRound) {
		return ErrAttackLimitExceeded
	}

	if table.PairsCount() == 0 {
		return nil
	}

	if !CanToss(table, card) {
		return ErrCannotToss
	}

	return nil
}

func ValidateDefense(table *Table, attackCard, defendCard Card, trump Suit) error {
	if attackCard.Rank < Rank2 || attackCard.Rank > RankAce || attackCard.Suit > Diamonds ||
		defendCard.Rank < Rank2 || defendCard.Rank > RankAce || defendCard.Suit > Diamonds ||
		trump > Diamonds {
		return ErrInvalidCard
	}

	if !table.HasUnbeatenAttack(attackCard) {
		for _, p := range table.Pairs() {
			if p.Attack == attackCard && p.Defend != nil {
				return ErrCardAlreadyBeaten
			}
		}
		return ErrAttackNotFound
	}

	if !CanBeat(attackCard, defendCard, trump) {
		return ErrCannotBeat
	}

	return nil
}

func ValidateTransfer(table *Table, card Card, nextDefenderHandCount int, isFirstRound bool) error {
	if card.Rank < Rank2 || card.Rank > RankAce || card.Suit > Diamonds {
		return ErrInvalidCard
	}

	if table.PairsCount() == 0 {
		return ErrTransferNotAllowed
	}

	if table.UnbeatenCount() != table.PairsCount() {
		return ErrCardAlreadyBeaten
	}

	if table.PairsCount() >= MaxRoundLimit(isFirstRound) {
		return ErrAttackLimitExceeded
	}

	ranks := table.RanksOnTable()
	if len(ranks) != 1 || !ranks[card.Rank] {
		return ErrTransferNotAllowed
	}

	if table.PairsCount()+1 > nextDefenderHandCount {
		return ErrTransferNotAllowed
	}

	return nil
}

func FindLowestTrump(hand []Card, trump Suit) (Card, bool) {
	if trump > Diamonds {
		return Card{}, false
	}

	var lowest Card
	found := false

	for _, card := range hand {
		if card.Rank < Rank2 || card.Rank > RankAce || card.Suit > Diamonds {
			continue
		}
		if card.Suit == trump {
			if !found || card.Rank < lowest.Rank {
				lowest = card
				found = true
			}
		}
	}
	return lowest, found
}

func DetermineFirstAttacker(hands map[string][]Card, trump Suit) (string, Card, bool) {
	if trump > Diamonds {
		return "", Card{}, false
	}

	var (
		bestPlayer string
		lowestCard Card
		found      bool
	)

	for playerID, hand := range hands {
		card, ok := FindLowestTrump(hand, trump)
		if !ok {
			continue
		}
		if !found || card.Rank < lowestCard.Rank {
			lowestCard = card
			bestPlayer = playerID
			found = true
		}
	}

	return bestPlayer, lowestCard, found
}
