package postgres

import (
	"errors"
	"testing"
	"testing/fstest"

	"bito/migrations"
)

func TestLoadMigrations_OrderAndUpSection(t *testing.T) {
	files := fstest.MapFS{
		"000002_second.sql": {Data: []byte("-- +migrate Up\nCREATE TABLE b ();\n-- +migrate Down\nDROP TABLE b;\n")},
		"000001_first.sql":  {Data: []byte("-- note\n-- +migrate Up\nCREATE TABLE a ();\n")},
		"README.md":         {Data: []byte("not a migration")},
	}

	list, err := loadMigrations(files)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(list) != 2 || list[0].version != "000001_first" || list[1].version != "000002_second" {
		t.Fatalf("migrations = %+v, want first then second", list)
	}
	if list[0].up != "CREATE TABLE a ();" || list[1].up != "CREATE TABLE b ();" {
		t.Errorf("up sections = %q, %q; want only the Up statements", list[0].up, list[1].up)
	}
}

func TestLoadMigrations_RequiresUpSection(t *testing.T) {
	files := fstest.MapFS{"000001_broken.sql": {Data: []byte("CREATE TABLE a ();")}}
	if _, err := loadMigrations(files); !errors.Is(err, ErrNoUpSection) {
		t.Errorf("err = %v, want ErrNoUpSection", err)
	}
}

func TestEmbeddedMigrationsLoad(t *testing.T) {
	list, err := loadMigrations(migrations.FS)
	if err != nil {
		t.Fatalf("embedded migrations: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("no migrations embedded")
	}
}
