package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"bito/internal/model"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PlayerService interface {
	Profile(ctx context.Context, me, id uuid.UUID) (model.PlayerProfile, error)
}

type PublicStatsResponse struct {
	Wins       int `json:"wins"`
	Losses     int `json:"losses"`
	TotalGames int `json:"totalGames"`
}

type PublicGameResponse struct {
	ID         string `json:"id"`
	FinishedAt string `json:"finishedAt"`
	Mode       string `json:"mode"`
	Deck       int    `json:"deck"`
	Players    int    `json:"players"`
	Stake      int64  `json:"stake"`
	Place      int    `json:"place"`
	Outcome    string `json:"outcome"`
}

type PlayerProfileResponse struct {
	ID        string               `json:"id"`
	Username  string               `json:"username"`
	CreatedAt string               `json:"createdAt"`
	Stats     PublicStatsResponse  `json:"stats"`
	Relation  string               `json:"relation"`
	Status    string               `json:"status,omitempty"`
	Games     []PublicGameResponse `json:"games"`
}

type PlayerHandler struct {
	players PlayerService
}

func NewPlayerHandler(players PlayerService) *PlayerHandler {
	return &PlayerHandler{players: players}
}

func (ph *PlayerHandler) Profile(w http.ResponseWriter, r *http.Request) {
	me, ok := UserIDFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный id игрока")
		return
	}

	p, err := ph.players.Profile(r.Context(), me, id)
	if errors.Is(err, model.ErrUserNotFound) {
		respondError(w, http.StatusNotFound, "игрок не найден")
		return
	}
	if err != nil {
		log.Printf("players: profile %s for %s: %v", id, me, err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileResponse(p))
}

func toPlayerProfileResponse(p model.PlayerProfile) PlayerProfileResponse {
	resp := PlayerProfileResponse{
		ID:        p.ID.String(),
		Username:  p.Username,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		Stats:     PublicStatsResponse{Wins: p.Stats.Wins, Losses: p.Stats.Losses, TotalGames: p.Stats.TotalGames},
		Relation:  string(p.Relation),
		Status:    string(p.Presence),
		Games:     make([]PublicGameResponse, 0, len(p.Games)),
	}
	for _, g := range p.Games {
		resp.Games = append(resp.Games, PublicGameResponse{
			ID:         g.ID.String(),
			FinishedAt: g.FinishedAt.Format(time.RFC3339),
			Mode:       g.Mode,
			Deck:       g.Deck,
			Players:    g.Players,
			Stake:      g.Stake,
			Place:      g.Place,
			Outcome:    publicOutcome(g),
		})
	}
	return resp
}

func publicOutcome(g model.GameHistoryEntry) string {
	if g.Outcome != "" {
		return g.Outcome
	}
	switch {
	case g.Payout > 0:
		return "win"
	case g.Payout < 0:
		return "lose"
	default:
		return "draw"
	}
}
