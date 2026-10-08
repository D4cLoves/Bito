package http

import (
	"errors"
	"net/http"

	"bito/internal/delivery/ws"
	"bito/internal/game"
)

const (
	minPlayers   = 2
	maxPlayers   = 6
	cardsPerHand = 6
)

var (
	ErrInvalidMode       = errors.New("mode must be podkidnoy or perevodnoy")
	ErrInvalidDeck       = errors.New("deck must be 24, 36 or 52")
	ErrInvalidMaxPlayers = errors.New("maxPlayers must be between 2 and 6")
	ErrNotEnoughCards    = errors.New("not enough cards for this many players")
)

type CreateTableRequest struct {
	Mode       string `json:"mode"`
	Deck       int    `json:"deck"`
	MaxPlayers int    `json:"maxPlayers"`
}

func (req CreateTableRequest) validate() (game.GameMode, game.DeckType, error) {
	var mode game.GameMode
	switch req.Mode {
	case "podkidnoy":
		mode = game.ModePodkidnoy
	case "perevodnoy":
		mode = game.ModePerevodnoy
	default:
		return 0, 0, ErrInvalidMode
	}

	deck := game.DeckType(req.Deck)
	switch deck {
	case game.Deck24, game.Deck36, game.Deck52:
	default:
		return 0, 0, ErrInvalidDeck
	}

	if req.MaxPlayers < minPlayers || req.MaxPlayers > maxPlayers {
		return 0, 0, ErrInvalidMaxPlayers
	}

	if req.MaxPlayers*cardsPerHand > req.Deck {
		return 0, 0, ErrNotEnoughCards
	}

	return mode, deck, nil
}

type TableHandler struct {
	hub *ws.Hub
}

func NewTableHandler(hub *ws.Hub) *TableHandler {
	return &TableHandler{hub: hub}
}

func (th *TableHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTableRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	mode, deck, err := req.validate()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	room := th.hub.CreateRoom(mode, deck, req.MaxPlayers, userID.String())
	respondJSON(w, http.StatusCreated, room.Info())
}

func (th *TableHandler) List(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, th.hub.ListOpenRooms())
}
