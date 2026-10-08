package ws

import (
	"log"
	"net/http"
	"sort"
	"sync"

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
	upgrader       websocket.Upgrader
}

func NewHub(tokenValidator TokenValidator, allowedOrigins []string) *Hub {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = true
	}

	return &Hub{
		rooms:          make(map[string]*Room),
		tokenValidator: tokenValidator,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				// Non-browser clients (tests, CLI) send no Origin and cannot carry a victim's cookie.
				return origin == "" || origins[origin]
			},
		},
	}
}

func (h *Hub) CreateRoom(mode game.GameMode, deckType game.DeckType, maxUsers int, createdBy string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomID := uuid.NewString()

	room := NewRoom(roomID, mode, deckType, maxUsers, createdBy)
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
		if !room.Started() {
			rooms = append(rooms, room.Info())
		}
	}
	h.mu.RUnlock()

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].CreatedAt.Before(rooms[j].CreatedAt)
	})
	return rooms
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

	client := NewClient(playerID, playerName, conn, room)
	if !room.RegisterClient(client) {
		client.Close()
		_ = conn.Close()
		return
	}

	go client.WritePump()
	go client.ReadPump()
}

func tokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("accessToken"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return r.URL.Query().Get("token")
}
