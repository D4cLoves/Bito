package game

import (
	"errors"
	"testing"
)

func TestGamePhase_String(t *testing.T) {
	tests := []struct {
		phase GamePhase
		want  string
	}{
		{PhaseDealing, "dealing"},
		{PhaseBattle, "battle"},
		{PhaseGivingMore, "giving_more"},
		{PhaseFinished, "finished"},
		{GamePhase(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.want {
			t.Errorf("GamePhase(%d).String() = %q, want %q", tt.phase, got, tt.want)
		}
	}
}

func TestGameMode_String(t *testing.T) {
	tests := []struct {
		mode GameMode
		want string
	}{
		{ModePodkidnoy, "podkidnoy"},
		{ModePerevodnoy, "perevodnoy"},
		{GameMode(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("GameMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestNewGame_Validation(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")

	_, err := NewGame(GameMode(99), Deck36, []*Player{p1, p2})
	if !errors.Is(err, ErrInvalidGameMode) {
		t.Errorf("NewGame invalid mode err = %v, want %v", err, ErrInvalidGameMode)
	}

	_, err = NewGame(ModePodkidnoy, Deck36, nil)
	if !errors.Is(err, ErrNoPlayers) {
		t.Errorf("NewGame nil players err = %v, want %v", err, ErrNoPlayers)
	}

	_, err = NewGame(ModePodkidnoy, Deck36, []*Player{p1})
	if !errors.Is(err, ErrInvalidPlayersCount) {
		t.Errorf("NewGame 1 player err = %v, want %v", err, ErrInvalidPlayersCount)
	}

	manyPlayers := make([]*Player, 7)
	for i := 0; i < 7; i++ {
		manyPlayers[i] = NewPlayer(string(rune('a'+i)), "Player")
	}
	_, err = NewGame(ModePodkidnoy, Deck36, manyPlayers)
	if !errors.Is(err, ErrInvalidPlayersCount) {
		t.Errorf("NewGame 7 players err = %v, want %v", err, ErrInvalidPlayersCount)
	}

	_, err = NewGame(ModePodkidnoy, Deck36, []*Player{p1, nil})
	if !errors.Is(err, ErrNilPlayer) {
		t.Errorf("NewGame nil player err = %v, want %v", err, ErrNilPlayer)
	}

	pDup := NewPlayer("p1", "AliceDuplicate")
	_, err = NewGame(ModePodkidnoy, Deck36, []*Player{p1, pDup})
	if !errors.Is(err, ErrDuplicatePlayerID) {
		t.Errorf("NewGame duplicate id err = %v, want %v", err, ErrDuplicatePlayerID)
	}

	_, err = NewGame(ModePodkidnoy, DeckType(99), []*Player{p1, p2})
	if !errors.Is(err, ErrInvalidDeckType) {
		t.Errorf("NewGame invalid deck type err = %v, want %v", err, ErrInvalidDeckType)
	}
}

func TestNewGame_Success(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")

	game, err := NewGame(ModePodkidnoy, Deck36, []*Player{p1, p2})
	if err != nil {
		t.Fatalf("NewGame unexpected error: %v", err)
	}

	if game.Mode() != ModePodkidnoy {
		t.Errorf("Mode() = %v, want ModePodkidnoy", game.Mode())
	}
	if game.Phase() != PhaseBattle {
		t.Errorf("Phase() = %v, want PhaseBattle", game.Phase())
	}
	if len(game.Players()) != 2 {
		t.Fatalf("len(Players()) = %d, want 2", len(game.Players()))
	}
	if game.Table() == nil {
		t.Fatal("Table() is nil")
	}
	if game.Deck() == nil {
		t.Fatal("Deck() is nil")
	}
	if len(game.Discard()) != 0 {
		t.Errorf("len(Discard()) = %d, want 0", len(game.Discard()))
	}
	if !game.IsFirstRound() {
		t.Error("IsFirstRound() want true")
	}
	if game.WinnerID() != "" || game.LoserID() != "" || game.IsDraw() {
		t.Error("initial game should not have winner, loser or draw")
	}

	if p1.HandCount() != 6 || p2.HandCount() != 6 {
		t.Errorf("hands = (%d, %d), want (6, 6)", p1.HandCount(), p2.HandCount())
	}

	expectedLeft := 36 - 12
	if game.Deck().CardsLeft() != expectedLeft {
		t.Errorf("CardsLeft() = %d, want %d", game.Deck().CardsLeft(), expectedLeft)
	}

	if game.Attacker() == nil || game.Defender() == nil {
		t.Fatal("Attacker or Defender is nil")
	}
	if game.Attacker() == game.Defender() {
		t.Error("Attacker and Defender cannot be the same player in 2-player game")
	}
	if game.AttackerIdx() < 0 || game.AttackerIdx() > 1 {
		t.Errorf("AttackerIdx() = %d", game.AttackerIdx())
	}
	if game.DefenderIdx() != (game.AttackerIdx()+1)%2 {
		t.Errorf("DefenderIdx() = %d, want %d", game.DefenderIdx(), (game.AttackerIdx()+1)%2)
	}
}

func setupTestGame(mode GameMode) (*Game, *Player, *Player) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")
	game, _ := NewGame(mode, Deck36, []*Player{p1, p2})

	p1.ClearHand()
	p2.ClearHand()
	game.attackerIdx = 0
	game.defenderIdx = 1

	return game, p1, p2
}

func TestGame_Attack(t *testing.T) {
	game, p1, p2 := setupTestGame(ModePodkidnoy)
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c7Hearts := Card{Suit: Hearts, Rank: Rank7}

	p1.AddCard(c6Spades)
	p2.AddCard(c7Hearts)

	if err := game.Attack("unknown", c6Spades); !errors.Is(err, ErrPlayerNotFound) {
		t.Errorf("Attack unknown player err = %v, want %v", err, ErrPlayerNotFound)
	}

	pInactive := NewPlayer("inactive", "Ghost")
	pInactive.SetState(PlayerFinished)
	game.players = append(game.players, pInactive)
	if err := game.Attack("inactive", c6Spades); !errors.Is(err, ErrPlayerNotActive) {
		t.Errorf("Attack inactive player err = %v, want %v", err, ErrPlayerNotActive)
	}
	game.players = game.players[:2]

	if err := game.Attack(p2.ID(), c7Hearts); !errors.Is(err, ErrDefenderCannotToss) {
		t.Errorf("Defender attack err = %v, want %v", err, ErrDefenderCannotToss)
	}

	cNotInHand := Card{Suit: Clubs, Rank: RankAce}
	if err := game.Attack(p1.ID(), cNotInHand); !errors.Is(err, ErrCardNotInHand) {
		t.Errorf("Attack card not in hand err = %v, want %v", err, ErrCardNotInHand)
	}

	if err := game.Attack(p1.ID(), c6Spades); err != nil {
		t.Fatalf("Valid attack unexpected error: %v", err)
	}

	if p1.HandCount() != 0 {
		t.Errorf("p1 handCount after attack = %d, want 0", p1.HandCount())
	}
	if game.Table().PairsCount() != 1 {
		t.Errorf("table pairs count = %d, want 1", game.Table().PairsCount())
	}

	game.phase = PhaseFinished
	if err := game.Attack(p1.ID(), c6Spades); !errors.Is(err, ErrGameOver) {
		t.Errorf("Attack game over err = %v, want %v", err, ErrGameOver)
	}
	game.phase = PhaseDealing
	if err := game.Attack(p1.ID(), c6Spades); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Attack dealing phase err = %v, want %v", err, ErrWrongPhase)
	}
}

func TestGame_Defend(t *testing.T) {
	game, p1, p2 := setupTestGame(ModePodkidnoy)
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c7Spades := Card{Suit: Spades, Rank: Rank7}
	c5Spades := Card{Suit: Spades, Rank: Rank5}

	p1.AddCard(c6Spades)
	p2.AddCards([]Card{c7Spades, c5Spades})

	_ = game.Attack(p1.ID(), c6Spades)

	if err := game.Defend(p1.ID(), c6Spades, c7Spades); !errors.Is(err, ErrAttackerCannotDefend) {
		t.Errorf("Attacker defend err = %v, want %v", err, ErrAttackerCannotDefend)
	}

	cNotInHand := Card{Suit: Clubs, Rank: RankAce}
	if err := game.Defend(p2.ID(), c6Spades, cNotInHand); !errors.Is(err, ErrCardNotInHand) {
		t.Errorf("Defend card not in hand err = %v, want %v", err, ErrCardNotInHand)
	}

	if err := game.Defend(p2.ID(), c6Spades, c5Spades); !errors.Is(err, ErrCannotBeat) {
		t.Errorf("Defend cannot beat err = %v, want %v", err, ErrCannotBeat)
	}

	if err := game.Defend(p2.ID(), c6Spades, c7Spades); err != nil {
		t.Fatalf("Valid defend unexpected error: %v", err)
	}

	if p2.HandCount() != 1 {
		t.Errorf("p2 handCount after defend = %d, want 1", p2.HandCount())
	}
	if !game.Table().IsAllBeaten() {
		t.Error("Table IsAllBeaten want true")
	}

	game.phase = PhaseFinished
	if err := game.Defend(p2.ID(), c6Spades, c7Spades); !errors.Is(err, ErrGameOver) {
		t.Errorf("Defend game over err = %v, want %v", err, ErrGameOver)
	}
	game.phase = PhaseGivingMore
	if err := game.Defend(p2.ID(), c6Spades, c7Spades); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Defend giving more err = %v, want %v", err, ErrWrongPhase)
	}
}

func TestGame_Transfer(t *testing.T) {
	game, p1, p2 := setupTestGame(ModePodkidnoy)
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c6Clubs := Card{Suit: Clubs, Rank: Rank6}

	p1.AddCard(c6Spades)
	p2.AddCard(c6Clubs)
	_ = game.Attack(p1.ID(), c6Spades)

	if err := game.Transfer(p2.ID(), c6Clubs); !errors.Is(err, ErrTransferNotAllowed) {
		t.Errorf("Transfer in Podkidnoy err = %v, want %v", err, ErrTransferNotAllowed)
	}

	gamePerevod, p1P, p2P := setupTestGame(ModePerevodnoy)
	p3P := NewPlayer("p3", "Charlie")
	gamePerevod.players = append(gamePerevod.players, p3P)

	p1P.AddCard(c6Spades)
	p2P.AddCard(c6Clubs)
	p3P.AddCards([]Card{
		{Suit: Hearts, Rank: Rank9},
		{Suit: Hearts, Rank: Rank10},
	})

	_ = gamePerevod.Attack(p1P.ID(), c6Spades)

	if err := gamePerevod.Transfer(p1P.ID(), c6Clubs); !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("Attacker transfer err = %v, want %v", err, ErrNotYourTurn)
	}

	cNotInHand := Card{Suit: Hearts, Rank: Rank6}
	if err := gamePerevod.Transfer(p2P.ID(), cNotInHand); !errors.Is(err, ErrCardNotInHand) {
		t.Errorf("Transfer card not in hand err = %v, want %v", err, ErrCardNotInHand)
	}

	if err := gamePerevod.Transfer(p2P.ID(), c6Clubs); err != nil {
		t.Fatalf("Valid transfer unexpected error: %v", err)
	}

	if gamePerevod.AttackerIdx() != 1 || gamePerevod.DefenderIdx() != 2 {
		t.Errorf("After transfer (attackerIdx, defenderIdx) = (%d, %d), want (1, 2)",
			gamePerevod.AttackerIdx(), gamePerevod.DefenderIdx())
	}
	if gamePerevod.Table().PairsCount() != 2 {
		t.Errorf("Table pairs after transfer = %d, want 2", gamePerevod.Table().PairsCount())
	}

	gamePerevod.phase = PhaseFinished
	if err := gamePerevod.Transfer(p3P.ID(), c6Clubs); !errors.Is(err, ErrGameOver) {
		t.Errorf("Transfer game over err = %v, want %v", err, ErrGameOver)
	}
	gamePerevod.phase = PhaseGivingMore
	if err := gamePerevod.Transfer(p3P.ID(), c6Clubs); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Transfer wrong phase err = %v, want %v", err, ErrWrongPhase)
	}
}

