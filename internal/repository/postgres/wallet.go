package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"bito/internal/economy"
	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	gameStatusFinished = "finished"
	gameStatusDraw     = "draw"
)

type WalletRepository struct {
	pool *pgxpool.Pool
}

func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

const (
	creditSQL = `
		UPDATE user_wallets
		SET balance = balance + $2, updated_at = NOW()
		WHERE user_id = $1
		RETURNING balance
	`
	journalSQL = `
		INSERT INTO wallet_transactions (user_id, amount, kind, game_id, balance_after)
		VALUES ($1, $2, $3, $4, $5)
	`
)

func (r *WalletRepository) Balance(ctx context.Context, userID uuid.UUID) (int64, error) {
	const querySQL = `
		SELECT balance
		FROM user_wallets
		WHERE user_id = $1
	`

	var balance int64
	err := r.pool.QueryRow(ctx, querySQL, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, model.ErrUserNotFound
		}
		return 0, fmt.Errorf("wallet repo: get balance: %w", err)
	}

	return balance, nil
}

func (r *WalletRepository) LockStakes(ctx context.Context, g model.GameRecord, userIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("wallet repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertGameSQL = `
		INSERT INTO games (id, mode, deck, stake)
		VALUES ($1, $2, $3, $4)
	`
	if _, err = tx.Exec(ctx, insertGameSQL, g.ID, g.Mode, g.Deck, g.Stake); err != nil {
		return fmt.Errorf("wallet repo: insert game: %w", err)
	}

	ids := slices.Clone(userIDs)
	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	const lockSQL = `
		UPDATE user_wallets
		SET balance = balance - $2, updated_at = NOW()
		WHERE user_id = $1 AND balance >= $2
		RETURNING balance
	`
	const seatSQL = `
		INSERT INTO game_players (game_id, user_id)
		VALUES ($1, $2)
	`

	for _, id := range ids {
		var balanceAfter int64
		err = tx.QueryRow(ctx, lockSQL, id, g.Stake).Scan(&balanceAfter)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &model.InsufficientFundsError{UserID: id}
			}
			return fmt.Errorf("wallet repo: lock stake: %w", err)
		}

		if _, err = tx.Exec(ctx, journalSQL, id, -g.Stake, model.TxStakeLock, g.ID, balanceAfter); err != nil {
			return fmt.Errorf("wallet repo: journal stake lock: %w", err)
		}
		if _, err = tx.Exec(ctx, seatSQL, g.ID, id); err != nil {
			return fmt.Errorf("wallet repo: seat player: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("wallet repo: commit tx: %w", err)
	}
	return nil
}

func (r *WalletRepository) Settle(ctx context.Context, gameID uuid.UUID, s economy.Settlement) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("wallet repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	status, kind := gameStatusFinished, model.TxWinPayout
	if s.Draw {
		status, kind = gameStatusDraw, model.TxStakeRefund
	}

	const closeGameSQL = `
		UPDATE games
		SET status = $2, fee = $3, finished_at = NOW()
		WHERE id = $1 AND status = 'playing'
	`
	tag, err := tx.Exec(ctx, closeGameSQL, gameID, status, s.Fee)
	if err != nil {
		return fmt.Errorf("wallet repo: close game: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}

	payouts := slices.Clone(s.Payouts)
	slices.SortFunc(payouts, func(a, b economy.Payout) int {
		return strings.Compare(a.UserID, b.UserID)
	})

	const seatSQL = `
		UPDATE game_players
		SET place = $3, outcome = $4, payout = $5
		WHERE game_id = $1 AND user_id = $2
	`
	const statsSQL = `
		UPDATE user_stats
		SET wins = wins + $2, losses = losses + $3, total_games = total_games + 1, updated_at = NOW()
		WHERE user_id = $1
	`

	for _, p := range payouts {
		userID, err := uuid.Parse(p.UserID)
		if err != nil {
			return fmt.Errorf("wallet repo: parse user id %q: %w", p.UserID, err)
		}

		if p.Credit > 0 {
			var balanceAfter int64
			if err = tx.QueryRow(ctx, creditSQL, userID, p.Credit).Scan(&balanceAfter); err != nil {
				return fmt.Errorf("wallet repo: credit payout: %w", err)
			}
			if _, err = tx.Exec(ctx, journalSQL, userID, p.Credit, kind, gameID, balanceAfter); err != nil {
				return fmt.Errorf("wallet repo: journal payout: %w", err)
			}
		}

		outcome, wins, losses := "lose", 0, 1
		switch {
		case s.Draw:
			outcome, wins, losses = "draw", 0, 0
		case p.Win:
			outcome, wins, losses = "win", 1, 0
		}

		if _, err = tx.Exec(ctx, seatSQL, gameID, userID, p.Place, outcome, p.Net); err != nil {
			return fmt.Errorf("wallet repo: update game player: %w", err)
		}
		if _, err = tx.Exec(ctx, statsSQL, userID, wins, losses); err != nil {
			return fmt.Errorf("wallet repo: update stats: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("wallet repo: commit tx: %w", err)
	}
	return nil
}

func (r *WalletRepository) TopUp(ctx context.Context, userID uuid.UUID, now time.Time) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("wallet repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const selectSQL = `
		SELECT balance, last_top_up_at
		FROM user_wallets
		WHERE user_id = $1
		FOR UPDATE
	`
	var balance int64
	var lastTopUp *time.Time
	if err = tx.QueryRow(ctx, selectSQL, userID).Scan(&balance, &lastTopUp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, model.ErrUserNotFound
		}
		return 0, fmt.Errorf("wallet repo: read wallet: %w", err)
	}

	if balance >= economy.TopUpThreshold {
		return 0, model.ErrTopUpNotAllowed
	}
	if lastTopUp != nil {
		if next := lastTopUp.Add(economy.TopUpCooldown); now.Before(next) {
			return 0, &model.TopUpCooldownError{NextAt: next}
		}
	}

	const updateSQL = `
		UPDATE user_wallets
		SET balance = $2, last_top_up_at = $3, updated_at = $3
		WHERE user_id = $1
	`
	if _, err = tx.Exec(ctx, updateSQL, userID, economy.TopUpTo, now); err != nil {
		return 0, fmt.Errorf("wallet repo: top up: %w", err)
	}
	if _, err = tx.Exec(ctx, journalSQL, userID, economy.TopUpTo-balance, model.TxTopUp, nil, economy.TopUpTo); err != nil {
		return 0, fmt.Errorf("wallet repo: journal top up: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("wallet repo: commit tx: %w", err)
	}
	return economy.TopUpTo, nil
}

func (r *WalletRepository) Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]model.WalletTransaction, error) {
	const querySQL = `
		SELECT id, amount, kind, game_id, balance_after, created_at
		FROM wallet_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, querySQL, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: list transactions: %w", err)
	}
	defer rows.Close()

	list := make([]model.WalletTransaction, 0, limit)
	for rows.Next() {
		var t model.WalletTransaction
		if err = rows.Scan(&t.ID, &t.Amount, &t.Kind, &t.GameID, &t.BalanceAfter, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("wallet repo: scan transaction: %w", err)
		}
		list = append(list, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("wallet repo: read transactions: %w", err)
	}
	return list, nil
}

func (r *WalletRepository) RefundUnfinished(ctx context.Context) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("wallet repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const pendingSQL = `
		SELECT g.id, g.stake, gp.user_id
		FROM games g
		JOIN game_players gp ON gp.game_id = g.id
		WHERE g.status = 'playing'
		ORDER BY gp.user_id, g.id
		FOR UPDATE OF g
	`
	rows, err := tx.Query(ctx, pendingSQL)
	if err != nil {
		return 0, fmt.Errorf("wallet repo: select unfinished: %w", err)
	}

	type pending struct {
		gameID uuid.UUID
		userID uuid.UUID
		stake  int64
	}
	var list []pending
	for rows.Next() {
		var p pending
		if err = rows.Scan(&p.gameID, &p.stake, &p.userID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("wallet repo: scan unfinished: %w", err)
		}
		list = append(list, p)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("wallet repo: read unfinished: %w", err)
	}

	for _, p := range list {
		var balanceAfter int64
		if err = tx.QueryRow(ctx, creditSQL, p.userID, p.stake).Scan(&balanceAfter); err != nil {
			return 0, fmt.Errorf("wallet repo: refund credit: %w", err)
		}
		if _, err = tx.Exec(ctx, journalSQL, p.userID, p.stake, model.TxStakeRefund, p.gameID, balanceAfter); err != nil {
			return 0, fmt.Errorf("wallet repo: refund journal: %w", err)
		}
	}

	const closeSQL = `
		UPDATE games
		SET status = 'refunded', finished_at = NOW()
		WHERE status = 'playing'
	`
	tag, err := tx.Exec(ctx, closeSQL)
	if err != nil {
		return 0, fmt.Errorf("wallet repo: close unfinished: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("wallet repo: commit tx: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
