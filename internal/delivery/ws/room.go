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

type IncomingMessage struct {
	Client *Client
	Data   []byte
}

type Room struct {
	id                string
	gameMode          game.GameMode
	deckType          game.DeckType
	maxUsers          int
	clients           map[string]*Client
	game              *game.Game
	register          chan *Client
	unregister        chan *Client
	incoming          chan IncomingMessage
	turnTimeout       chan struct{}
	disconnectTimeout chan string
	stop              chan struct{}
	turnTimer         *time.Timer
	disconnectTimers  map[string]*time.Timer
	mu                sync.RWMutex
	closed            bool
}

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
		turnTimeout:       make(chan struct{}, 1),
		disconnectTimeout: make(chan string, 10),
		stop:              make(chan struct{}),
		disconnectTimers:  make(map[string]*time.Timer),
	}
}

func (r *Room) ID() string {
	return r.id
}

func (r *Room) RegisterClient(c *Client) bool {
	select {
	case r.register <- c:
		return true
	case <-r.stop:
		return false
	}
}

func (r *Room) HasGraceTimer(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.disconnectTimers[playerID]
	return exists
}

func (r *Room) Run() {
	log.Printf("[Room %s] цикл событий запущен", r.id)
	defer func() {
		r.cleanup()
		log.Printf("[Room %s] цикл событий остановлен", r.id)
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

func (r *Room) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed {
		r.closed = true
		close(r.stop)
	}
}

func (r *Room) handleRegister(client *Client) {
	playerID := client.ID()

	r.mu.Lock()
	timer, exists := r.disconnectTimers[playerID]
	if exists {
		timer.Stop()
		delete(r.disconnectTimers, playerID)
		log.Printf("[Room %s] игрок %s вернулся в игру во время льготного периода", r.id, playerID)
	}
	r.mu.Unlock()

	if r.game != nil {
		if r.game.Phase() != game.PhaseFinished {
			player, _ := r.game.PlayerByID(game.PlayerID(playerID))
			if player == nil {
				r.sendError(client, "партия уже идет, стол заполнен")
				client.Close()
				return
			}

			r.clients[playerID] = client

			notifyMsg, err := NewMessage(TypePlayerReconnected, PlayerStatusPayload{PlayerID: playerID})
			if err == nil {
				for id, c := range r.clients {
					if id != playerID {
						c.Send(notifyMsg)
					}
				}
			}

			r.sendGameStateTo(client)
			return
		} else {
			r.sendError(client, "партия уже завершена")
			client.Close()
			return
		}
	}

	if len(r.clients) >= r.maxUsers {
		if _, alreadyPresent := r.clients[playerID]; !alreadyPresent {
			r.sendError(client, "стол уже заполнен")
			client.Close()
			return
		}
	}

	r.clients[playerID] = client

	if len(r.clients) == r.maxUsers {
		r.startGame()
	}
}

func (r *Room) handleUnregister(client *Client) {
	playerID := client.ID()
	currentClient, exists := r.clients[playerID]
	if !exists || currentClient != client {
		return
	}

	delete(r.clients, playerID)
	client.Close()

	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		if len(r.clients) == 0 {
			r.Close()
		}
		return
	}

	log.Printf("[Room %s] игрок %s потерял связь, запущен таймер ожидания 30 сек", r.id, playerID)

	notifyMsg, err := NewMessage(TypePlayerDisconnected, PlayerStatusPayload{PlayerID: playerID})
	if err == nil {
		for _, c := range r.clients {
			c.Send(notifyMsg)
		}
	}

	r.mu.Lock()
	r.disconnectTimers[playerID] = time.AfterFunc(disconnectGraceDur, func() {
		select {
		case r.disconnectTimeout <- playerID:
		case <-r.stop:
		}
	})
	r.mu.Unlock()
}

func (r *Room) handleDisconnectTimeout(playerID string) {
	r.mu.Lock()
	delete(r.disconnectTimers, playerID)
	r.mu.Unlock()

	if _, connected := r.clients[playerID]; connected {
		return
	}

	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		return
	}

	log.Printf("[Room %s] время ожидания игрока %s истекло -> автоматическая сдача", r.id, playerID)
	_ = r.game.Surrender(game.PlayerID(playerID))
	r.onGameStateChanged()
}