func TestGame_Take_And_Pass(t *testing.T) {
	game, p1, p2 := setupTestGame(ModePodkidnoy)
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c6Hearts := Card{Suit: Hearts, Rank: Rank6}
	c7Spades := Card{Suit: Spades, Rank: Rank7}
	c8Spades := Card{Suit: Spades, Rank: Rank8}

	p1.AddCards([]Card{c6Spades, c6Hearts})
	p2.AddCards([]Card{c7Spades, c8Spades})

	if err := game.Take(p2.ID()); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Take on empty table err = %v, want %v", err, ErrWrongPhase)
	}

	_ = game.Attack(p1.ID(), c6Spades)

	if err := game.Take(p1.ID()); !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("Attacker Take err = %v, want %v", err, ErrNotYourTurn)
	}

	if err := game.Take(p2.ID()); err != nil {
		t.Fatalf("Valid Take unexpected error: %v", err)
	}
	if game.Phase() != PhaseGivingMore {
		t.Errorf("Phase after Take = %v, want PhaseGivingMore", game.Phase())
	}

	if err := game.Attack(p1.ID(), c6Hearts); err != nil {
		t.Fatalf("Toss in PhaseGivingMore unexpected error: %v", err)
	}

	if err := game.Pass(p2.ID()); !errors.Is(err, ErrDefenderCannotToss) {
		t.Errorf("Defender pass err = %v, want %v", err, ErrDefenderCannotToss)
	}

	if err := game.Pass(p1.ID()); err != nil {
		t.Fatalf("Attacker pass unexpected error: %v", err)
	}

	if p2.HandCount() != 4 {
		t.Errorf("p2 handCount after taking = %d, want 4 (2 initial + 2 taken)", p2.HandCount())
	}
	if game.Table().PairsCount() != 0 {
		t.Errorf("table pairs count after round = %d, want 0", game.Table().PairsCount())
	}
	if game.Phase() != PhaseBattle {
		t.Errorf("Phase after finishRound = %v, want PhaseBattle", game.Phase())
	}
	if game.AttackerIdx() != 0 || game.DefenderIdx() != 1 {
		t.Errorf("After defender took in 2p game: attacker=%d, defender=%d, want (0, 1)",
			game.AttackerIdx(), game.DefenderIdx())
	}
}

