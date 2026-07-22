package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codetheuri/tusk/config"
	// Import modules to trigger explicit permission registration in init()
	_ "github.com/codetheuri/tusk/internal/auth"
	appDatabase "github.com/codetheuri/tusk/internal/platform/database"
	"github.com/codetheuri/tusk/pkg/authz"
	"github.com/codetheuri/tusk/pkg/logger"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "auth":
		handleAuthCommand(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleAuthCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tusk auth <subcommand>")
		fmt.Println("Subcommands:")
		fmt.Println("  sync    Synchronize code-first permissions into the database")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "sync":
		syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
		prune := syncCmd.Bool("prune", false, "Remove obsolete permissions from database that are no longer in code")
		_ = syncCmd.Parse(args[1:])

		log := logger.NewConsoleLogger()
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatal("Failed to load configuration", err)
		}

		db, err := appDatabase.NewGoRMDB(cfg, log)
		if err != nil {
			log.Fatal("Failed to connect to database", err)
		}

		// Ensure RBAC schema exists
		if err := db.AutoMigrate(&authz.PermissionRecord{}, &authz.Role{}, &authz.RolePermission{}, &authz.UserRole{}); err != nil {
			log.Fatal("Failed to migrate authorization tables", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		syncer := authz.NewSynchronizer(db, authz.DefaultRegistry())
		result, err := syncer.Sync(ctx, *prune)
		if err != nil {
			log.Fatal("Permission synchronization failed", err)
		}

		log.Info(fmt.Sprintf("Authorization Sync Complete: %d inserted, %d updated, %d pruned",
			result.Inserted, result.Updated, result.Pruned))
	default:
		fmt.Printf("Unknown auth subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Tusk Framework CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  tusk <command> [arguments]")
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  auth sync [--prune]    Sync registered code permissions to runtime database")
}
