package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var migrationFS embed.FS

// Up applies all embedded PostgreSQL migrations.
func Up(ctx context.Context, db *sql.DB) error {
	migrations, err := fs.Glob(migrationFS, "sql/*.sql")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	if len(migrations) == 0 {
		return fmt.Errorf("no embedded migrations found")
	}

	migrationDir, err := fs.Sub(migrationFS, "sql")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	goose.SetBaseFS(migrationDir)
	goose.SetLogger(goose.NopLogger())

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