func TestGame_Bita_Pass_Refill(t *testing.T) {
	game, p1, p2 := setupTestGame(ModePodkidnoy)
	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c7Spades := Card{Suit: Spades, Rank: Rank7}

	p1.AddCard(c6Spades)
	p2.AddCard(c7Spades)

	_ = game.Attack(p1.ID(), c6Spades)

	if err := game.Pass(p1.ID()); !errors.Is(err, ErrTableNotBeaten) {
		t.Errorf("Pass with unbeaten cards err = %v, want %v", err, ErrTableNotBeaten)
	}

	_ = game.Defend(p2.ID(), c6Spades, c7Spades)

	if err := game.Pass(p1.ID()); err != nil {
		t.Fatalf("Valid Pass unexpected error: %v", err)
	}

	if len(game.Discard()) != 2 {
		t.Errorf("Discard count = %d, want 2", len(game.Discard()))
	}
	if p1.HandCount() != 6 || p2.HandCount() != 6 {
		t.Errorf("Hands after refill: p1=%d, p2=%d, want (6, 6)", p1.HandCount(), p2.HandCount())
	}
	if game.AttackerIdx() != 1 || game.DefenderIdx() != 0 {
		t.Errorf("After bita: attacker=%d, defender=%d, want (1, 0)", game.AttackerIdx(), game.DefenderIdx())
	}
	if game.IsFirstRound() {
		t.Error("isFirstRound after bita want false")
	}
}

