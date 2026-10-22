package http

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"bito/internal/model"

	"github.com/google/uuid"
)

const defaultHistoryLimit = 20

type GameService interface {
	History(ctx context.Context, userID uuid.UUID, limit int) ([]model.GameHistoryEntry, error)
}

type GameHandler struct {
	games GameService
}

func NewGameHandler(games GameService) *GameHandler {
	return &GameHandler{games: games}
}

func (gh *GameHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := defaultHistoryLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			respondError(w, http.StatusBadRequest, "limit must be a positive number")
			return
		}
		limit = n
	}

	list, err := gh.games.History(r.Context(), userID, limit)
	if err != nil {
		log.Printf("games: history of %s: %v", userID, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, list)
}
