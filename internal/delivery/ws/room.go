package ws

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"bito/internal/game"
)

const (
	turnDuration       = 20 * time.Second
	disconnectGraceDur = 30 * time.Second
)

// Room represents an isolated in-memory game room running its own Event Loop.
type Room struct {
	id       string
	gameMode game.GameMode
	deckType game.DeckType
	maxUsers int

	// Connected clients mapped by PlayerID.
	clients map[string]*Client

	// Active game engine instance (only accessed by room's event loop goroutine).
	game *game.Game

	// Event loop communication channels.
	register          chan *Client
	unregister        chan *Client
	incoming          chan IncomingMessage
	turnTimeout       chan struct{}
	disconnectTimeout chan string
	stop              chan struct{}

	// Timers.
	turnTimer        *time.Timer
	disconnectTimers map[string]*time.Timer

	mu     sync.RWMutex
	closed bool
}

// NewRoom creates a new game room.
func NewRoom(id string, mode game.GameMode, deckType game.DeckType, maxUsers int) *Room {
	return &Room{
		id:                id,
		gameMode:          mode,
		deckType:          deckType,
		maxUsers:          maxUsers,
		clients:           make(map[string]*Client),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		incoming:          make(chan IncomingMessage, 100),
		turnTimeout:       make(chan struct{}),
		disconnectTimeout: make(chan string),
		stop:              make(chan struct{}),
		disconnectTimers:  make(map[string]*time.Timer),
	}
}

// ID returns the room identifier.
func (r *Room) ID() string {
	return r.id
}

// RegisterClient safely requests registering a client into the room.
func (r *Room) RegisterClient(c *Client) {
	r.register <- c
}

// HasGraceTimer checks if a player currently has an active disconnect timer (thread-safe).
func (r *Room) HasGraceTimer(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.disconnectTimers[playerID]
	return exists
}

// Run is the single-threaded Event Loop of the room (Actor Pattern).
// All game logic and state modifications happen strictly within this goroutine.
func (r *Room) Run() {
	log.Printf("[Room %s] event loop started", r.id)
	defer func() {
		r.cleanup()
		log.Printf("[Room %s] event loop stopped", r.id)
	}()

	for {
		select {
		case <-r.stop:
			return

		case client := <-r.register:
			r.handleRegister(client)

		case client := <-r.unregister:
			r.handleUnregister(client)

		case msg := <-r.incoming:
			r.handleIncoming(msg)

		case <-r.turnTimeout:
			r.handleTurnTimeout()

		case playerID := <-r.disconnectTimeout:
			r.handleDisconnectTimeout(playerID)
		}
	}
}

// Close terminates the room's event loop.
func (r *Room) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed {
		r.closed = true
		close(r.stop)
	}
}

// handleRegister handles client connection or reconnection.
func (r *Room) handleRegister(client *Client) {
	playerID := client.ID()

	// 1. Check if this is a reconnecting player with an active grace period timer.
	r.mu.Lock()
	timer, exists := r.disconnectTimers[playerID]
	if exists {
		timer.Stop()
		delete(r.disconnectTimers, playerID)
		log.Printf("[Room %s] player %s reconnected within grace period", r.id, playerID)
	}
	r.mu.Unlock()

	r.clients[playerID] = client

	// 2. If the game is already in progress, immediately sync the state for this reconnected player.
	if r.game != nil && r.game.Phase() != game.PhaseFinished {
		r.sendGameStateTo(client)
		return
	}

	// 3. If game not started yet, check if we reached the required player count.
	if r.game == nil && len(r.clients) == r.maxUsers {
		r.startGame()
	}
}

// handleUnregister handles player disconnection and starts grace period.
func (r *Room) handleUnregister(client *Client) {
	playerID := client.ID()
	currentClient, exists := r.clients[playerID]
	if !exists || currentClient != client {
		return
	}

	delete(r.clients, playerID)
	close(client.send)

	// If game is not active, nothing more to do.
	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		return
	}

	log.Printf("[Room %s] player %s disconnected, starting 30s grace period", r.id, playerID)

	// Start 30-second Grace Period timer for reconnect.
	r.mu.Lock()
	r.disconnectTimers[playerID] = time.AfterFunc(disconnectGraceDur, func() {
		r.disconnectTimeout <- playerID
	})
	r.mu.Unlock()
}

