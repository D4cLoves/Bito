package service

import (
	"context"

	"bito/internal/model"

	"github.com/google/uuid"
)

const maxHistoryLimit = 50

type GameRepository interface {
	History(ctx context.Context, userID uuid.UUID, limit int) ([]model.GameHistoryEntry, error)
}

type GameService struct {
	repo GameRepository
}

func NewGameService(repo GameRepository) *GameService {
	return &GameService{repo: repo}
}

func (s *GameService) History(ctx context.Context, userID uuid.UUID, limit int) ([]model.GameHistoryEntry, error) {
	if limit <= 0 || limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}
	return s.repo.History(ctx, userID, limit)
}
