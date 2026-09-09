package database_test

import (
	"database/sql"
	"regexp"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/codetheuri/tusk/database"
	"github.com/codetheuri/tusk/pkg/testdb"
)

// migrationTestDB is created and dropped by this test.
//
// It deliberately does not share the database used by the other integration
// tests: this one drops every table, and Go runs different packages in parallel,
// so sharing would tear the schema out from under a concurrently running suite.
const migrationTestDB = "tusk_test_migrations"

var dbNamePattern = regexp.MustCompile(`dbname=\S+`)

// dsnFor rewrites the configured test DSN to point at a different database on the
// same server, so credentials and host stay in one place.
func dsnFor(name string) string {
	return dbNamePattern.ReplaceAllString(testdb.DSN(), "dbname="+name)
}

// TestMigrations_UpDownUp verifies the schema can be built, torn down, and built
// again.
//
// The Down direction is the half nobody exercises until the day a deploy has to
// be rolled back — which is the worst possible moment to discover a typo in it.
func TestMigrations_UpDownUp(t *testing.T) {
	admin, err := sql.Open("pgx", dsnFor("postgres"))
	if err != nil {
		t.Skipf("cannot open admin connection, skipping: %v", err)
	}
	defer admin.Close()

	if err := admin.Ping(); err != nil {
		// Same policy as testdb.Connect: skip locally, fail in CI.
		if testdb.Required() {
			t.Fatalf("REQUIRE_DB_TESTS is set but no database is reachable: %v", err)
		}
		t.Skipf("no test database reachable, skipping: %v", err)
	}

	// A previous crashed run may have left the database behind.
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + migrationTestDB); err != nil {
		t.Fatalf("failed to drop stale test database: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + migrationTestDB); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		// Connections must be closed before the database can be dropped.
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + migrationTestDB)
	})

	db, err := sql.Open("pgx", dsnFor(migrationTestDB))
	if err != nil {
		t.Fatalf("failed to connect to the migration test database: %v", err)
	}

	expected := []string{
		"users", "user_profiles", "refresh_tokens",
		"permissions", "roles", "role_permissions", "user_roles",
		"sessions",
	}

	// --- Up ---
	if err := database.RunMigrations(db, "postgres"); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}
	for _, table := range expected {
		if !tableExists(t, db, table) {
			t.Errorf("after migrate up, table %q is missing", table)
		}
	}

	// --- Down ---
	// Every migration, not just the most recent: rolling back one step would
	// leave earlier migrations applied and prove nothing about their Down.
	if err := database.ResetMigrations(db, "postgres"); err != nil {
		t.Fatalf("migrate reset failed: %v", err)
	}
	for _, table := range expected {
		if tableExists(t, db, table) {
			t.Errorf("after migrate down, table %q still exists", table)
		}
	}

	// --- Up again ---
	// Re-applying proves the Down left the schema in a state the Up can rebuild,
	// rather than merely appearing to succeed.
	if err := database.RunMigrations(db, "postgres"); err != nil {
		t.Fatalf("migrate up after rollback failed: %v", err)
	}
	for _, table := range expected {
		if !tableExists(t, db, table) {
			t.Errorf("after re-applying migrations, table %q is missing", table)
		}
	}

	db.Close()
}

// TestMigrations_UnknownDriverIsRejected guards the dialect routing: an
// unsupported driver must fail loudly rather than silently running no migrations.
func TestMigrations_UnknownDriverIsRejected(t *testing.T) {
	err := database.RunMigrations(nil, "cassandra")
	if err == nil {
		t.Fatal("expected an unsupported driver to be rejected")
	}
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname='public' AND tablename=$1)`,
		name,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check for table %q: %v", name, err)
	}
	return exists
}
