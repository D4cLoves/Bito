//go:build integration

package postgres

import (
	"context"
	"testing"

	"bito/migrations"
)

func TestMigrate_AppliesOnce(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	n, err := Migrate(ctx, pool, migrations.FS)
	if err != nil || n != 0 {
		t.Fatalf("second run applied %d, err %v; want nothing new", n, err)
	}

	list, _ := loadMigrations(migrations.FS)
	var recorded int
	if err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&recorded); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if recorded != len(list) {
		t.Errorf("schema_migrations has %d rows, want %d", recorded, len(list))
	}

	var wallets bool
	if err = pool.QueryRow(ctx, "SELECT to_regclass('public.wallet_transactions') IS NOT NULL").Scan(&wallets); err != nil || !wallets {
		t.Errorf("wallet_transactions missing after migrations (err %v)", err)
	}
}
