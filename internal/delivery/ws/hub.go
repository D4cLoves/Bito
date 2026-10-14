package ws

import (
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"bito/internal/game"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type TokenValidator interface {
	ValidateAccessToken(token string) (*jwt.Claims, error)
}

type Hub struct {
	mu             sync.RWMutex
	rooms          map[string]*Room
	tokenValidator TokenValidator
	bank           Bank
	upgrader       websocket.Upgrader
}

func NewHub(tokenValidator TokenValidator, allowedOrigins []string, bank Bank) *Hub {
	if bank == nil {
		bank = freeBank{}
	}
	origins := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = true
	}

	return &Hub{
		rooms:          make(map[string]*Room),
		tokenValidator: tokenValidator,
		bank:           bank,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				// Non-browser clients (tests, CLI) send no Origin and cannot carry a victim's cookie.
				return origin == "" || origins[origin]
			},
		},
	}
}

func (h *Hub) CreateRoom(s RoomSettings) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.createRoomLocked(s)
}

func (h *Hub) createRoomLocked(s RoomSettings) *Room {
	roomID := uuid.NewString()

	room := NewRoom(roomID, s, h.bank)
	h.rooms[roomID] = room

	go func() {
		room.Run()
		h.RemoveRoom(roomID)
	}()

	return room
}

func (h *Hub) GetRoom(id string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.rooms[id]
	return room, ok
}

func (h *Hub) ListOpenRooms() []RoomInfo {
	h.mu.RLock()
	rooms := make([]RoomInfo, 0, len(h.rooms))
	for _, room := range h.rooms {
		if !room.Started() && !room.matchmaking {
			rooms = append(rooms, room.Info())
		}
	}
	h.mu.RUnlock()

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].CreatedAt.Before(rooms[j].CreatedAt)
	})
	return rooms
}

type MatchQuery struct {
	Mode  game.GameMode
	Deck  game.DeckType
	Stake int64
}

func (q MatchQuery) fits(room *Room) bool {
	return room.matchmaking && !room.Started() &&
		room.stake == q.Stake &&
		(q.Mode == 0 || room.gameMode == q.Mode) &&
		(q.Deck == 0 || room.deckType == q.Deck)
}

func (q MatchQuery) settings(createdBy string) RoomSettings {
	s := RoomSettings{
		Mode:        q.Mode,
		Deck:        q.Deck,
		MaxPlayers:  duelPlayers,
		Stake:       q.Stake,
		CreatedBy:   createdBy,
		Matchmaking: true,
	}
	if s.Mode == 0 {
		s.Mode = defaultDuelMode
	}
	if s.Deck == 0 {
		s.Deck = defaultDuelDeck
	}
	return s
}

func (h *Hub) MatchOrCreate(q MatchQuery, playerID string) (room *Room, created bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room = h.findMatchLocked(q, playerID); room == nil {
		room, created = h.createRoomLocked(q.settings(playerID)), true
	}
	room.Reserve(playerID)
	return room, created
}

func (h *Hub) findMatchLocked(q MatchQuery, playerID string) *Room {
	var best *Room
	for _, room := range h.rooms {
		if !q.fits(room) {
			continue
		}
		if room.Holds(playerID) || room.Reserved(playerID) {
			return room
		}
		if room.Occupancy() >= room.maxUsers {
			continue
		}
		if best == nil || room.createdAt.Before(best.createdAt) {
			best = room
		}
	}
	return best
}

func (h *Hub) ActiveRoomOf(playerID string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, room := range h.rooms {
		if room.Started() && room.Holds(playerID) {
			return room, true
		}
	}
	return nil, false
}

func (h *Hub) SeatedElsewhere(playerID, exceptRoomID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, room := range h.rooms {
		if id != exceptRoomID && room.Holds(playerID) {
			return true
		}
	}
	return false
}

func (h *Hub) RemoveRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	accessToken := tokenFromRequest(r)
	if accessToken == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := h.tokenValidator.ValidateAccessToken(accessToken)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	playerID := claims.UserID.String()
	playerName := claims.Username

	query := r.URL.Query()

	roomID := query.Get("room_id")
	if roomID == "" {
		http.Error(w, "missing room_id", http.StatusBadRequest)
		return
	}
	if playerName == "" {
		playerName = "Player_" + playerID
	}

	room, ok := h.GetRoom(roomID)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	if reason := h.admit(room, playerID); reason != "" {
		rejectConn(conn, reason)
		return
	}

	client := NewClient(playerID, playerName, conn, room)
	if !room.RegisterClient(client) {
		client.Close()
		_ = conn.Close()
		return
	}

	go client.WritePump()
	go client.ReadPump()
}

func (h *Hub) admit(room *Room, playerID string) string {
	if room.Holds(playerID) {
		return ""
	}
	if h.SeatedElsewhere(playerID, room.ID()) {
		return errOtherTable
	}
	if room.Stake() <= 0 {
		return ""
	}

	ctx, cancel := bankContext()
	defer cancel()
	balance, err := h.bank.Balance(ctx, playerID)
	if err != nil {
		log.Printf("[Hub] не удалось проверить баланс %s: %v", playerID, err)
		return errBankUnavailable
	}
	if balance < room.Stake() {
		return errNoFunds
	}
	return ""
}

func rejectConn(conn *websocket.Conn, reason string) {
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if data, err := NewMessage(TypeError, ErrorPayload{Message: reason}); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, ""))
	_ = conn.Close()
}

func tokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("accessToken"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return r.URL.Query().Get("token")
}