// handleDisconnectTimeout handles expired grace period (technical defeat / fold).
func (r *Room) handleDisconnectTimeout(playerID string) {
	r.mu.Lock()
	delete(r.disconnectTimers, playerID)
	r.mu.Unlock()

	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		return
	}

	log.Printf("[Room %s] player %s grace period expired -> surrender", r.id, playerID)
	_ = r.game.Surrender(game.PlayerID(playerID))
	r.onGameStateChanged()
}

// startGame initializes the game engine and sends initial state.
func (r *Room) startGame() {
	players := make([]*game.Player, 0, len(r.clients))
	for _, c := range r.clients {
		players = append(players, game.NewPlayer(c.ID(), c.Name()))
	}

	newGame, err := game.NewGame(r.gameMode, r.deckType, players)
	if err != nil {
		log.Printf("[Room %s] failed to create game: %v", r.id, err)
		return
	}

	r.game = newGame
	log.Printf("[Room %s] game started with %d players", r.id, len(players))

	r.resetTurnTimer()
	r.broadcastGameState()
}

// handleIncoming parses and processes client actions sequentially.
func (r *Room) handleIncoming(msg IncomingMessage) {
	if r.game == nil {
		r.sendError(msg.Client, "game not started yet")
		return
	}

	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg.Data, &raw); err != nil {
		r.sendError(msg.Client, "invalid json format")
		return
	}

	var err error
	pID := game.PlayerID(msg.Client.ID())

	switch raw.Type {
	case TypeAttack:
		var p Message[AttackPayload]
		if err = json.Unmarshal(msg.Data, &p); err == nil {
			err = r.game.Attack(pID, p.Payload.AttackCard)
		}

	case TypeDefend:
		var p Message[DefendPayload]
		if err = json.Unmarshal(msg.Data, &p); err == nil {
			err = r.game.Defend(pID, p.Payload.AttackCard, p.Payload.DefendCard)
		}

	case TypeTransfer:
		var p Message[TransferPayload]
		if err = json.Unmarshal(msg.Data, &p); err == nil {
			err = r.game.Transfer(pID, p.Payload.Card)
		}

	case TypeTake:
		err = r.game.Take(pID)

	case TypePass:
		err = r.game.Pass(pID)

	case TypeSurrender:
		err = r.game.Surrender(pID)

	case TypeChat:
		r.handleChat(msg.Client, msg.Data)
		return

	default:
		err = errors.New("unknown message type: " + raw.Type)
	}

	if err != nil {
		r.sendError(msg.Client, err.Error())
		return
	}

	// Action applied successfully -> update timers & notify players.
	r.onGameStateChanged()
}

// onGameStateChanged handles post-action routine: check game over, reset timer, broadcast state.
func (r *Room) onGameStateChanged() {
	if r.game.Phase() == game.PhaseFinished {
		r.stopTurnTimer()
		r.broadcastGameOver()
		return
	}

	r.resetTurnTimer()
	r.broadcastGameState()
}

// handleTurnTimeout handles automatic decision when a player runs out of 20 seconds.
func (r *Room) handleTurnTimeout() {
	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		return
	}

	defender := r.game.Defender()
	attacker := r.game.Attacker()

	// If defender timed out during battle, defender takes the cards.
	if r.game.Phase() == game.PhaseBattle && r.game.Table().PairsCount() > 0 {
		log.Printf("[Room %s] turn timeout: defender %s takes cards", r.id, defender.ID())
		_ = r.game.Take(defender.ID())
		r.onGameStateChanged()
		return
	}

	// If attacker timed out:
	if attacker != nil {
		if r.game.Table().PairsCount() == 0 {
			// Attack with lowest non-trump card
			card, err := r.game.AutoAttack(attacker.ID())
			if err == nil {
				log.Printf("[Room %s] turn timeout: auto-attacked card %s", r.id, card)
				r.onGameStateChanged()
				return
			}
		} else {
			// Pass / bita
			_ = r.game.Pass(attacker.ID())
			r.onGameStateChanged()
			return
		}
	}
}

