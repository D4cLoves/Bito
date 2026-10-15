package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrateUpMarker   = "-- +migrate Up"
	migrateDownMarker = "-- +migrate Down"
	migrateLockID     = 7_310_202_601
)

var ErrNoUpSection = errors.New("migration has no '-- +migrate Up' section")

type migration struct {
	version string
	up      string
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, files fs.FS) (int, error) {
	list, err := loadMigrations(files)
	if err != nil {
		return 0, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrateLockID); err != nil {
		return 0, fmt.Errorf("migrate: lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrateLockID)
	}()

	const createSQL = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	if _, err = conn.Exec(ctx, createSQL); err != nil {
		return 0, fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return 0, fmt.Errorf("migrate: read applied: %w", err)
	}
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err = rows.Scan(&v); err != nil {
			rows.Close()
			return 0, fmt.Errorf("migrate: scan applied: %w", err)
		}
		applied[v] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("migrate: read applied: %w", err)
	}

	count := 0
	for _, m := range list {
		if applied[m.version] {
			continue
		}
		if err = applyMigration(ctx, conn, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, m migration) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate %s: begin tx: %w", m.version, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err = tx.Exec(ctx, m.up); err != nil {
		return fmt.Errorf("migrate %s: %w", m.version, err)
	}
	if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.version); err != nil {
		return fmt.Errorf("migrate %s: record version: %w", m.version, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate %s: commit: %w", m.version, err)
	}
	return nil
}

func loadMigrations(files fs.FS) ([]migration, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("migrate: list files: %w", err)
	}
	sort.Strings(names)

	list := make([]migration, 0, len(names))
	for _, name := range names {
		raw, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, fmt.Errorf("migrate: read %s: %w", name, err)
		}
		up, err := upSection(string(raw))
		if err != nil {
			return nil, fmt.Errorf("migrate %s: %w", name, err)
		}
		list = append(list, migration{version: strings.TrimSuffix(path.Base(name), ".sql"), up: up})
	}
	return list, nil
}

func upSection(script string) (string, error) {
	_, rest, found := strings.Cut(script, migrateUpMarker)
	if !found {
		return "", ErrNoUpSection
	}
	up, _, _ := strings.Cut(rest, migrateDownMarker)
	return strings.TrimSpace(up), nil
}
