package http

import (
	"context"
	"errors"
	"log"
	"net/http"

	"bito/internal/delivery/ws"
	"bito/internal/economy"
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
	ErrInvalidStake      = errors.New("stake must be one of 100, 250, 500, 1000, 2500, 5000")
)

const (
	msgNoFunds    = "Не хватает фишек на эту ставку"
	msgOtherTable = "Вы уже играете за другим столом"
	msgBankDown   = "Касса недоступна, попробуйте позже"
)

type CreateTableRequest struct {
	Mode       string `json:"mode"`
	Deck       int    `json:"deck"`
	MaxPlayers int    `json:"maxPlayers"`
	Stake      int64  `json:"stake"`
}

func (req CreateTableRequest) validate() (ws.RoomSettings, error) {
	var s ws.RoomSettings
	switch req.Mode {
	case "podkidnoy":
		s.Mode = game.ModePodkidnoy
	case "perevodnoy":
		s.Mode = game.ModePerevodnoy
	default:
		return s, ErrInvalidMode
	}

	s.Deck = game.DeckType(req.Deck)
	switch s.Deck {
	case game.Deck24, game.Deck36, game.Deck52:
	default:
		return s, ErrInvalidDeck
	}

	if req.MaxPlayers < minPlayers || req.MaxPlayers > maxPlayers {
		return s, ErrInvalidMaxPlayers
	}
	if req.MaxPlayers*cardsPerHand > req.Deck {
		return s, ErrNotEnoughCards
	}
	s.MaxPlayers = req.MaxPlayers

	s.Stake = req.Stake
	if s.Stake == 0 {
		s.Stake = economy.MinStake
	}
	if !economy.IsAllowedStake(s.Stake) {
		return s, ErrInvalidStake
	}

	return s, nil
}

type BalanceReader interface {
	Balance(ctx context.Context, userID string) (int64, error)
}

type TableHandler struct {
	hub    *ws.Hub
	wallet BalanceReader
}

func NewTableHandler(hub *ws.Hub, wallet BalanceReader) *TableHandler {
	return &TableHandler{hub: hub, wallet: wallet}
}

func (th *TableHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, settings, ok := th.prepare(w, r)
	if !ok {
		return
	}

	room := th.hub.CreateRoom(settings)
	respondJSON(w, http.StatusCreated, room.Info())
}

func (th *TableHandler) Quick(w http.ResponseWriter, r *http.Request) {
	userID, settings, ok := th.prepare(w, r)
	if !ok {
		return
	}

	if room, found := th.hub.FindOpenRoom(settings, userID); found {
		respondJSON(w, http.StatusOK, room.Info())
		return
	}

	room := th.hub.CreateRoom(settings)
	respondJSON(w, http.StatusCreated, room.Info())
}

func (th *TableHandler) Active(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	room, found := th.hub.ActiveRoomOf(userID.String())
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, room.Info())
}

func (th *TableHandler) List(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, th.hub.ListOpenRooms())
}

func (th *TableHandler) prepare(w http.ResponseWriter, r *http.Request) (string, ws.RoomSettings, bool) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return "", ws.RoomSettings{}, false
	}
	userID := uid.String()

	var req CreateTableRequest
	if !decodeJSON(w, r, &req) {
		return "", ws.RoomSettings{}, false
	}

	settings, err := req.validate()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return "", ws.RoomSettings{}, false
	}
	settings.CreatedBy = userID

	if th.hub.SeatedElsewhere(userID, "") {
		respondError(w, http.StatusConflict, msgOtherTable)
		return "", ws.RoomSettings{}, false
	}

	if th.wallet != nil {
		balance, err := th.wallet.Balance(r.Context(), userID)
		if err != nil {
			log.Printf("tables: read balance of %s: %v", userID, err)
			respondError(w, http.StatusServiceUnavailable, msgBankDown)
			return "", ws.RoomSettings{}, false
		}
		if balance < settings.Stake {
			respondError(w, http.StatusPaymentRequired, msgNoFunds)
			return "", ws.RoomSettings{}, false
		}
	}

	return userID, settings, true
}
