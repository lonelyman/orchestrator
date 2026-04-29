package migrations

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrationsPresent(t *testing.T) {
	migrations, err := fs.Glob(migrationFS, "sql/*.sql")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatalf("expected embedded migrations")
	}
}
