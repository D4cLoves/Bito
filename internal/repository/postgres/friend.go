package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FriendRepository struct {
	pool *pgxpool.Pool
}

func NewFriendRepository(pool *pgxpool.Pool) *FriendRepository {
	return &FriendRepository{
		pool: pool,
	}
}

func orderedPair(a, b uuid.UUID) (low, high uuid.UUID) {
	if bytes.Compare(a[:], b[:]) < 0 {
		return a, b
	}
	return b, a
}

func (r *FriendRepository) Request(ctx context.Context, from, to uuid.UUID) (model.FriendState, error) {
	low, high := orderedPair(from, to)

	const upsertSQL = `
	INSERT INTO friendships (user_low, user_high, requested_by)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_low, user_high) DO UPDATE
	SET status = 'accepted', accepted_at = NOW()
	WHERE friendships.status = 'pending'
	  AND friendships.requested_by <> EXCLUDED.requested_by
	RETURNING status
	`
	var state model.FriendState
	err := r.pool.QueryRow(ctx, upsertSQL, low, high, from).Scan(&state)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("friend repo: request: %w", err)
	}

	const currentSQL = `SELECT status FROM friendships WHERE user_low = $1 AND user_high = $2`
	if err = r.pool.QueryRow(ctx, currentSQL, low, high).Scan(&state); err != nil {
		return "", fmt.Errorf("friend repo: read pair: %w", err)
	}
	if state == model.FriendAccepted {
		return "", model.ErrAlreadyFriends
	}
	return "", model.ErrFriendRequestExists
}

func (r *FriendRepository) Accept(ctx context.Context, me, from uuid.UUID) error {
	low, high := orderedPair(me, from)
	const acceptSQL = `
		UPDATE friendships
		SET status = 'accepted', accepted_at = NOW()
		WHERE user_low = $1 AND user_high = $2 AND status = 'pending' AND requested_by = $3
	`
	tag, err := r.pool.Exec(ctx, acceptSQL, low, high, from)
	if err != nil {
		return fmt.Errorf("friend repo: accept: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrFriendRequestNotFound
	}
	return nil
}

func (r *FriendRepository) Remove(ctx context.Context, me, other uuid.UUID) error {
	low, high := orderedPair(me, other)
	const removeSQL = `DELETE FROM friendships WHERE user_low = $1 AND user_high = $2`
	if _, err := r.pool.Exec(ctx, removeSQL, low, high); err != nil {
		return fmt.Errorf("friend repo: remove: %w", err)
	}
	return nil
}

func (r *FriendRepository) List(ctx context.Context, me uuid.UUID) ([]model.FriendLink, error) {
	const listSQL = `
		SELECT u.id, u.name, f.status, f.requested_by = $1, COALESCE(f.accepted_at, f.created_at)
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.user_low = $1 THEN f.user_high ELSE f.user_low END
		WHERE (f.user_low = $1 OR f.user_high = $1) AND u.deleted_at IS NULL
		ORDER BY LOWER(u.name)
	`
	rows, err := r.pool.Query(ctx, listSQL, me)
	if err != nil {
		return nil, fmt.Errorf("friend repo: list: %w", err)
	}
	defer rows.Close()

	links := make([]model.FriendLink, 0)
	for rows.Next() {
		var l model.FriendLink
		if err = rows.Scan(&l.UserID, &l.Username, &l.State, &l.Outgoing, &l.Since); err != nil {
			return nil, fmt.Errorf("friend repo: scan: %w", err)
		}
		links = append(links, l)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("friend repo: read list: %w", err)
	}
	return links, nil
}
