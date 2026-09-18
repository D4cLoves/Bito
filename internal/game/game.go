package game

import "errors"

var (
	ErrNoPlayers            = errors.New("no players")
	ErrInvalidPlayersCount  = errors.New("invalid players count")
	ErrInvalidGameMode      = errors.New("invalid game mode")
	ErrNilPlayer            = errors.New("nil player")
	ErrDuplicatePlayerID    = errors.New("duplicate player id")
	ErrGameOver             = errors.New("game is over")
	ErrWrongPhase           = errors.New("action not allowed in current phase")
	ErrPlayerNotFound       = errors.New("player not found")
	ErrPlayerNotActive      = errors.New("player is not active")
	ErrNotYourTurn          = errors.New("not your turn")
	ErrDefenderCannotToss   = errors.New("defender cannot attack or toss")
	ErrAttackerCannotDefend = errors.New("only defender can defend")
	ErrTableNotBeaten       = errors.New("table is not all beaten")
	ErrAlreadyPassed        = errors.New("player has already passed")
)

type GamePhase uint8

const (
	PhaseDealing GamePhase = iota
	PhaseBattle
	PhaseGivingMore
	PhaseFinished
)

func (p GamePhase) String() string {
	switch p {
	case PhaseDealing:
		return "dealing"
	case PhaseBattle:
		return "battle"
	case PhaseGivingMore:
		return "giving_more"
	case PhaseFinished:
		return "finished"
	default:
		return "unknown"
	}
}

type GameMode uint8

const (
	ModePodkidnoy  GameMode = 1
	ModePerevodnoy GameMode = 2
)

func (m GameMode) String() string {
	switch m {
	case ModePodkidnoy:
		return "podkidnoy"
	case ModePerevodnoy:
		return "perevodnoy"
	default:
		return "unknown"
	}
}

type Game struct {
	gameMode     GameMode
	phase        GamePhase
	players      []*Player
	deck         *Deck
	table        *Table
	discard      []Card
	attackerIdx  int
	defenderIdx  int
	isFirstRound bool
	winnerID     PlayerID
	loserID      PlayerID
	isDraw       bool
	passed       map[PlayerID]bool
}

func NewGame(mode GameMode, deckType DeckType, players []*Player) (*Game, error) {
	if mode != ModePodkidnoy && mode != ModePerevodnoy {
		return nil, ErrInvalidGameMode
	}

	if len(players) == 0 {
		return nil, ErrNoPlayers
	}

	if len(players) < 2 || len(players) > 6 {
		return nil, ErrInvalidPlayersCount
	}

	seen := make(map[PlayerID]bool, len(players))
	for _, p := range players {
		if p == nil {
			return nil, ErrNilPlayer
		}
		if seen[p.ID()] {
			return nil, ErrDuplicatePlayerID
		}
		seen[p.ID()] = true
	}

	deck, err := NewDeck(deckType)
	if err != nil {
		return nil, err
	}

	for _, p := range players {
		p.ClearHand()
		p.SetState(PlayerActive)

		cards, err := deck.DrawMany(6)
		if err != nil {
			return nil, err
		}
		p.AddCards(cards)
	}

	trumpSuit := deck.Trump().Suit
	attackerIdx := 0
	var lowestTrump Card
	foundTrump := false

	for i, p := range players {
		card, ok := p.LowestTrump(trumpSuit)
		if ok {
			if !foundTrump || card.Rank < lowestTrump.Rank {
				lowestTrump = card
				attackerIdx = i
				foundTrump = true
			}
		}
	}

	defenderIdx := (attackerIdx + 1) % len(players)

	playersCopy := make([]*Player, len(players))
	copy(playersCopy, players)

	return &Game{
		gameMode:     mode,
		phase:        PhaseBattle,
		players:      playersCopy,
		deck:         deck,
		table:        NewTable(),
		discard:      make([]Card, 0),
		attackerIdx:  attackerIdx,
		defenderIdx:  defenderIdx,
		isFirstRound: true,
		passed:       make(map[PlayerID]bool),
	}, nil
}

func (g *Game) Mode() GameMode {
	return g.gameMode
}

func (g *Game) Phase() GamePhase {
	return g.phase
}

func (g *Game) Players() []*Player {
	playersCopy := make([]*Player, len(g.players))
	copy(playersCopy, g.players)
	return playersCopy
}

