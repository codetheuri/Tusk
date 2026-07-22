package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

// RunMigrations executes all embedded SQL migrations against the database.
func RunMigrations(db *sql.DB, dialect string) error {
	goose.SetBaseFS(EmbedMigrations)

	// Ensure dialect is compatible with goose (e.g., pgsql -> postgres)
	if dialect == "pgsql" {
		dialect = "postgres"
	}

	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
