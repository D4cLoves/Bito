package ws

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bito/internal/economy"
	"bito/internal/game"
	"bito/internal/model"
)

const (
	duelPlayers     = 2
	defaultDuelMode = game.ModePodkidnoy
	defaultDuelDeck = game.Deck36
	reservationTTL  = 15 * time.Second
	settleAttempts  = 3
	settleRetryWait = time.Second
)

const (
	turnDuration       = 20 * time.Second
	disconnectGraceDur = 30 * time.Second
	// emptyRoomTTL closes a table created via REST that nobody joined.
	emptyRoomTTL = reservationTTL
	chatMaxRunes = 200
	chatCooldown = 700 * time.Millisecond
)

// RoomInfo is the lobby view of a table; safe to build from any goroutine.
type RoomInfo struct {
	ID          string    `json:"id"`
	Mode        string    `json:"mode"`
	Deck        int       `json:"deck"`
	MaxPlayers  int       `json:"maxPlayers"`
	Players     int       `json:"players"`
	Stake       int64     `json:"stake"`
	Matchmaking bool      `json:"matchmaking"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RoomSettings struct {
	Mode        game.GameMode
	Deck        game.DeckType
	MaxPlayers  int
	Stake       int64
	CreatedBy   string
	Matchmaking bool
}

type IncomingMessage struct {
	Client *Client
	Data   []byte
}

type Room struct {
	id                string
	gameMode          game.GameMode
	deckType          game.DeckType
	maxUsers          int
	stake             int64
	matchmaking       bool
	bank              Bank
	clients           map[string]*Client
	seated            map[string]bool
	reservations      map[string]time.Time
	game              *game.Game
	payouts           map[string]int64
	draw              bool
	register          chan *Client
	unregister        chan *Client
	incoming          chan IncomingMessage
	turnTimeout       chan struct{}
	disconnectTimeout chan string
	stop              chan struct{}
	turnTimer         *time.Timer
	turnEndsAt        time.Time
	disconnectTimers  map[string]*time.Timer
	lastChatAt        map[string]time.Time
	settleRetryDelay  time.Duration
	mu                sync.RWMutex
	closed            bool
	createdBy         string
	createdAt         time.Time
	playersCount      atomic.Int32
	started           atomic.Bool
	finished          atomic.Bool
}

func NewRoom(id string, s RoomSettings, bank Bank) *Room {
	if bank == nil {
		bank = freeBank{}
	}
	return &Room{
		id:                id,
		gameMode:          s.Mode,
		deckType:          s.Deck,
		maxUsers:          s.MaxPlayers,
		stake:             s.Stake,
		matchmaking:       s.Matchmaking,
		bank:              bank,
		createdBy:         s.CreatedBy,
		createdAt:         time.Now(),
		clients:           make(map[string]*Client),
		seated:            make(map[string]bool),
		reservations:      make(map[string]time.Time),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		incoming:          make(chan IncomingMessage, 100),
		turnTimeout:       make(chan struct{}, 1),
		disconnectTimeout: make(chan string, 10),
		stop:              make(chan struct{}),
		disconnectTimers:  make(map[string]*time.Timer),
		lastChatAt:        make(map[string]time.Time),
		settleRetryDelay:  settleRetryWait,
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

func (r *Room) Started() bool {
	return r.started.Load()
}

func (r *Room) PlayersCount() int {
	return int(r.playersCount.Load())
}

func (r *Room) Stake() int64 {
	return r.stake
}

func (r *Room) Holds(playerID string) bool {
	if r.finished.Load() {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return !r.closed && r.seated[playerID]
}

func (r *Room) Reserve(playerID string) {
	r.mu.Lock()
	r.reservations[playerID] = time.Now().Add(reservationTTL)
	r.mu.Unlock()
}

func (r *Room) Unreserve(playerID string) {
	r.mu.Lock()
	delete(r.reservations, playerID)
	r.mu.Unlock()
}

func (r *Room) Claims(playerID string) bool {
	return r.Holds(playerID) || (!r.matchmaking && r.Reserved(playerID))
}

func (r *Room) Reserved(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	until, ok := r.reservations[playerID]
	return ok && time.Now().Before(until)
}

func (r *Room) Occupancy() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	n := len(r.seated)
	for id, until := range r.reservations {
		if !r.seated[id] && now.Before(until) {
			n++
		}
	}
	return n
}

func (r *Room) seat(playerID string) {
	r.mu.Lock()
	r.seated[playerID] = true
	delete(r.reservations, playerID)
	r.mu.Unlock()
}

func (r *Room) unseat(playerID string) {
	r.mu.Lock()
	delete(r.seated, playerID)
	r.mu.Unlock()
}

// Info reads only immutable fields, atomics and mutex-guarded seats, so the hub may call it concurrently with Run.
func (r *Room) Info() RoomInfo {
	return RoomInfo{
		ID:          r.id,
		Mode:        r.gameMode.String(),
		Deck:        int(r.deckType),
		MaxPlayers:  r.maxUsers,
		Players:     r.Occupancy(),
		Stake:       r.stake,
		Matchmaking: r.matchmaking,
		CreatedBy:   r.createdBy,
		CreatedAt:   r.createdAt,
	}
}

func (r *Room) Run() {
	log.Printf("[Room %s] цикл событий запущен", r.id)
	idle := time.NewTimer(emptyRoomTTL)
	defer func() {
		idle.Stop()
		r.cleanup()
		log.Printf("[Room %s] цикл событий остановлен", r.id)
	}()

	for {
		select {
		case <-r.stop:
			return

		case <-idle.C:
			if len(r.clients) == 0 && r.game == nil && r.Occupancy() == 0 {
				log.Printf("[Room %s] за столом никого нет, закрываем", r.id)
				return
			}
			idle.Reset(emptyRoomTTL)

		case client := <-r.register:
			r.handleRegister(client)
			r.playersCount.Store(int32(len(r.clients)))

		case client := <-r.unregister:
			r.handleUnregister(client)
			r.playersCount.Store(int32(len(r.clients)))

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
	r.seat(playerID)

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

	if r.game == nil {
		r.unseat(playerID)
	}

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
	ids := make([]string, 0, len(r.clients))
	for _, c := range r.clients {
		players = append(players, game.NewPlayer(c.ID(), c.Name()))
		ids = append(ids, c.ID())
	}

	newGame, err := game.NewGame(r.gameMode, r.deckType, players)
	if err != nil {
		log.Printf("[Room %s] ошибка создания игры: %v", r.id, err)
		return
	}

	ctx, cancel := bankContext()
	err = r.bank.LockStakes(ctx, r.id, r.gameMode.String(), int(r.deckType), r.stake, ids)
	cancel()
	if err != nil {
		r.handleLockFailure(err)
		return
	}

	r.game = newGame
	r.started.Store(true)
	log.Printf("[Room %s] партия началась, участников: %d", r.id, len(players))

	r.resetTurnTimer()
	r.broadcastGameState()
}

func (r *Room) handleLockFailure(err error) {
	var noFunds *model.InsufficientFundsError
	if errors.As(err, &noFunds) {
		playerID := noFunds.UserID.String()
		log.Printf("[Room %s] у игрока %s не хватает фишек, ждём другого", r.id, playerID)
		if c, ok := r.clients[playerID]; ok {
			r.sendError(c, errNoFunds)
			delete(r.clients, playerID)
			c.Close()
		}
		r.unseat(playerID)
		return
	}

	log.Printf("[Room %s] не удалось списать ставки: %v", r.id, err)
	for _, c := range r.clients {
		r.sendError(c, errStartFailed)
	}
	r.Close()
}

func (r *Room) settle() {
	if r.payouts != nil {
		return
	}

	losers := make(map[string]bool)
	for _, id := range r.game.Losers() {
		losers[string(id)] = true
	}
	places := make([]string, 0, len(r.game.Places()))
	for _, id := range r.game.Places() {
		places = append(places, string(id))
	}

	s, err := r.settleWithRetry(economy.Result{Places: places, Losers: losers})
	if err != nil {
		log.Printf("[Room %s] расчёт партии не удался после %d попыток: %v", r.id, settleAttempts, err)
		r.refundStakes()
		return
	}

	r.draw = s.Draw

	r.payouts = make(map[string]int64, len(s.Payouts))
	for _, p := range s.Payouts {
		r.payouts[p.UserID] = p.Net
	}
}

func (r *Room) settleWithRetry(result economy.Result) (economy.Settlement, error) {
	var err error
	for attempt := 1; attempt <= settleAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(r.settleRetryDelay)
		}
		ctx, cancel := bankContext()
		var s economy.Settlement
		s, err = r.bank.Settle(ctx, r.id, r.stake, result)
		cancel()
		if err == nil {
			return s, nil
		}
		log.Printf("[Room %s] ошибка расчёта партии (попытка %d из %d): %v", r.id, attempt, settleAttempts, err)
	}
	return economy.Settlement{}, err
}

func (r *Room) refundStakes() {
	r.payouts = map[string]int64{}
	r.draw = r.game.IsDraw()

	ctx, cancel := bankContext()
	refunded, err := r.bank.RefundGame(ctx, r.id)
	cancel()
	if err != nil {
		log.Printf("[Room %s] не удалось вернуть ставки, вернём при перезапуске: %v", r.id, err)
		return
	}
	if !refunded {
		log.Printf("[Room %s] партия уже закрыта в кассе, возврат не нужен", r.id)
		return
	}

	log.Printf("[Room %s] ставки партии возвращены игрокам", r.id)
	for _, p := range r.game.Players() {
		r.payouts[string(p.ID())] = 0
	}
	for _, c := range r.clients {
		r.sendError(c, errStakesRefunded)
	}
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
		err = errors.New("unknown action type: " + raw.Type)
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
		r.settle()
		r.finished.Store(true)
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

	if !r.autoMove() {
		log.Printf("[Room %s] таймаут хода: автоматический ход не удался, перезапускаем таймер", r.id)
	}
	r.onGameStateChanged()
}

func (r *Room) autoMove() bool {
	defender := r.game.Defender()
	attacker := r.game.Attacker()

	if r.game.Phase() == game.PhaseBattle && r.game.Table().HasUnbeaten() && defender != nil {
		log.Printf("[Room %s] таймаут хода: защитник %s автоматически берет карты", r.id, defender.ID())
		return r.game.Take(defender.ID()) == nil
	}

	if r.game.Phase() == game.PhaseBattle && r.game.Table().PairsCount() == 0 && attacker != nil {
		card, err := r.game.AutoAttack(attacker.ID())
		if err == nil {
			log.Printf("[Room %s] таймаут хода: авто-ход картой %s", r.id, card)
			return true
		}
	}

	if (r.game.Table().IsAllBeaten() && r.game.Phase() == game.PhaseBattle) || r.game.Phase() == game.PhaseGivingMore {
		passed := false
		for _, p := range r.game.Players() {
			if defender != nil && p.ID() == defender.ID() {
				continue
			}
			if p.IsActive() && r.game.Pass(p.ID()) == nil {
				passed = true
			}
		}
		return passed
	}
	return false
}

func (r *Room) resetTurnTimer() {
	r.stopTurnTimer()
	r.turnEndsAt = time.Now().Add(turnDuration)
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
		r.turnEndsAt = time.Time{}
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
		Mode:        r.game.Mode().String(),
		TurnLeftMs:  turnLeftMs(r.turnEndsAt),
		YouPassed:   r.game.HasPassed(game.PlayerID(client.ID())),
		Stake:       r.stake,
	}

	data, err := NewMessage(TypeGameState, payload)
	if err == nil {
		client.Send(data)
	}
}

func (r *Room) broadcastGameOver() {
	places := make([]string, 0, len(r.game.Places()))
	for _, id := range r.game.Places() {
		places = append(places, string(id))
	}
	payload := GameOverPayload{
		WinnerID: string(r.game.WinnerID()),
		LoserID:  string(r.game.LoserID()),
		Places:   places,
		Payouts:  r.payouts,
		Stake:    r.stake,
		IsDraw:   r.draw,
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

	text := strings.TrimSpace(chatMsg.Payload.Message)
	if text == "" {
		return
	}
	if runes := []rune(text); len(runes) > chatMaxRunes {
		text = string(runes[:chatMaxRunes])
	}

	now := time.Now()
	if now.Sub(r.lastChatAt[sender.ID()]) < chatCooldown {
		r.sendError(sender, "слишком часто, подождите секунду")
		return
	}
	r.lastChatAt[sender.ID()] = now

	chatMsg.Payload.Message = text
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

func turnLeftMs(endsAt time.Time) int64 {
	if endsAt.IsZero() {
		return 0
	}
	return max(time.Until(endsAt).Milliseconds(), 0)
}
