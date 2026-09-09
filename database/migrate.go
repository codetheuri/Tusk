// Package database owns schema migrations and seed data.
package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"
)

// EmbedMigrations carries the SQL files into the binary so a deployed artefact
// can migrate itself with no accompanying files on disk.
//
// The pattern reaches into the dialect subdirectory; `migrations/*.sql` would
// silently match nothing and produce a no-op migration run.
//
//go:embed migrations/postgres/*.sql
var EmbedMigrations embed.FS

// migrationDir maps a driver name to its migration directory.
//
// PostgreSQL is the only supported dialect. The MySQL migration was removed when
// primary keys became UUIDs: MySQL has no native UUID type, so the schema and the
// Go models could no longer describe the same thing. Rather than ship a migration
// that produces tables the application cannot read, the path is gone. The SQL
// remains in git history for anyone who needs it.
func migrationDir(dialect string) (string, string, error) {
	switch dialect {
	case "postgres", "pgsql":
		return "postgres", "migrations/postgres", nil
	default:
		return "", "", fmt.Errorf(
			"no migrations available for driver %q: Tusk targets PostgreSQL only "+
				"(set DB_DRIVER=postgres)", dialect)
	}
}

// Migrator applies one set of migrations.
//
// It exists so a service built on Tusk can run its own schema through the same
// machinery. Tusk's built-in migrations create Tusk's identity tables, which an
// application with different ones — users who log in by phone and have no email,
// say — does not want. Such a service supplies its own embedded filesystem here.
//
// Migrations are deliberately not shareable as library code. A migration is a
// record of a change already applied to a specific database; if it could change
// underneath you when a dependency updated, it would no longer be a record of
// anything. Each service owns its own.
type Migrator struct {
	fsys fs.FS
	dir  string
}

// NewMigrator returns a Migrator over a caller-supplied migration set.
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	m := database.NewMigrator(migrations, "migrations")
//	err := m.Up(db)
func NewMigrator(fsys fs.FS, dir string) *Migrator {
	return &Migrator{fsys: fsys, dir: dir}
}

// gooseMu guards goose's package-level configuration.
//
// SetBaseFS and SetDialect are global, so two Migrators running at once would
// otherwise apply one's migrations from the other's filesystem. Rare in a server,
// entirely plausible in a test binary.
var gooseMu sync.Mutex

// prepare points goose at this Migrator's files. Callers hold gooseMu.
func (m *Migrator) prepare() error {
	goose.SetBaseFS(m.fsys)

	// Always postgres: Tusk targets it exclusively, and config rejects anything
	// else long before this runs.
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	return nil
}

// Up applies every pending migration.
func (m *Migrator) Up(db *sql.DB) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	if err := m.prepare(); err != nil {
		return err
	}
	if err := goose.Up(db, m.dir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Down reverts the most recently applied migration.
func (m *Migrator) Down(db *sql.DB) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	if err := m.prepare(); err != nil {
		return err
	}
	return goose.Down(db, m.dir)
}

// Reset rolls every migration back, in reverse order.
//
// Destructive by design: it drops the schema. Provided because "down" alone
// reverts a single step, so verifying that a rollback path actually works — that
// each Down truly reverses its Up, rather than merely appearing to — requires
// unwinding the whole stack and rebuilding it.
func (m *Migrator) Reset(db *sql.DB) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	if err := m.prepare(); err != nil {
		return err
	}
	return goose.DownTo(db, m.dir, 0)
}

// Status prints which migrations have been applied.
func (m *Migrator) Status(db *sql.DB) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	if err := m.prepare(); err != nil {
		return err
	}
	return goose.Status(db, m.dir)
}

// builtin returns a Migrator over Tusk's own migrations, rejecting any dialect
// Tusk does not support.
func builtin(dialect string) (*Migrator, error) {
	_, dir, err := migrationDir(dialect)
	if err != nil {
		return nil, err
	}
	return NewMigrator(EmbedMigrations, dir), nil
}

// RunMigrations applies all pending Tusk migrations for the configured driver.
func RunMigrations(db *sql.DB, dialect string) error {
	m, err := builtin(dialect)
	if err != nil {
		return err
	}
	return m.Up(db)
}

// RollbackMigration reverts the most recently applied Tusk migration.
func RollbackMigration(db *sql.DB, dialect string) error {
	m, err := builtin(dialect)
	if err != nil {
		return err
	}
	return m.Down(db)
}

// ResetMigrations rolls every Tusk migration back.
func ResetMigrations(db *sql.DB, dialect string) error {
	m, err := builtin(dialect)
	if err != nil {
		return err
	}
	return m.Reset(db)
}

// MigrationStatus prints the state of Tusk's migrations.
func MigrationStatus(db *sql.DB, dialect string) error {
	m, err := builtin(dialect)
	if err != nil {
		return err
	}
	return m.Status(db)
}
