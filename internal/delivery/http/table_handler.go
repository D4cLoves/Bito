package http

import (
	"context"
	"errors"
	"log"
	"net/http"

	"bito/internal/delivery/ws"
	"bito/internal/economy"
	"bito/internal/game"

	"github.com/go-chi/chi/v5"
)

const (
	minCompanySize = 3
	maxCompanySize = 6
	cardsPerHand   = 6
	anyMode        = "any"
)

var (
	ErrInvalidMode       = errors.New("mode must be podkidnoy or perevodnoy")
	ErrInvalidQuickMode  = errors.New("mode must be podkidnoy, perevodnoy or any")
	ErrInvalidDeck       = errors.New("deck must be 24, 36 or 52")
	ErrInvalidQuickDeck  = errors.New("deck must be 24, 36, 52 or 0 for any")
	ErrInvalidMaxPlayers = errors.New("maxPlayers must be between 3 and 6")
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

type QuickMatchRequest struct {
	Mode  string `json:"mode"`
	Deck  int    `json:"deck"`
	Stake int64  `json:"stake"`
}

func parseMode(raw string) (game.GameMode, bool) {
	switch raw {
	case "podkidnoy":
		return game.ModePodkidnoy, true
	case "perevodnoy":
		return game.ModePerevodnoy, true
	}
	return 0, false
}

func parseDeck(raw int) (game.DeckType, bool) {
	switch d := game.DeckType(raw); d {
	case game.Deck24, game.Deck36, game.Deck52:
		return d, true
	}
	return 0, false
}

func parseStake(raw int64) (int64, error) {
	if raw == 0 {
		return economy.MinStake, nil
	}
	if !economy.IsAllowedStake(raw) {
		return 0, ErrInvalidStake
	}
	return raw, nil
}

func (req CreateTableRequest) validate() (ws.RoomSettings, error) {
	var s ws.RoomSettings
	var ok bool

	if s.Mode, ok = parseMode(req.Mode); !ok {
		return s, ErrInvalidMode
	}
	if s.Deck, ok = parseDeck(req.Deck); !ok {
		return s, ErrInvalidDeck
	}
	if req.MaxPlayers < minCompanySize || req.MaxPlayers > maxCompanySize {
		return s, ErrInvalidMaxPlayers
	}
	if req.MaxPlayers*cardsPerHand > req.Deck {
		return s, ErrNotEnoughCards
	}
	s.MaxPlayers = req.MaxPlayers

	stake, err := parseStake(req.Stake)
	if err != nil {
		return s, err
	}
	s.Stake = stake
	return s, nil
}

func (req QuickMatchRequest) validate() (ws.MatchQuery, error) {
	var q ws.MatchQuery
	var ok bool

	if req.Mode != "" && req.Mode != anyMode {
		if q.Mode, ok = parseMode(req.Mode); !ok {
			return q, ErrInvalidQuickMode
		}
	}
	if req.Deck != 0 {
		if q.Deck, ok = parseDeck(req.Deck); !ok {
			return q, ErrInvalidQuickDeck
		}
	}

	stake, err := parseStake(req.Stake)
	if err != nil {
		return q, err
	}
	q.Stake = stake
	return q, nil
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
	userID, ok := th.userID(w, r)
	if !ok {
		return
	}

	var req CreateTableRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	settings, err := req.validate()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !th.admit(w, r, userID, settings.Stake) {
		return
	}

	settings.CreatedBy = userID
	room := th.hub.CreateRoom(settings)
	respondJSON(w, http.StatusCreated, room.Info())
}

func (th *TableHandler) Quick(w http.ResponseWriter, r *http.Request) {
	userID, ok := th.userID(w, r)
	if !ok {
		return
	}

	var req QuickMatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	query, err := req.validate()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !th.admit(w, r, userID, query.Stake) {
		return
	}

	room, created := th.hub.MatchOrCreate(query, userID)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, room.Info())
}

func (th *TableHandler) Get(w http.ResponseWriter, r *http.Request) {
	room, found := th.hub.GetRoom(chi.URLParam(r, "id"))
	if !found {
		respondError(w, http.StatusNotFound, "table not found")
		return
	}
	respondJSON(w, http.StatusOK, room.Info())
}

func (th *TableHandler) Active(w http.ResponseWriter, r *http.Request) {
	userID, ok := th.userID(w, r)
	if !ok {
		return
	}

	room, found := th.hub.ActiveRoomOf(userID)
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, room.Info())
}

func (th *TableHandler) List(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, th.hub.ListOpenRooms())
}

func (th *TableHandler) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return uid.String(), true
}

func (th *TableHandler) admit(w http.ResponseWriter, r *http.Request, userID string, stake int64) bool {
	if th.hub.SeatedElsewhere(userID, "") {
		respondError(w, http.StatusConflict, msgOtherTable)
		return false
	}
	if th.wallet == nil {
		return true
	}

	balance, err := th.wallet.Balance(r.Context(), userID)
	if err != nil {
		log.Printf("tables: read balance of %s: %v", userID, err)
		respondError(w, http.StatusServiceUnavailable, msgBankDown)
		return false
	}
	if balance < stake {
		respondError(w, http.StatusPaymentRequired, msgNoFunds)
		return false
	}
	return true
}
