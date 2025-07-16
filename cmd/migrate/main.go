package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/database/migrations"
	"github.com/codetheuri/todolist/database/seeders"
	"gorm.io/gorm"
)

var databaseURl string
var appMode string

func init() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// databaseURl = cfg.DBUser + ":" + cfg.DBPass + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName
	databaseURl = cfg.DbURL
	if databaseURl == "" {
		log.Fatal("Database URL is not set in the configuration")
	}

	appMode = cfg.AppMode
	if appMode == "" {
		log.Fatal("App mode is not set in the configuration")
	}
}
func main() {
	upCmd := flag.NewFlagSet("up", flag.ExitOnError)
	downCmd := flag.NewFlagSet("down", flag.ExitOnError)
	createCmd := flag.NewFlagSet("create", flag.ExitOnError)
	freshcmd := flag.NewFlagSet("fresh", flag.ExitOnError)
	seeders := flag.NewFlagSet("seeders", flag.ExitOnError)

	downSteps := downCmd.Int("steps", 1, "Number of migrations to revert")
	createName := createCmd.String("name", "", "Name of the new migration (e.g create_users_table)")
	seedName := seeders.String("name", "", "Optional : Name of specific seeder to run (eg. 01UsersTableSeeder)")
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/migrate <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  up              Apply all pending migrations")
		fmt.Println("  down [-steps N] Roll back the last N migrations (default: 1)")
		fmt.Println("  create -name NAME Generate a new migration file")
		fmt.Println("fresh            Drop all tables and reapply all migrations")
		fmt.Println("seed   	  Run all registered seeders")
		fmt.Println("seed [-name NAME] Run a specific seeder")
		os.Exit(1)
	}

	command := os.Args[1]

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	//ensure schema migrations table exists
	if err := db.AutoMigrate(&migrations.SchemaMigrationModel{}); err != nil {
		log.Fatalf("Failed to auto-migrate schema_migrations table : %v", err)
	}

	switch command {
	case "up":
		upCmd.Parse(os.Args[2:])
		log.Println("Applying pending migrations ...")
		runMigrations(db, "up")
	case "down":
		downCmd.Parse(os.Args[2:])
		log.Printf("Reverting back %d migrations(s)...\n", *downSteps)
		for i := 0; i < *downSteps; i++ {
			runMigrations(db, "down")
		}
	case "create":
		createCmd.Parse(os.Args[2:])
		if *createName == "" {
			log.Fatal("Error: -name flag is required for 'create' command.")
		}
		createMigrationFile(*createName)
	case "fresh":
		freshcmd.Parse(os.Args[2:])
		if appMode != "development" && appMode != "dev" {
			log.Fatalf("Error: 'fresh' command can only be run in development mode. Current mode: %s", appMode)
		}
		log.Println("Dropping all tables and reapplying all migrations...")
		runFresh(db)
		log.Println("Fresh migration completed successfully.")
	case "seed":
		seeders.Parse(os.Args[2:])
		log.Println("Running all registered seeders...")
		runSeeders(db, *seedName)
		log.Println("Database seeding completed.")
	case "help":
		fmt.Println("Usage: go run ./cmd/migrate <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  up              Apply all pending migrations")
		fmt.Println("  down [-steps N] Roll back the last N migrations (default: 1)")
		fmt.Println("  create -name NAME Generate a new migration file")
		fmt.Println("  help            Show this help message")
		fmt.Println("  fresh           Drop all tables and reapply all migrations")
		fmt.Println("  seed            Run all registered seeders")
		fmt.Println("  seed -name NAME Run a specific seeder")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, down, create, help")
		fmt.Println("Use 'go run ./cmd/migrate help' for more information.")
		os.Exit(1)
	}

}

