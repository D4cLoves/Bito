package postgres

import (
	"context"
	"fmt"

	"bito/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository struct {
	pool *pgxpool.Pool
}

func NewGameRepository(pool *pgxpool.Pool) *GameRepository {
	return &GameRepository{pool: pool}
}

func (r *GameRepository) History(ctx context.Context, userID uuid.UUID, limit int) ([]model.GameHistoryEntry, error) {
	const querySQL = `
		SELECT g.id, g.finished_at, g.mode, g.deck, g.stake, gp.place, gp.payout, gp.outcome,
		       (SELECT COUNT(*) FROM game_players x WHERE x.game_id = g.id) AS players
		FROM game_players gp
		JOIN games g ON g.id = gp.game_id
		WHERE gp.user_id = $1 AND g.status IN ('finished', 'draw')
		ORDER BY g.finished_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, querySQL, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("game repo: history: %w", err)
	}
	defer rows.Close()

	list := make([]model.GameHistoryEntry, 0, limit)
	for rows.Next() {
		var e model.GameHistoryEntry
		var place *int
		var outcome *string
		if err = rows.Scan(&e.ID, &e.FinishedAt, &e.Mode, &e.Deck, &e.Stake, &place, &e.Payout, &outcome, &e.Players); err != nil {
			return nil, fmt.Errorf("game repo: scan history: %w", err)
		}
		if place != nil {
			e.Place = *place
		}
		if outcome != nil {
			e.Outcome = *outcome
		}
		list = append(list, e)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("game repo: read history: %w", err)
	}
	return list, nil
}