func (r *Room) startGame() {
	players := make([]*game.Player, 0, len(r.clients))
	for _, c := range r.clients {
		players = append(players, game.NewPlayer(c.ID(), c.Name()))
	}

	newGame, err := game.NewGame(r.gameMode, r.deckType, players)
	if err != nil {
		log.Printf("[Room %s] ошибка создания игры: %v", r.id, err)
		return
	}

	r.game = newGame
	log.Printf("[Room %s] партия началась, участников: %d", r.id, len(players))

	r.resetTurnTimer()
	r.broadcastGameState()
}

func (r *Room) handleIncoming(msg IncomingMessage) {
	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg.Data, &raw); err != nil {
		r.sendError(msg.Client, "неверный формат JSON")
		return
	}

	if raw.Type == TypeChat {
		r.handleChat(msg.Client, msg.Data)
		return
	}

	if r.game == nil {
		r.sendError(msg.Client, "игра еще не началась")
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

	default:
		err = errors.New("неизвестный тип действия: " + raw.Type)
	}

	if err != nil {
		r.sendError(msg.Client, err.Error())
		return
	}

	r.onGameStateChanged()
}

func (r *Room) onGameStateChanged() {
	if r.game.Phase() == game.PhaseFinished {
		r.stopTurnTimer()
		r.broadcastGameOver()
		time.AfterFunc(30*time.Second, func() {
			r.Close()
		})
		return
	}

	r.resetTurnTimer()
	r.broadcastGameState()
}

func (r *Room) handleTurnTimeout() {
	if r.game == nil || r.game.Phase() == game.PhaseFinished {
		return
	}

	defender := r.game.Defender()
	attacker := r.game.Attacker()

	if r.game.Phase() == game.PhaseBattle && r.game.Table().HasUnbeaten() && defender != nil {
		log.Printf("[Room %s] таймаут хода: защитник %s автоматически берет карты", r.id, defender.ID())
		_ = r.game.Take(defender.ID())
		r.onGameStateChanged()
		return
	}

	if r.game.Phase() == game.PhaseBattle && r.game.Table().PairsCount() == 0 && attacker != nil {
		card, err := r.game.AutoAttack(attacker.ID())
		if err == nil {
			log.Printf("[Room %s] таймаут хода: авто-ход картой %s", r.id, card)
			r.onGameStateChanged()
			return
		}
	}

	if (r.game.Table().IsAllBeaten() && r.game.Phase() == game.PhaseBattle) || r.game.Phase() == game.PhaseGivingMore {
		for _, p := range r.game.Players() {
			if defender != nil && p.ID() == defender.ID() {
				continue
			}
			if p.IsActive() {
				_ = r.game.Pass(p.ID())
			}
		}
		r.onGameStateChanged()
		return
	}
}

func (r *Room) resetTurnTimer() {
	r.stopTurnTimer()
	r.turnTimer = time.AfterFunc(turnDuration, func() {
		select {
		case r.turnTimeout <- struct{}{}:
		case <-r.stop:
		}
	})
}

func (r *Room) stopTurnTimer() {
	if r.turnTimer != nil {
		r.turnTimer.Stop()
		r.turnTimer = nil
	}
}

func (r *Room) broadcastGameState() {
	for _, client := range r.clients {
		r.sendGameStateTo(client)
	}
}

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

func (r *Room) sendError(client *Client, errMsg string) {
	data, err := NewMessage(TypeError, ErrorPayload{Message: errMsg})
	if err == nil {
		client.Send(data)
	}
}

func (r *Room) cleanup() {
	r.stopTurnTimer()
	r.mu.Lock()
	for _, timer := range r.disconnectTimers {
		timer.Stop()
	}
	r.disconnectTimers = make(map[string]*time.Timer)
	r.mu.Unlock()

	for _, c := range r.clients {
		c.Close()
	}
}