// resetTurnTimer resets the 20-second decision timer.
func (r *Room) resetTurnTimer() {
	r.stopTurnTimer()
	r.turnTimer = time.AfterFunc(turnDuration, func() {
		r.turnTimeout <- struct{}{}
	})
}

// stopTurnTimer stops the current turn timer.
func (r *Room) stopTurnTimer() {
	if r.turnTimer != nil {
		r.turnTimer.Stop()
		r.turnTimer = nil
	}
}

// broadcastGameState prepares personalized state views and sends to each client.
func (r *Room) broadcastGameState() {
	for _, client := range r.clients {
		r.sendGameStateTo(client)
	}
}

// sendGameStateTo sends personalized game state to a single client (zero-knowledge for opponent cards).
func (r *Room) sendGameStateTo(client *Client) {
	if r.game == nil {
		return
	}

	var myHand []game.Card
	playersView := make([]PlayerView, 0, len(r.game.Players()))

	for _, p := range r.game.Players() {
		if string(p.ID()) == client.ID() {
			myHand = p.Hand()
		}
		playersView = append(playersView, PlayerView{
			ID:        string(p.ID()),
			Name:      p.Name(),
			HandCount: p.HandCount(),
			IsActive:  p.IsActive(),
		})
	}

	var attackerID, defenderID string
	if atk := r.game.Attacker(); atk != nil {
		attackerID = string(atk.ID())
	}
	if def := r.game.Defender(); def != nil {
		defenderID = string(def.ID())
	}

	payload := GameStatePayload{
		Phase:       r.game.Phase().String(),
		TrumpCard:   r.game.Deck().Trump(),
		TrumpSuit:   r.game.Deck().Trump().Suit,
		CardsLeft:   r.game.Deck().CardsLeft(),
		Table:       r.game.Table().Pairs(),
		DiscardSize: len(r.game.Discard()),
		AttackerID:  attackerID,
		DefenderID:  defenderID,
		MyHand:      myHand,
		Players:     playersView,
	}

	data, err := NewMessage(TypeGameState, payload)
	if err == nil {
		client.Send(data)
	}
}

// broadcastGameOver notifies players about match outcome.
func (r *Room) broadcastGameOver() {
	payload := GameOverPayload{
		WinnerID: string(r.game.WinnerID()),
		LoserID:  string(r.game.LoserID()),
		IsDraw:   r.game.IsDraw(),
	}

	data, err := NewMessage(TypeGameOver, payload)
	if err != nil {
		return
	}

	for _, c := range r.clients {
		c.Send(data)
	}
}

// handleChat forwards chat message to all players at the table.
func (r *Room) handleChat(sender *Client, rawData []byte) {
	var chatMsg Message[ChatPayload]
	if err := json.Unmarshal(rawData, &chatMsg); err != nil {
		return
	}

	chatMsg.Payload.SenderID = sender.ID()
	chatMsg.Payload.SenderName = sender.Name()

	data, err := json.Marshal(chatMsg)
	if err != nil {
		return
	}

	for _, c := range r.clients {
		c.Send(data)
	}
}

// sendError sends a structured error message to a specific client.
func (r *Room) sendError(client *Client, errMsg string) {
	data, err := NewMessage(TypeError, ErrorPayload{Message: errMsg})
	if err == nil {
		client.Send(data)
	}
}

// cleanup stops all timers and notifies clients upon room shutdown.
func (r *Room) cleanup() {
	r.stopTurnTimer()
	r.mu.Lock()
	for _, timer := range r.disconnectTimers {
		timer.Stop()
	}
	r.mu.Unlock()
	for _, c := range r.clients {
		close(c.send)
	}
}
