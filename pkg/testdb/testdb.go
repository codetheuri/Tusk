// Package testdb provides a real PostgreSQL database for integration tests.
//
// Unit tests verify a function in isolation; these verify the half of the system
// that only exists once a query is actually issued. That distinction matters more
// than it sounds: GORM builds SQL at runtime from struct tags, so a mismatched
// column type or a broken join compiles cleanly, passes `go vet`, and fails only
// when the query runs. No amount of unit testing catches it.
//
// A service built on Tusk uses the same harness for its own schema, by passing
// its migrator to ConnectWith. The three things this package gets right are each
// a bug someone has already had to find: connecting through pgx because that is
// what production uses, serialising database tests across packages because
// `go test ./...` runs them concurrently, and failing rather than skipping in CI.
//
// SQLite is deliberately not used as a stand-in. It diverges from PostgreSQL on
// precisely the behaviour worth testing — row-level security, partial indexes,
// ON CONFLICT semantics, and type strictness — so a green SQLite suite would be
// evidence of very little.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	// pgx, not lib/pq: gorm.io/driver/postgres opens production connections
	// through stdlib.OpenDB, so a harness on a different driver would test SQL
	// that production never sends. They genuinely differ — the migrator's
	// introspection queries reuse a placeholder, which pgx accepts and lib/pq
	// rejects outright.
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/codetheuri/tusk/v2/database"
	"github.com/codetheuri/tusk/v2/pkg/tenant"
)

// defaultDSN points at a local development PostgreSQL.
//
// Note the database name: tests TRUNCATE every table between cases, so they must
// never be pointed at a database holding data anyone cares about. Keeping the
// "_test" suffix in the default is a deliberate guard rail.
const defaultDSN = "host=127.0.0.1 port=5434 user=root password=root dbname=tusk_test sslmode=disable TimeZone=UTC"

// envDSN overrides the default. CI sets this; developers usually do not need to.
const envDSN = "TEST_DATABASE_URL"

// envRequire turns an unavailable database from a skip into a failure.
//
// This exists because the alternative is worse than having no tests at all: a CI
// run that skips its integration suite reports success while having verified
// nothing. Locally, skipping is the right behaviour — someone reading the code
// should not need PostgreSQL to run `go test ./...`. In CI it must be fatal.
const envRequire = "REQUIRE_DB_TESTS"

// testLockKey identifies the advisory lock that serialises database tests.
//
// `go test ./...` runs packages concurrently, and every test here TRUNCATEs the
// whole database. Two packages overlapping produces failures that look like
// application bugs — a row disappearing mid-test, a foreign key violated against
// a record that was definitely just written — and that reproduce only sometimes.
//
// A lock in the harness, rather than `-p 1` in the Makefile, because the harness
// is what everyone actually uses: a plain `go test ./...` typed by hand has to be
// correct too. Packages with no database tests still run in parallel.
//
// The value is arbitrary; it only has to be a constant nothing else uses.
// Advisory locks are scoped to the current database, so separate test databases
// do not contend.
const testLockKey int64 = 0x5455534b // "TUSK"

// lockTimeout bounds the wait for that lock. A hung test that never reports is
// worse than a failing one, especially in CI where it burns the whole job.
const lockTimeout = 2 * time.Minute

// The lock is held once per test binary and reference-counted, not taken afresh
// on every Connect. A test that connects twice — one database for the owner and
// one for an unprivileged role, say — would otherwise wait on a lock it is
// already holding and stall until the timeout.
var (
	lockMu    sync.Mutex
	lockConn  *sql.Conn
	lockDepth int
)

// Required reports whether a missing database must fail rather than skip.
// CI sets REQUIRE_DB_TESTS so a green build can never mean "tested nothing".
func Required() bool {
	return os.Getenv(envRequire) != ""
}

// DSN returns the connection string tests should use.
func DSN() string {
	if dsn := strings.TrimSpace(os.Getenv(envDSN)); dsn != "" {
		return dsn
	}
	return defaultDSN
}

// Options selects which schema and which database a test runs against.
type Options struct {
	// Migrator supplies the migrations to apply. Nil means Tusk's own — what
	// Tusk's tests want, and almost never what an application's do.
	Migrator *database.Migrator

	// DefaultDSN overrides the fallback connection string. TEST_DATABASE_URL
	// still wins over it, so CI keeps one place to point everything at.
	DefaultDSN string
}

// Connect returns a database migrated with Tusk's own schema.
func Connect(t *testing.T) *gorm.DB {
	t.Helper()
	return ConnectWith(t, Options{})
}

