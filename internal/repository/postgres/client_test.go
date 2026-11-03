package postgres

import (
	"testing"

	"bito/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDSN_PasswordWithSpecialCharacters(t *testing.T) {
	cfg := &config.Config{DBUser: "bito", DBPassword: "p@ss/w#rd:1?", DBHost: "db", DBPort: "5432", DBName: "bito_db", DBSSLMode: "disable"}

	pc, err := pgxpool.ParseConfig(dsn(cfg))
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	conn := pc.ConnConfig
	if conn.Password != cfg.DBPassword || conn.User != "bito" || conn.Host != "db" || conn.Port != 5432 || conn.Database != "bito_db" {
		t.Errorf("parsed %s@%s:%d/%s pass %q", conn.User, conn.Host, conn.Port, conn.Database, conn.Password)
	}
}