func TestGame_Multiplayer_Pass(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")
	p3 := NewPlayer("p3", "Charlie")

	game, _ := NewGame(ModePodkidnoy, Deck36, []*Player{p1, p2, p3})
	p1.ClearHand()
	p2.ClearHand()
	p3.ClearHand()

	game.attackerIdx = 0
	game.defenderIdx = 1

	c6Spades := Card{Suit: Spades, Rank: Rank6}
	c7Spades := Card{Suit: Spades, Rank: Rank7}
	p1.AddCard(c6Spades)
	p2.AddCard(c7Spades)

	_ = game.Attack(p1.ID(), c6Spades)
	_ = game.Defend(p2.ID(), c6Spades, c7Spades)

	if err := game.Pass(p1.ID()); err != nil {
		t.Fatalf("p1 Pass unexpected error: %v", err)
	}
	if !game.HasPassed(p1.ID()) {
		t.Error("HasPassed(p1) want true")
	}
	if game.Table().PairsCount() != 1 {
		t.Error("round should not finish until all non-defenders pass")
	}

	if err := game.Pass(p1.ID()); !errors.Is(err, ErrAlreadyPassed) {
		t.Errorf("Duplicate pass err = %v, want %v", err, ErrAlreadyPassed)
	}

	if err := game.Pass(p3.ID()); err != nil {
		t.Fatalf("p3 Pass unexpected error: %v", err)
	}
	if game.Table().PairsCount() != 0 {
		t.Errorf("Table pairs after all pass = %d, want 0", game.Table().PairsCount())
	}
}

