package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/database"
	"github.com/pressly/goose/v3"
	// Ensure drivers are loaded
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: go run ./cmd/migrate/main.go [up|down|status]")
		os.Exit(1)
	}

	command := args[0]

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	driver := cfg.DBDriver
	if driver == "pgsql" {
		driver = "postgres"
	}

	db, err := sql.Open(driver, cfg.DbURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Tell goose to use the embedded migrations
	goose.SetBaseFS(database.EmbedMigrations)

	if err := goose.SetDialect(driver); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	switch command {
	case "up":
		if err := goose.Up(db, "migrations"); err != nil {
			log.Fatalf("Goose up failed: %v", err)
		}
	case "down":
		if err := goose.Down(db, "migrations"); err != nil {
			log.Fatalf("Goose down failed: %v", err)
		}
	case "status":
		if err := goose.Status(db, "migrations"); err != nil {
			log.Fatalf("Goose status failed: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	log.Printf("Migration command '%s' completed successfully.", command)
}
