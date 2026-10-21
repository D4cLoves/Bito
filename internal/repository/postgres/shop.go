package postgres

import (
	"context"
	"errors"
	"fmt"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShopRepository struct {
	pool *pgxpool.Pool
}

func NewShopRepository(pool *pgxpool.Pool) *ShopRepository {
	return &ShopRepository{pool: pool}
}

func (r *ShopRepository) Buy(ctx context.Context, userID uuid.UUID, item string, price int64) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("shop repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const ownSQL = `
		INSERT INTO user_items (user_id, item, price)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item) DO NOTHING
	`
	tag, err := tx.Exec(ctx, ownSQL, userID, item, price)
	if err != nil {
		return 0, fmt.Errorf("shop repo: grant item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, model.ErrAlreadyOwned
	}

	const debitSQL = `
		UPDATE user_wallets
		SET balance = balance - $2, updated_at = NOW()
		WHERE user_id = $1 AND balance >= $2
		RETURNING balance
	`
	var balance int64
	err = tx.QueryRow(ctx, debitSQL, userID, price).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, &model.InsufficientFundsError{UserID: userID}
	}
	if err != nil {
		return 0, fmt.Errorf("shop repo: debit: %w", err)
	}

	if _, err = tx.Exec(ctx, journalSQL, userID, -price, model.TxPurchase, nil, balance); err != nil {
		return 0, fmt.Errorf("shop repo: journal purchase: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("shop repo: commit tx: %w", err)
	}
	return balance, nil
}

func (r *ShopRepository) Owned(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT item FROM user_items WHERE user_id = $1 ORDER BY bought_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("shop repo: owned: %w", err)
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var item string
		if err = rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("shop repo: scan owned: %w", err)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("shop repo: read owned: %w", err)
	}
	return items, nil
}

func (r *ShopRepository) Look(ctx context.Context, userID uuid.UUID) (model.Look, bool, error) {
	var look model.Look
	err := r.pool.QueryRow(ctx, `SELECT theme, felt, back FROM user_looks WHERE user_id = $1`, userID).
		Scan(&look.Theme, &look.Felt, &look.Back)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Look{}, false, nil
	}
	if err != nil {
		return model.Look{}, false, fmt.Errorf("shop repo: look: %w", err)
	}
	return look, true, nil
}

func (r *ShopRepository) SaveLook(ctx context.Context, userID uuid.UUID, look model.Look) error {
	const saveSQL = `
		INSERT INTO user_looks (user_id, theme, felt, back)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET theme = EXCLUDED.theme, felt = EXCLUDED.felt, back = EXCLUDED.back, updated_at = NOW()
	`
	if _, err := r.pool.Exec(ctx, saveSQL, userID, look.Theme, look.Felt, look.Back); err != nil {
		return fmt.Errorf("shop repo: save look: %w", err)
	}
	return nil
}
