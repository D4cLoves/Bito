package postgres

import (
	"context"
	"errors"
	"fmt"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

const pgErrUniqueViolation = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	user *model.User,
	stats *model.UserStats,
	wallet *model.UserWallet,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertUserSQL = `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertUserSQL, user.Email, user.Name, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("user repo: insert user: %w", err)
	}

	stats.UserID = user.ID
	wallet.UserID = user.ID

	const insertStatsSQL = `
		INSERT INTO user_stats (user_id)
		VALUES ($1)
		RETURNING elo, wins, losses, total_games, updated_at
	`
	err = tx.QueryRow(ctx, insertStatsSQL, user.ID).
		Scan(&stats.Elo, &stats.Wins, &stats.Losses, &stats.TotalGames, &stats.UpdatedAt)
	if err != nil {
		return fmt.Errorf("user repo: insert user_stats: %w", err)
	}

	const insertWalletSQL = `
		INSERT INTO user_wallets (user_id)
		VALUES ($1)
		RETURNING balance, updated_at
	`
	err = tx.QueryRow(ctx, insertWalletSQL, user.ID).
		Scan(&wallet.Balance, &wallet.UpdatedAt)
	if err != nil {
		return fmt.Errorf("user repo: insert user_wallets: %w", err)
	}

	user.Stats = stats
	user.Wallet = wallet

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("user repo: commit tx: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const querySQL = `
		SELECT
			u.id, u.email, u.name, u.password_hash, u.created_at, u.updated_at, u.deleted_at,
			s.elo, s.wins, s.losses, s.total_games, s.updated_at,
			w.balance, w.updated_at
		FROM users u
		LEFT JOIN user_stats s ON s.user_id = u.id
		LEFT JOIN user_wallets w ON w.user_id = u.id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	var user model.User
	var stats model.UserStats
	var wallet model.UserWallet

	err := r.pool.QueryRow(ctx, querySQL, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&stats.Elo, &stats.Wins, &stats.Losses, &stats.TotalGames, &stats.UpdatedAt,
		&wallet.Balance, &wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repo: get by id: %w", err)
	}

	stats.UserID = user.ID
	wallet.UserID = user.ID
	user.Stats = &stats
	user.Wallet = &wallet

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const querySQL = `
		SELECT
			u.id, u.email, u.name, u.password_hash, u.created_at, u.updated_at, u.deleted_at,
			s.elo, s.wins, s.losses, s.total_games, s.updated_at,
			w.balance, w.updated_at
		FROM users u
		LEFT JOIN user_stats s ON s.user_id = u.id
		LEFT JOIN user_wallets w ON w.user_id = u.id
		WHERE LOWER(u.email) = LOWER($1) AND u.deleted_at IS NULL
	`

	var user model.User
	var stats model.UserStats
	var wallet model.UserWallet

	err := r.pool.QueryRow(ctx, querySQL, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&stats.Elo, &stats.Wins, &stats.Losses, &stats.TotalGames, &stats.UpdatedAt,
		&wallet.Balance, &wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repo: get by email: %w", err)
	}

	stats.UserID = user.ID
	wallet.UserID = user.ID
	user.Stats = &stats
	user.Wallet = &wallet

	return &user, nil
}

func (r *UserRepository) GetByName(ctx context.Context, name string) (*model.User, error) {
	const querySQL = `
		SELECT
			u.id, u.email, u.name, u.password_hash, u.created_at, u.updated_at, u.deleted_at,
			s.elo, s.wins, s.losses, s.total_games, s.updated_at,
			w.balance, w.updated_at
		FROM users u
		LEFT JOIN user_stats s ON s.user_id = u.id
		LEFT JOIN user_wallets w ON w.user_id = u.id
		WHERE LOWER(u.name) = LOWER($1) AND u.deleted_at IS NULL
	`

	var user model.User
	var stats model.UserStats
	var wallet model.UserWallet

	err := r.pool.QueryRow(ctx, querySQL, name).Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&stats.Elo, &stats.Wins, &stats.Losses, &stats.TotalGames, &stats.UpdatedAt,
		&wallet.Balance, &wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repo: get by name: %w", err)
	}

	stats.UserID = user.ID
	wallet.UserID = user.ID
	user.Stats = &stats
	user.Wallet = &wallet

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	const querySQL = `
		UPDATE users
		SET name = $1, password_hash = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, querySQL, user.Name, user.PasswordHash, user.ID).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("user repo: update user: %w", err)
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const querySQL = `
		UPDATE users
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	cmdTag, err := r.pool.Exec(ctx, querySQL, id)
	if err != nil {
		return fmt.Errorf("user repo: delete user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
