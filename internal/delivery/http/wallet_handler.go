package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"bito/internal/model"

	"github.com/google/uuid"
)

const defaultTransactionsLimit = 50

type WalletService interface {
	TopUp(ctx context.Context, userID uuid.UUID) (int64, error)
	Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.WalletTransaction, error)
}

type TopUpResponse struct {
	Balance int64 `json:"balance"`
}

type TopUpRefusal struct {
	Error  string     `json:"error"`
	NextAt *time.Time `json:"nextAt,omitempty"`
}

type WalletHandler struct {
	wallet WalletService
}

func NewWalletHandler(wallet WalletService) *WalletHandler {
	return &WalletHandler{wallet: wallet}
}

func (wh *WalletHandler) TopUp(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	balance, err := wh.wallet.TopUp(r.Context(), userID)
	if err != nil {
		var cooldown *model.TopUpCooldownError
		switch {
		case errors.As(err, &cooldown):
			respondJSON(w, http.StatusConflict, TopUpRefusal{Error: "Подачка уже была, следующая позже", NextAt: &cooldown.NextAt})
		case errors.Is(err, model.ErrTopUpNotAllowed):
			respondJSON(w, http.StatusConflict, TopUpRefusal{Error: "Подачка доступна, только если фишек меньше 100"})
		case errors.Is(err, model.ErrUserNotFound):
			respondError(w, http.StatusNotFound, "user not found")
		default:
			log.Printf("wallet: top up %s: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	respondJSON(w, http.StatusOK, TopUpResponse{Balance: balance})
}

func (wh *WalletHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := defaultTransactionsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			respondError(w, http.StatusBadRequest, "limit must be a positive number")
			return
		}
		limit = n
	}

	list, err := wh.wallet.Transactions(r.Context(), userID, limit)
	if err != nil {
		log.Printf("wallet: transactions of %s: %v", userID, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, list)
}