func TestGame_GameOver_Winner_Loser(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")
	game, _ := NewGame(ModePodkidnoy, Deck36, []*Player{p1, p2})

	p1.ClearHand()
	p2.ClearHand()
	game.attackerIdx = 0
	game.defenderIdx = 1

	for game.deck.CardsLeft() > 0 {
		_, _ = game.deck.Draw()
	}

	c6 := Card{Suit: Spades, Rank: Rank6}
	c7 := Card{Suit: Hearts, Rank: Rank7}
	p1.AddCard(c6)
	p2.AddCard(c7)

	_ = game.Attack(p1.ID(), c6)
	_ = game.Take(p2.ID())
	_ = game.Pass(p1.ID())

	if game.Phase() != PhaseFinished {
		t.Fatalf("Phase = %v, want PhaseFinished", game.Phase())
	}
	if game.WinnerID() != "p1" {
		t.Errorf("WinnerID = %v, want p1", game.WinnerID())
	}
	if game.LoserID() != "p2" {
		t.Errorf("LoserID = %v, want p2", game.LoserID())
	}
	if game.IsDraw() {
		t.Error("IsDraw want false")
	}
}

func TestGame_GameOver_Draw(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")
	game, _ := NewGame(ModePodkidnoy, Deck36, []*Player{p1, p2})

	p1.ClearHand()
	p2.ClearHand()
	game.attackerIdx = 0
	game.defenderIdx = 1

	for game.deck.CardsLeft() > 0 {
		_, _ = game.deck.Draw()
	}

	c6 := Card{Suit: Spades, Rank: Rank6}
	c7 := Card{Suit: Spades, Rank: Rank7}
	p1.AddCard(c6)
	p2.AddCard(c7)

	_ = game.Attack(p1.ID(), c6)
	_ = game.Defend(p2.ID(), c6, c7)
	_ = game.Pass(p1.ID())

	if game.Phase() != PhaseFinished {
		t.Fatalf("Phase = %v, want PhaseFinished", game.Phase())
	}
	if !game.IsDraw() {
		t.Error("IsDraw want true")
	}
}

