// Command migrate applies, reverts, and reports database schema migrations.
//
// Usage: go run ./cmd/migrate [up|down|status]
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	// PostgreSQL driver, registered for its side effects.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/database"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: go run ./cmd/migrate [up|down|status]")
		os.Exit(1)
	}
	command := args[0]

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// pgx, whatever DB_DRIVER is spelled as. Tusk is PostgreSQL-only (config
	// rejects anything else), and gorm.io/driver/postgres opens the running
	// application's connections through pgx — so applying migrations through a
	// second driver would mean the schema is built by one and used by another.
	db, err := sql.Open("pgx", cfg.DbURL)
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Fail here rather than at the first query: sql.Open does not actually
	// connect, so without a ping a bad DSN surfaces as a confusing migration
	// error instead of a connection error.
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to reach the database: %v", err)
	}

	// The goose wiring — dialect, embedded FS, and which directory belongs to
	// which driver — lives in package database so this CLI and the application
	// cannot disagree about where migrations come from.
	switch command {
	case "up":
		err = database.RunMigrations(db, cfg.DBDriver)
	case "down":
		err = database.RollbackMigration(db, cfg.DBDriver)
	case "reset":
		err = database.ResetMigrations(db, cfg.DBDriver)
	case "status":
		err = database.MigrationStatus(db, cfg.DBDriver)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("migration %q failed: %v", command, err)
	}

	log.Printf("Migration command %q completed successfully.", command)
}
