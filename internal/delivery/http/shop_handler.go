package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"bito/internal/economy"
	"bito/internal/model"
	"bito/internal/service"

	"github.com/google/uuid"
)

type ShopService interface {
	State(ctx context.Context, me uuid.UUID) (service.ShopState, error)
	Buy(ctx context.Context, me uuid.UUID, key string) (int64, error)
	Wear(ctx context.Context, me uuid.UUID, look model.Look) error
	SetAvatar(ctx context.Context, me uuid.UUID, id string) error
	Avatars(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

type ShopItemResponse struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Price int64  `json:"price"`
}

type LookPayload struct {
	Theme string `json:"theme"`
	Felt  string `json:"felt"`
	Back  string `json:"back"`
}

type ShopResponse struct {
	Items     []ShopItemResponse `json:"items"`
	Owned     []string           `json:"owned"`
	Look      LookPayload        `json:"look"`
	LookSaved bool               `json:"lookSaved"`
	Avatar    string             `json:"avatar"`
}

type BuyRequest struct {
	Item string `json:"item"`
}

type BuyResponse struct {
	Balance int64    `json:"balance"`
	Owned   []string `json:"owned"`
}

type ShopHandler struct {
	shop ShopService
}

func NewShopHandler(shop ShopService) *ShopHandler {
	return &ShopHandler{shop: shop}
}

func (sh *ShopHandler) State(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	state, err := sh.shop.State(r.Context(), me)
	if err != nil {
		respondShopError(w, me, err)
		return
	}

	items := make([]ShopItemResponse, 0, len(economy.Catalog))
	for _, it := range economy.Catalog {
		items = append(items, ShopItemResponse{Kind: it.Kind, ID: it.ID, Price: it.Price})
	}
	respondJSON(w, http.StatusOK, ShopResponse{
		Items:     items,
		Owned:     state.Owned,
		Look:      LookPayload{Theme: state.Look.Theme, Felt: state.Look.Felt, Back: state.Look.Back},
		LookSaved: state.LookSaved,
		Avatar:    state.Avatar,
	})
}

func (sh *ShopHandler) Buy(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req BuyRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	balance, err := sh.shop.Buy(r.Context(), me, req.Item)
	if err != nil {
		respondShopError(w, me, err)
		return
	}
	state, err := sh.shop.State(r.Context(), me)
	if err != nil {
		respondShopError(w, me, err)
		return
	}
	respondJSON(w, http.StatusOK, BuyResponse{Balance: balance, Owned: state.Owned})
}

func (sh *ShopHandler) Wear(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req LookPayload
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := sh.shop.Wear(r.Context(), me, model.Look{Theme: req.Theme, Felt: req.Felt, Back: req.Back}); err != nil {
		respondShopError(w, me, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type AvatarRequest struct {
	Avatar string `json:"avatar"`
}

func (sh *ShopHandler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req AvatarRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := sh.shop.SetAvatar(r.Context(), me, req.Avatar); err != nil {
		respondShopError(w, me, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *ShopHandler) Avatars(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var ids []uuid.UUID
	for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		respondJSON(w, http.StatusOK, map[string]string{})
		return
	}

	avatars, err := sh.shop.Avatars(r.Context(), ids)
	if err != nil {
		respondShopError(w, me, err)
		return
	}
	out := make(map[string]string, len(avatars))
	for id, avatar := range avatars {
		out[id.String()] = avatar
	}
	respondJSON(w, http.StatusOK, out)
}

func respondShopError(w http.ResponseWriter, me uuid.UUID, err error) {
	switch {
	case errors.Is(err, model.ErrUnknownItem):
		respondError(w, http.StatusNotFound, "такого предмета нет")
	case errors.Is(err, model.ErrAlreadyOwned):
		respondError(w, http.StatusConflict, "уже куплено")
	case errors.Is(err, model.ErrInsufficientFunds):
		respondError(w, http.StatusPaymentRequired, "не хватает фишек на покупку")
	case errors.Is(err, model.ErrNotOwned):
		respondError(w, http.StatusForbidden, "сначала купите этот предмет")
	default:
		log.Printf("shop: %s: %v", me, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
