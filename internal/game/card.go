package game

import "strconv"

type Suit uint8

const (
	Spades   Suit = iota // пики
	Clubs                // трефы
	Hearts               // червы
	Diamonds             // бубны
)

type Rank uint8

const (
	RankInvalid Rank = iota // 0
	_                       // 1
	Rank2                   // 2
	Rank3                   // 3
	Rank4                   // 4
	Rank5                   // 5
	Rank6                   // 6
	Rank7                   // 7
	Rank8                   // 8
	Rank9                   // 9
	Rank10                  // 10
	RankJack                // 11 (валет)
	RankQueen               // 12 (дама)
	RankKing                // 13 (король)
	RankAce                 // 14 (туз)
)

type Card struct {
	Suit Suit `json:"suit"`
	Rank Rank `json:"rank"`
}

func (s Suit) String() string {
	switch s {
	case Spades:
		return "♠"
	case Clubs:
		return "♣"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	default:
		return "?"
	}
}

func (r Rank) String() string {
	switch {
	case r >= Rank2 && r <= Rank10:
		return strconv.Itoa(int(r))
	case r == RankJack:
		return "J"
	case r == RankQueen:
		return "Q"
	case r == RankKing:
		return "K"
	case r == RankAce:
		return "A"
	default:
		return "?"
	}
}

func (c Card) String() string {
	return c.Rank.String() + c.Suit.String()
}