// ConnectWith returns a migrated database, or skips the test when none is
// reachable.
//
// The returned database has every table truncated, so each test starts from a
// known empty state. Cleanup is registered with t.Cleanup rather than left to the
// caller — a forgotten teardown leaks state into whichever test runs next, and
// the resulting failure appears in an unrelated place.
func ConnectWith(t *testing.T, opts Options) *gorm.DB {
	t.Helper()

	dsn := DSN()
	if os.Getenv(envDSN) == "" && opts.DefaultDSN != "" {
		dsn = opts.DefaultDSN
	}

	migrator := opts.Migrator
	if migrator == nil {
		migrator = NewTuskMigrator()
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		skipOrFail(t, fmt.Errorf("open: %w", err))
		return nil
	}

	// sql.Open does not connect. Without an explicit ping, an unreachable
	// database surfaces later as a confusing query error instead of a clear
	// "there is no database here".
	sqlDB.SetConnMaxLifetime(time.Minute)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		skipOrFail(t, fmt.Errorf("ping %s: %w", redact(dsn), err))
		return nil
	}

	if err := migrator.Up(sqlDB); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed to migrate test database: %v", err)
		return nil
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		// Silent: a failing test should be legible. Every statement echoed to
		// stdout buries the actual assertion failure.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed to open gorm connection: %v", err)
		return nil
	}

	// Mirrors database.Connect. Tests that never opt a model in to tenancy are
	// therefore running the same callbacks production does, which is what makes
	// them evidence that tenancy stays inert.
	if err := tenant.Register(gormDB); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed to register tenant callbacks: %v", err)
		return nil
	}

	// Serialise before the first TRUNCATE, release after the last one.
	lockDatabase(t, sqlDB)

	Truncate(t, gormDB)
	t.Cleanup(func() {
		Truncate(t, gormDB)
		unlockDatabase()
		_ = sqlDB.Close()
	})

	return gormDB
}

// lockDatabase takes the shared advisory lock, or joins one this binary already
// holds.
//
// The connection is pinned deliberately: advisory locks belong to a session, so
// releasing from a different pooled connection would silently do nothing.
func lockDatabase(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	lockMu.Lock()
	defer lockMu.Unlock()

	if lockDepth > 0 {
		lockDepth++
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("failed to pin a connection for the test lock: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", testLockKey); err != nil {
		_ = conn.Close()
		t.Fatalf("timed out after %s waiting for the database test lock; "+
			"another test package may be stuck holding it: %v", lockTimeout, err)
	}

	lockConn = conn
	lockDepth = 1
}

func unlockDatabase() {
	lockMu.Lock()
	defer lockMu.Unlock()

	if lockDepth == 0 {
		return
	}
	lockDepth--
	if lockDepth > 0 {
		return
	}

	_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", testLockKey)
	_ = lockConn.Close()
	lockConn = nil
}

// Truncate empties every application table, resetting identity sequences.
//
// TRUNCATE is used rather than wrapping each test in a rolled-back transaction,
// because the code under test opens transactions of its own — the permission
// synchroniser, for one — and nesting those inside a test transaction changes the
// behaviour being verified.
func Truncate(t *testing.T, db *gorm.DB) {
	t.Helper()

	var tables []string
	err := db.Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
	`).Scan(&tables).Error
	if err != nil {
		t.Fatalf("failed to list tables for truncation: %v", err)
	}
	if len(tables) == 0 {
		return
	}

	for i, name := range tables {
		tables[i] = `"` + name + `"`
	}

	// CASCADE because foreign keys otherwise make the order significant; RESTART
	// IDENTITY so a test asserting on ID 1 is not defeated by a previous run.
	stmt := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("failed to truncate test tables: %v", err)
	}
}

// skipOrFail skips locally and fails in CI, according to REQUIRE_DB_TESTS.
func skipOrFail(t *testing.T, err error) {
	t.Helper()
	if os.Getenv(envRequire) != "" {
		t.Fatalf("%s is set but no test database is reachable: %v", envRequire, err)
	}
	t.Skipf("no test database reachable, skipping integration test (set %s to make this fatal): %v", envDSN, err)
}

// redact removes the password before a DSN reaches a log or a test failure.
func redact(dsn string) string {
	fields := strings.Fields(dsn)
	for i, f := range fields {
		if strings.HasPrefix(f, "password=") {
			fields[i] = "password=****"
		}
	}
	return strings.Join(fields, " ")
}

// NewTuskMigrator returns a migrator over Tusk's own schema, for a test that
// wants Tusk's tables alongside an application's.
func NewTuskMigrator() *database.Migrator {
	return database.NewMigrator(database.EmbedMigrations, "migrations/postgres")
}
