//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"bito/internal/economy"
	"bito/internal/model"
	"bito/migrations"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBName = "bito_test"

func adminDSN() string {
	if dsn := os.Getenv("BITO_TEST_ADMIN_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://postgres:1221@localhost:5433/postgres?sslmode=disable"
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, adminDSN())
	if err != nil {
		t.Skipf("postgres is not reachable (%v); start docker compose to run integration tests", err)
	}
	defer admin.Close(ctx)

	if _, err = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+testDBName+" WITH (FORCE)"); err != nil {
		t.Fatalf("drop test db: %v", err)
	}
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+testDBName); err != nil {
		t.Fatalf("create test db: %v", err)
	}

	cfg, err := pgx.ParseConfig(adminDSN())
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.User, cfg.Password, cfg.Host, cfg.Port, testDBName)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		if a, err := pgx.Connect(context.Background(), adminDSN()); err == nil {
			_, _ = a.Exec(context.Background(), "DROP DATABASE IF EXISTS "+testDBName+" WITH (FORCE)")
			_ = a.Close(context.Background())
		}
	})

	if _, err = Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

func createUsers(t *testing.T, pool *pgxpool.Pool, n int) []uuid.UUID {
	t.Helper()
	repo := NewUserRepository(pool)
	ids := make([]uuid.UUID, 0, n)
	for i := range n {
		u := &model.User{
			Email:        fmt.Sprintf("player%d-%s@bito.test", i, uuid.NewString()[:8]),
			Name:         fmt.Sprintf("p%d_%s", i, uuid.NewString()[:8]),
			PasswordHash: "x",
		}
		if err := repo.CreateUser(context.Background(), u, &model.UserStats{}, &model.UserWallet{}); err != nil {
			t.Fatalf("create user: %v", err)
		}
		ids = append(ids, u.ID)
	}
	return ids
}

func assertJournalMatchesBalances(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const q = `
		SELECT COUNT(*) FROM (
			SELECT w.user_id
			FROM user_wallets w
			LEFT JOIN wallet_transactions t USING (user_id)
			GROUP BY w.user_id, w.balance
			HAVING w.balance <> COALESCE(SUM(t.amount), 0)
		) broken
	`
	var broken int
	if err := pool.QueryRow(context.Background(), q).Scan(&broken); err != nil {
		t.Fatalf("journal check: %v", err)
	}
	if broken != 0 {
		t.Fatalf("%d wallets do not match their journal", broken)
	}
}

func balanceOf(t *testing.T, repo *WalletRepository, id uuid.UUID) int64 {
	t.Helper()
	b, err := repo.Balance(context.Background(), id)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	return b
}

func TestWallet_FullGameLifecycle(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWalletRepository(pool)
	ctx := context.Background()
	users := createUsers(t, pool, 3)

	for _, id := range users {
		if b := balanceOf(t, repo, id); b != economy.StartBalance {
			t.Fatalf("start balance = %d, want %d", b, economy.StartBalance)
		}
	}
	assertJournalMatchesBalances(t, pool)

	gameID := uuid.New()
	if err := repo.LockStakes(ctx, model.GameRecord{ID: gameID, Mode: "podkidnoy", Deck: 36, Stake: 500}, users); err != nil {
		t.Fatalf("lock stakes: %v", err)
	}
	for _, id := range users {
		if b := balanceOf(t, repo, id); b != economy.StartBalance-500 {
			t.Fatalf("after lock = %d, want %d", b, economy.StartBalance-500)
		}
	}
	assertJournalMatchesBalances(t, pool)

	places := []string{users[0].String(), users[1].String(), users[2].String()}
	s := economy.Settle(500, economy.Result{Places: places, Losers: map[string]bool{users[2].String(): true}})
	if err := repo.Settle(ctx, gameID, s); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if err := repo.Settle(ctx, gameID, s); err != nil {
		t.Fatalf("second settle must be a no-op, got %v", err)
	}

	want := map[uuid.UUID]int64{
		users[0]: economy.StartBalance + 317,
		users[1]: economy.StartBalance + 158,
		users[2]: economy.StartBalance - 500,
	}
	for id, w := range want {
		if b := balanceOf(t, repo, id); b != w {
			t.Errorf("balance of %s = %d, want %d", id, b, w)
		}
	}
	assertJournalMatchesBalances(t, pool)

	var status string
	var fee int64
	if err := pool.QueryRow(ctx, `SELECT status, fee FROM games WHERE id = $1`, gameID).Scan(&status, &fee); err != nil {
		t.Fatalf("read game: %v", err)
	}
	if status != "finished" || fee != 25 {
		t.Errorf("game = %s / fee %d, want finished / 25", status, fee)
	}

	var wins, losses, total int
	if err := pool.QueryRow(ctx, `SELECT wins, losses, total_games FROM user_stats WHERE user_id = $1`, users[2]).Scan(&wins, &losses, &total); err != nil {
		t.Fatalf("read stats: %v", err)
	}
	if wins != 0 || losses != 1 || total != 1 {
		t.Errorf("loser stats = %d/%d/%d, want 0/1/1", wins, losses, total)
	}
}

