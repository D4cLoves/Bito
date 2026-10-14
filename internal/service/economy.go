package service

import (
	"context"
	"fmt"
	"time"

	"bito/internal/economy"
	"bito/internal/model"

	"github.com/google/uuid"
)

const maxTransactionsLimit = 100

type WalletRepository interface {
	Balance(ctx context.Context, userID uuid.UUID) (int64, error)
	LockStakes(ctx context.Context, g model.GameRecord, userIDs []uuid.UUID) error
	Settle(ctx context.Context, gameID uuid.UUID, s economy.Settlement) error
	TopUp(ctx context.Context, userID uuid.UUID, now time.Time) (int64, error)
	Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.WalletTransaction, error)
	RefundUnfinished(ctx context.Context) (int, error)
}

type EconomyService struct {
	repo WalletRepository
	now  func() time.Time
}

func NewEconomyService(repo WalletRepository) *EconomyService {
	return &EconomyService{repo: repo, now: time.Now}
}

func (s *EconomyService) Balance(ctx context.Context, userID string) (int64, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("economy: parse user id: %w", err)
	}
	return s.repo.Balance(ctx, id)
}

func (s *EconomyService) LockStakes(ctx context.Context, gameID, mode string, deck int, stake int64, players []string) error {
	gid, err := uuid.Parse(gameID)
	if err != nil {
		return fmt.Errorf("economy: parse game id: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(players))
	for _, p := range players {
		id, err := uuid.Parse(p)
		if err != nil {
			return fmt.Errorf("economy: parse player id: %w", err)
		}
		ids = append(ids, id)
	}
	return s.repo.LockStakes(ctx, model.GameRecord{ID: gid, Mode: mode, Deck: deck, Stake: stake}, ids)
}

func (s *EconomyService) Settle(ctx context.Context, gameID string, stake int64, r economy.Result) (economy.Settlement, error) {
	gid, err := uuid.Parse(gameID)
	if err != nil {
		return economy.Settlement{}, fmt.Errorf("economy: parse game id: %w", err)
	}
	settlement := economy.Settle(stake, r)
	if err := s.repo.Settle(ctx, gid, settlement); err != nil {
		return economy.Settlement{}, err
	}
	return settlement, nil
}

func (s *EconomyService) TopUp(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.TopUp(ctx, userID, s.now())
}

func (s *EconomyService) Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.WalletTransaction, error) {
	if limit <= 0 || limit > maxTransactionsLimit {
		limit = maxTransactionsLimit
	}
	return s.repo.Transactions(ctx, userID, limit)
}

func (s *EconomyService) RefundUnfinished(ctx context.Context) (int, error) {
	return s.repo.RefundUnfinished(ctx)
}
