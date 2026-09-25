package ws

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"bito/internal/game"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true 
	},
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetOrCreateRoom(id string, mode game.GameMode, deckType game.DeckType, maxUsers int) *Room {
	h.mu.RLock()
	if room, exists := h.rooms[id]; exists {
		h.mu.RUnlock()
		return room
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[id]; exists {
		return room
	}

	room := NewRoom(id, mode, deckType, maxUsers)
	h.rooms[id] = room

	go func() {
		room.Run()
		h.RemoveRoom(id)
	}()

	return room
}

func (h *Hub) RemoveRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	roomID := query.Get("room_id")
	playerID := query.Get("player_id")
	playerName := query.Get("player_name")

	if roomID == "" || playerID == "" {
		http.Error(w, "missing room_id or player_id", http.StatusBadRequest)
		return
	}
	if playerName == "" {
		playerName = "Player_" + playerID
	}

	modeStr := query.Get("mode")
	mode := game.ModePodkidnoy
	if modeStr == "perevodnoy" {
		mode = game.ModePerevodnoy
	}

	deckType := game.Deck36
	if deckVal, err := strconv.Atoi(query.Get("deck")); err == nil {
		switch deckVal {
		case 24:
			deckType = game.Deck24
		case 36:
			deckType = game.Deck36
		case 52:
			deckType = game.Deck52
		default:
			deckType = game.Deck36
		}
	}

	maxUsers := 2
	if usersVal, err := strconv.Atoi(query.Get("max_users")); err == nil && usersVal >= 2 && usersVal <= 6 {
		maxUsers = usersVal
	}

	room := h.GetOrCreateRoom(roomID, mode, deckType, maxUsers)

	conn, err := upgrader.Upgrade(w, r, nil)
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