// up or down based on direction
func runMigrations(db *gorm.DB, direction string) {
	// sort registered migrations by version
	sort.Slice(migrations.RegisteredMigrations, func(i, j int) bool {
		return migrations.RegisteredMigrations[i].Version() < migrations.RegisteredMigrations[j].Version()
	})
	var appliedMigrations []migrations.SchemaMigrationModel
	if err := db.Find(&appliedMigrations).Error; err != nil {
		log.Fatalf("Failed to fetch applied migrations: %v", err)
	}
	appliedversions := make(map[string]bool)
	for _, m := range appliedMigrations {
		appliedversions[m.Version] = true
	}
	switch direction {
	case "up":
		for _, m := range migrations.RegisteredMigrations {
			if !appliedversions[m.Version()] {
				log.Printf("Applying migration: %s (%s)\n", m.Name(), m.Version())
				if err := db.Transaction(func(tx *gorm.DB) error {
					return m.Up(tx)
				}); err != nil {
					log.Fatalf("Failed to apply migration %s: %v", m.Name(), err)
				}
				if err := db.Create(&migrations.SchemaMigrationModel{
					Version: m.Version(),
					Name:    m.Name(),
				}).Error; err != nil {
					log.Fatalf("Failed to record migration %s: %v", m.Name(), err)
				}
				log.Printf("Successfully applied migration: %s\n", m.Name())
			}
		}
		log.Printf("All pending migrations applied.")
	case "down":
		if len(appliedMigrations) == 0 {
			log.Println("No migrations to roll back.")
			return
		}
		//sort applied migrations in  desc order to get the latest first
		sort.Slice(appliedMigrations, func(i, j int) bool {
			return appliedMigrations[i].Version > appliedMigrations[j].Version
		})
		latestApplied := appliedMigrations[0]
		var migrationToRollback migrations.Migration
		for _, m := range migrations.RegisteredMigrations {
			if m.Version() == latestApplied.Version {
				migrationToRollback = m
				break
			}
		}
		if migrationToRollback == nil {
			log.Fatalf("Migration %s not found in registered migrations. Cannot revert", latestApplied.Version)
		}
		log.Printf("Reverting back migration: %s (%s)\n", migrationToRollback.Name(), migrationToRollback.Version())
		if err := db.Transaction(func(tx *gorm.DB) error {
			return migrationToRollback.Down(tx)
		}); err != nil {
			log.Fatalf("Failed to revert back migration %s: %v", migrationToRollback.Name(), err)
		}
		if err := db.Where("version = ?", migrationToRollback.Version()).Delete(&migrations.SchemaMigrationModel{}).Error; err != nil {
			log.Fatalf("Failed to remove reverted back migration record %s: ", migrationToRollback.Name())
		}
		log.Printf("Successfully reverted back migration: %s\n", migrationToRollback.Name())
	}
}
func createMigrationFile(name string) {
	timestamp := time.Now().Format("20060102150405") // format: YYYYMMDDHHMMSS
	version := timestamp

	fileName := fmt.Sprintf("database/migrations/%s_%s.go", version, name)
	structName := strings.ReplaceAll(name, "_", "")
	structName = strings.ToUpper(string(structName[0])) + structName[1:] // Capitalize first letter
	content := fmt.Sprintf(`package migrations
	import (
		"gorm.io/gorm"
		"log"
)
		// %s struct implements migration interface
		type %s struct {}

		func (m *%s) Version() string{
			return "%s"
			}
		func (m *%s) Name() string {
			return "%s"
		}	
			//up migration method
		func (m *%s) Up(tx *gorm.DB) error {
		log.Printf("Running Up migration: %%s", m.Name())
		// Add your migration logic here
		//example : 
		// 1. using SQL calls
		// if  err := tx.Exec("CREATE TABLE IF NOT EXISTS example (id INT PRIMARY KEY)").Error; err != nil {
		// 	return err
		// }

		// 2. using gorm methods
		//type NewModel struct {
		// gorm.Model
		// Field1 string
		//}
		// if err := tx.AutoMigrate(&NewModel{}); err != nil {
		// 	return err
		// }
		log.Printf("Successfully applied Up migration: %%s", m.Name())
		return nil
		}
		//down migration method
		func (m *%s) Down(tx *gorm.DB) error {
		log.Printf("Running Down migration: %%s", m.Name())
		// Example:
		// 1. using SQL calls
		// if err := tx.Exec("DROP TABLE IF EXISTS example").Error; err != nil {
		// 	return err
		// }

		// 2. using gorm methods
		// if err := tx.Migrator().DropTable("new_models"); err != nil {
		// 	return err
		// }
		log.Printf("Successfully applied Down migration: %%s", m.Name())
		return nil
		}

		func init() {
		  // Register the migration
		  RegisteredMigrations = append(RegisteredMigrations, &%s{})
		}
`,
		structName,          // for struct name
		structName,          // method receiver
		structName, version, // for Version method
		structName, name,
		structName,
		structName,
		// structName,
		// structName,
		structName, // for init registration

	)
	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		log.Fatalf("Failed to create migration file : %v", err)
	}
	log.Printf("Migration file created: %s", fileName)
}

// run fresh migration
func runFresh(db *gorm.DB) {
	migrator := db.Migrator()
	tableNames, err := migrator.GetTables()
	if err != nil {
		log.Fatalf("Failed to get tables: %v", err)
	}

	// var tablesToDrop []string
	// for _, tableName := range tableNames {
	// 	tablesToDrop = append(tablesToDrop, tableName)
	// }

	if len(tableNames) > 0 {
		log.Printf("Dropping %d tables: %v\n", len(tableNames), tableNames)
		tablesAsInterfaces := make([]interface{}, len(tableNames))
		for i, tablename := range tableNames {
			tablesAsInterfaces[i] = tablename
		}
		if err := migrator.DropTable(tablesAsInterfaces...); err != nil {
			log.Fatalf("Failed to drop tables: %v", err)
		}
		log.Println("All tables dropped successfully.")
	} else {
		log.Println("No tables found to drop.")
	}

	log.Println("Re-Creating schema_migrations table...")
	if err := db.AutoMigrate(&migrations.SchemaMigrationModel{}); err != nil {
		log.Fatalf("Failed to re-create schema_migrations table: %v", err)
	}

	log.Println("Reapplying all migrations...")
	runMigrations(db, "up")
	log.Println("All migration re-applied.")

}

func runSeeders(db *gorm.DB, seederName string) {
	if len(seeders.RegisteredSeeders) == 0 {
		log.Println("No database seeders registered.")
		return
	}
    
	sort.Slice(seeders.RegisteredSeeders, func(i, j int) bool {
		return seeders.RegisteredSeeders[i].Name() < seeders.RegisteredSeeders[j].Name()
	})
   
	for _, s := range seeders.RegisteredSeeders {
		if seederName != "" && s.Name() != seederName {
			log.Printf("Skipping seeder: %s (not '%s')\n", s.Name(), seederName)
			continue
		}
		log.Printf("Executing seeder: %s", s.Name())
		if err := s.Run(db); err != nil {
			log.Fatalf("Failed to run seeder %s: %v", s.Name(), err)
		}
		log.Printf("Seeder %s completed.", s.Name())
	}
    if seederName != "" {
		found := false
		for _, s := range seeders.RegisteredSeeders {
			if s.Name() == seederName {
				found = true
				break
			}
		}
		if !found {
		log.Fatalf("Error: seeder '%s' not found ", seederName)
		}
	}
}