func TestWallet_LockIsAllOrNothing(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWalletRepository(pool)
	ctx := context.Background()
	users := createUsers(t, pool, 2)

	if _, err := pool.Exec(ctx, `UPDATE user_wallets SET balance = 50 WHERE user_id = $1`, users[1]); err != nil {
		t.Fatalf("drain wallet: %v", err)
	}

	err := repo.LockStakes(ctx, model.GameRecord{ID: uuid.New(), Mode: "podkidnoy", Deck: 36, Stake: 100}, users)
	var noFunds *model.InsufficientFundsError
	if !errors.As(err, &noFunds) || noFunds.UserID != users[1] {
		t.Fatalf("err = %v, want insufficient funds for %s", err, users[1])
	}
	if b := balanceOf(t, repo, users[0]); b != economy.StartBalance {
		t.Errorf("rich player balance = %d, want untouched %d", b, economy.StartBalance)
	}
	var games int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM games`).Scan(&games)
	if games != 0 {
		t.Errorf("games = %d, want the failed game rolled back", games)
	}
}

func TestWallet_TopUpRules(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWalletRepository(pool)
	ctx := context.Background()
	user := createUsers(t, pool, 1)[0]
	now := time.Now()

	if _, err := repo.TopUp(ctx, user, now); !errors.Is(err, model.ErrTopUpNotAllowed) {
		t.Fatalf("rich top-up err = %v, want not allowed", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE user_wallets SET balance = 40 WHERE user_id = $1`, user); err != nil {
		t.Fatalf("drain wallet: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO wallet_transactions (user_id, amount, kind, balance_after) VALUES ($1, $2, 'stake_lock', 40)`, user, 40-economy.StartBalance); err != nil {
		t.Fatalf("journal drain: %v", err)
	}

	b, err := repo.TopUp(ctx, user, now)
	if err != nil || b != economy.TopUpTo {
		t.Fatalf("top-up = %d, %v; want %d", b, err, economy.TopUpTo)
	}
	assertJournalMatchesBalances(t, pool)

	if _, err = pool.Exec(ctx, `UPDATE user_wallets SET balance = 10 WHERE user_id = $1`, user); err != nil {
		t.Fatalf("drain wallet: %v", err)
	}
	var cooldown *model.TopUpCooldownError
	if _, err = repo.TopUp(ctx, user, now.Add(30*time.Minute)); !errors.As(err, &cooldown) {
		t.Fatalf("second top-up err = %v, want cooldown", err)
	}
	if _, err = repo.TopUp(ctx, user, now.Add(61*time.Minute)); err != nil {
		t.Fatalf("top-up after cooldown: %v", err)
	}
}

func TestWallet_RefundUnfinished(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewWalletRepository(pool)
	ctx := context.Background()
	users := createUsers(t, pool, 2)

	gameID := uuid.New()
	if err := repo.LockStakes(ctx, model.GameRecord{ID: gameID, Mode: "perevodnoy", Deck: 24, Stake: 250}, users); err != nil {
		t.Fatalf("lock stakes: %v", err)
	}

	n, err := repo.RefundUnfinished(ctx)
	if err != nil || n != 1 {
		t.Fatalf("refund = %d, %v; want 1 game", n, err)
	}
	for _, id := range users {
		if b := balanceOf(t, repo, id); b != economy.StartBalance {
			t.Errorf("balance after refund = %d, want %d", b, economy.StartBalance)
		}
	}
	assertJournalMatchesBalances(t, pool)

	if n, _ = repo.RefundUnfinished(ctx); n != 0 {
		t.Errorf("second refund touched %d games, want 0", n)
	}
	list, err := repo.Transactions(ctx, users[0], 10)
	if err != nil || len(list) != 3 || list[0].Kind != model.TxStakeRefund {
		t.Errorf("journal = %+v, %v; want refund on top of 3 records", list, err)
	}
}

func TestGames_History(t *testing.T) {
	pool := setupTestDB(t)
	wallet := NewWalletRepository(pool)
	games := NewGameRepository(pool)
	ctx := context.Background()
	users := createUsers(t, pool, 3)

	finished := uuid.New()
	if err := wallet.LockStakes(ctx, model.GameRecord{ID: finished, Mode: "perevodnoy", Deck: 36, Stake: 250}, users); err != nil {
		t.Fatalf("lock: %v", err)
	}
	places := []string{users[1].String(), users[0].String(), users[2].String()}
	s := economy.Settle(250, economy.Result{Places: places, Losers: map[string]bool{users[2].String(): true}})
	if err := wallet.Settle(ctx, finished, s); err != nil {
		t.Fatalf("settle: %v", err)
	}

	running := uuid.New()
	if err := wallet.LockStakes(ctx, model.GameRecord{ID: running, Mode: "podkidnoy", Deck: 24, Stake: 100}, users[:2]); err != nil {
		t.Fatalf("lock running: %v", err)
	}

	list, err := games.History(ctx, users[0], 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("history = %+v, want only the finished game", list)
	}
	g := list[0]
	if g.ID != finished || g.Mode != "perevodnoy" || g.Deck != 36 || g.Players != 3 || g.Stake != 250 || g.Place != 2 || g.Payout != 79 {
		t.Errorf("entry = %+v, want perevodnoy/36/3 players/250/2nd place/+79", g)
	}
	if g.FinishedAt.IsZero() || g.Outcome != "win" {
		t.Errorf("finishedAt = %v outcome = %q, want a time and win", g.FinishedAt, g.Outcome)
	}

	loser, _ := games.History(ctx, users[2], 10)
	if len(loser) != 1 || loser[0].Payout != -250 || loser[0].Place != 3 || loser[0].Outcome != "lose" {
		t.Errorf("loser entry = %+v, want 3rd place and -250", loser)
	}
}