func (g *Game) Attacker() *Player {
	return g.players[g.attackerIdx]
}

func (g *Game) Defender() *Player {
	return g.players[g.defenderIdx]
}

func (g *Game) AttackerIdx() int {
	return g.attackerIdx
}

func (g *Game) DefenderIdx() int {
	return g.defenderIdx
}

func (g *Game) Table() *Table {
	return g.table
}

func (g *Game) Deck() *Deck {
	return g.deck
}

func (g *Game) Discard() []Card {
	discardCopy := make([]Card, len(g.discard))
	copy(discardCopy, g.discard)
	return discardCopy
}

func (g *Game) IsFirstRound() bool {
	return g.isFirstRound
}

func (g *Game) WinnerID() PlayerID {
	return g.winnerID
}

func (g *Game) LoserID() PlayerID {
	return g.loserID
}

func (g *Game) IsDraw() bool {
	return g.isDraw
}

func (g *Game) HasPassed(id PlayerID) bool {
	return g.passed[id]
}

func (g *Game) PlayerByID(id PlayerID) (*Player, int) {
	for i, p := range g.players {
		if p.ID() == id {
			return p, i
		}
	}
	return nil, -1
}

func (g *Game) nextActivePlayerIdx(fromIdx int) int {
	for i := 1; i <= len(g.players); i++ {
		idx := (fromIdx + i) % len(g.players)
		if g.players[idx].IsActive() {
			return idx
		}
	}
	return -1
}

func (g *Game) Attack(playerID PlayerID, card Card) error {
	if g.phase == PhaseFinished {
		return ErrGameOver
	}
	if g.phase != PhaseBattle && g.phase != PhaseGivingMore {
		return ErrWrongPhase
	}

	player, _ := g.PlayerByID(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}
	if !player.IsActive() {
		return ErrPlayerNotActive
	}

	if playerID == g.Defender().ID() {
		return ErrDefenderCannotToss
	}

	if g.phase == PhaseBattle && g.table.PairsCount() == 0 && playerID != g.Attacker().ID() {
		return ErrNotYourTurn
	}

	if !player.HasCard(card) {
		return ErrCardNotInHand
	}

	if err := ValidateAttack(g.table, card, g.Defender().HandCount(), g.isFirstRound); err != nil {
		return err
	}

	if err := player.RemoveCard(card); err != nil {
		return err
	}

	g.table.AddAttack(card)
	g.passed = make(map[PlayerID]bool)
	return nil
}

func (g *Game) Defend(playerID PlayerID, attackCard, defendCard Card) error {
	if g.phase == PhaseFinished {
		return ErrGameOver
	}
	if g.phase != PhaseBattle {
		return ErrWrongPhase
	}

	if playerID != g.Defender().ID() {
		return ErrAttackerCannotDefend
	}

	defender := g.Defender()
	if !defender.IsActive() {
		return ErrPlayerNotActive
	}

	if !defender.HasCard(defendCard) {
		return ErrCardNotInHand
	}

	if err := ValidateDefense(g.table, attackCard, defendCard, g.deck.Trump().Suit); err != nil {
		return err
	}

	if err := defender.RemoveCard(defendCard); err != nil {
		return err
	}

	if err := g.table.AddDefense(attackCard, defendCard); err != nil {
		return err
	}

	g.passed = make(map[PlayerID]bool)
	return nil
}

func (g *Game) Transfer(playerID PlayerID, card Card) error {
	if g.gameMode != ModePerevodnoy {
		return ErrTransferNotAllowed
	}
	if g.phase == PhaseFinished {
		return ErrGameOver
	}
	if g.phase != PhaseBattle {
		return ErrWrongPhase
	}

	if playerID != g.Defender().ID() {
		return ErrNotYourTurn
	}

	defender := g.Defender()
	if !defender.IsActive() {
		return ErrPlayerNotActive
	}

	nextDefenderIdx := g.nextActivePlayerIdx(g.defenderIdx)
	if nextDefenderIdx == -1 || nextDefenderIdx == g.defenderIdx {
		return ErrTransferNotAllowed
	}
	nextDefender := g.players[nextDefenderIdx]

	if !defender.HasCard(card) {
		return ErrCardNotInHand
	}

	if err := ValidateTransfer(g.table, card, nextDefender.HandCount(), g.isFirstRound); err != nil {
		return err
	}

	if err := defender.RemoveCard(card); err != nil {
		return err
	}

	g.table.AddAttack(card)
	g.attackerIdx = g.defenderIdx
	g.defenderIdx = nextDefenderIdx
	g.passed = make(map[PlayerID]bool)
	return nil
}

