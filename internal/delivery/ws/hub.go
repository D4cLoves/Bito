package ws

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"bito/internal/game"
	"bito/internal/pkg/jwt"

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

// NewHub authenticates players by access token. allowedOrigins guards against
// Cross-Site WebSocket Hijacking: the token may come from a cookie, so without
// an origin check a page on any other site could connect with the victim's session.
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

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	room := h.GetOrCreateRoom(roomID, mode, deckType, maxUsers)

	client := NewClient(playerID, playerName, conn, room)
	if !room.RegisterClient(client) {
		client.Close()
		_ = conn.Close()
		return
	}

	go client.WritePump()
	go client.ReadPump()
}

// tokenFromRequest reads the access token from the cookie set at login, falling back
// to the "token" query parameter: browsers cannot set headers on a WebSocket handshake.
func tokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("accessToken"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return r.URL.Query().Get("token")
}