func TestGame_EdgeCases(t *testing.T) {
	p1 := NewPlayer("p1", "Alice")
	p2 := NewPlayer("p2", "Bob")
	p3 := NewPlayer("p3", "Charlie")

	game, _ := NewGame(ModePerevodnoy, Deck36, []*Player{p1, p2, p3})
	p1.ClearHand()
	p2.ClearHand()
	p3.ClearHand()
	game.attackerIdx = 0
	game.defenderIdx = 1

	c6 := Card{Suit: Spades, Rank: Rank6}
	c7 := Card{Suit: Spades, Rank: Rank7}
	p1.AddCard(c6)
	p3.AddCard(c6)

	if err := game.Attack(p3.ID(), c6); !errors.Is(err, ErrNotYourTurn) {
		t.Errorf("Non-primary initial attack err = %v, want %v", err, ErrNotYourTurn)
	}

	if err := game.Pass(p1.ID()); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Pass on empty table err = %v, want %v", err, ErrWrongPhase)
	}
	if err := game.Pass("unknown"); !errors.Is(err, ErrPlayerNotFound) {
		t.Errorf("Pass unknown player err = %v, want %v", err, ErrPlayerNotFound)
	}

	pInactive := NewPlayer("inactive", "Ghost")
	pInactive.SetState(PlayerFinished)
	game.players = append(game.players, pInactive)
	if err := game.Pass("inactive"); !errors.Is(err, ErrPlayerNotActive) {
		t.Errorf("Pass inactive player err = %v, want %v", err, ErrPlayerNotActive)
	}
	game.players = game.players[:3]

	_ = game.Attack(p1.ID(), c6)

	p2.SetState(PlayerFinished)
	if err := game.Defend(p2.ID(), c6, c7); !errors.Is(err, ErrPlayerNotActive) {
		t.Errorf("Defend inactive defender err = %v, want %v", err, ErrPlayerNotActive)
	}
	if err := game.Transfer(p2.ID(), c6); !errors.Is(err, ErrPlayerNotActive) {
		t.Errorf("Transfer inactive defender err = %v, want %v", err, ErrPlayerNotActive)
	}
	p2.SetState(PlayerActive)

	p3.SetState(PlayerFinished)
	p2.AddCard(c6)
	p1.SetState(PlayerFinished)
	if err := game.Transfer(p2.ID(), c6); !errors.Is(err, ErrTransferNotAllowed) {
		t.Errorf("Transfer with no other active players err = %v, want %v", err, ErrTransferNotAllowed)
	}
	p1.SetState(PlayerActive)
	p3.SetState(PlayerActive)

	game.phase = PhaseFinished
	if err := game.Take(p2.ID()); !errors.Is(err, ErrGameOver) {
		t.Errorf("Take game over err = %v, want %v", err, ErrGameOver)
	}
	if err := game.Pass(p1.ID()); !errors.Is(err, ErrGameOver) {
		t.Errorf("Pass game over err = %v, want %v", err, ErrGameOver)
	}
	game.phase = PhaseBattle

	p2.AddCard(c7)
	_ = game.Defend(p2.ID(), c6, c7)
	if err := game.Take(p2.ID()); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Take when table all beaten err = %v, want %v", err, ErrWrongPhase)
	}

	for _, p := range game.players {
		p.SetState(PlayerFinished)
	}
	if idx := game.nextActivePlayerIdx(0); idx != -1 {
		t.Errorf("nextActivePlayerIdx when all finished = %d, want -1", idx)
	}
}