func (g *Game) Take(playerID PlayerID) error {
	if g.phase == PhaseFinished {
		return ErrGameOver
	}
	if g.phase != PhaseBattle {
		return ErrWrongPhase
	}

	if playerID != g.Defender().ID() {
		return ErrNotYourTurn
	}

	if g.table.PairsCount() == 0 || !g.table.HasUnbeaten() {
		return ErrWrongPhase
	}

	g.phase = PhaseGivingMore
	g.passed = make(map[PlayerID]bool)
	return nil
}

func (g *Game) Pass(playerID PlayerID) error {
	if g.phase == PhaseFinished {
		return ErrGameOver
	}
	if g.phase != PhaseBattle && g.phase != PhaseGivingMore {
		return ErrWrongPhase
	}

	player, _ := g.PlayerByID(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}
	if !player.IsActive() {
		return ErrPlayerNotActive
	}

	if playerID == g.Defender().ID() {
		return ErrDefenderCannotToss
	}

	if g.phase == PhaseBattle {
		if g.table.PairsCount() == 0 {
			return ErrWrongPhase
		}
		if !g.table.IsAllBeaten() {
			return ErrTableNotBeaten
		}
	}

	if g.passed[playerID] {
		return ErrAlreadyPassed
	}
	g.passed[playerID] = true

	allPassed := true
	for i, p := range g.players {
		if i == g.defenderIdx || !p.IsActive() {
			continue
		}
		if !g.passed[p.ID()] {
			allPassed = false
			break
		}
	}

	if !allPassed {
		return nil
	}

	g.finishRound()
	return nil
}

func (g *Game) finishRound() {
	isBeaten := g.phase == PhaseBattle
	defender := g.Defender()

	if isBeaten {
		g.discard = append(g.discard, g.table.AllCards()...)
	} else {
		defender.AddCards(g.table.AllCards())
	}
	g.table.Clear()

	g.refillHands(isBeaten)
	g.checkGameOver()

	if g.phase == PhaseFinished {
		return
	}

	var newAttackerIdx int
	if isBeaten {
		if defender.IsActive() {
			newAttackerIdx = g.defenderIdx
		} else {
			newAttackerIdx = g.nextActivePlayerIdx(g.defenderIdx)
		}
	} else {
		newAttackerIdx = g.nextActivePlayerIdx(g.defenderIdx)
	}

	newDefenderIdx := g.nextActivePlayerIdx(newAttackerIdx)

	g.attackerIdx = newAttackerIdx
	g.defenderIdx = newDefenderIdx
	g.isFirstRound = false
	g.phase = PhaseBattle
	g.passed = make(map[PlayerID]bool)
}

func (g *Game) refillHands(isBeaten bool) {
	g.refillPlayer(g.attackerIdx)

	for i := 1; i < len(g.players); i++ {
		idx := (g.attackerIdx + i) % len(g.players)
		if idx == g.defenderIdx || !g.players[idx].IsActive() {
			continue
		}
		g.refillPlayer(idx)
	}

	if isBeaten && g.players[g.defenderIdx].IsActive() {
		g.refillPlayer(g.defenderIdx)
	}
}

func (g *Game) refillPlayer(idx int) {
	p := g.players[idx]
	needed := 6 - p.HandCount()
	if needed <= 0 || g.deck.CardsLeft() == 0 {
		return
	}
	cards, _ := g.deck.DrawMany(needed)
	p.AddCards(cards)
}

func (g *Game) checkGameOver() {
	if g.deck.CardsLeft() > 0 {
		return
	}

	for _, p := range g.players {
		if p.IsActive() && p.HandCount() == 0 {
			p.SetState(PlayerFinished)
			if g.winnerID == "" {
				g.winnerID = p.ID()
			}
		}
	}

	activeCount := 0
	var lastActive *Player
	for _, p := range g.players {
		if p.IsActive() {
			activeCount++
			lastActive = p
		}
	}

	if activeCount == 0 {
		g.phase = PhaseFinished
		g.isDraw = true
		return
	}

	if activeCount == 1 {
		g.phase = PhaseFinished
		g.loserID = lastActive.ID()
		return
	}
}
