package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.NewDBConfigFromEnv()

	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	db, err := database.New(&cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	cmd := args[0]

	switch cmd {
	case "up":
		fmt.Println("Running migrations...")
		if err := db.Migrate(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back all migrations...")
		if err := db.MigrateDown(); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")

	case "step":
		if len(args) < 2 {
			log.Fatal("step requires a number argument (positive=up, negative=down)")
		}
		var n int
		if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil {
			log.Fatalf("Invalid step number: %v", err)
		}
		fmt.Printf("Running %d migration steps...\n", n)
		if err := db.MigrateSteps(n); err != nil {
			log.Fatalf("Migration steps failed: %v", err)
		}
		fmt.Println("Migration steps completed successfully")

	case "version":
		version, dirty, err := db.MigrationVersion()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			fmt.Printf("Version: %d (dirty)\n", version)
		} else {
			fmt.Printf("Version: %d\n", version)
		}

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up        Run all pending migrations")
	fmt.Println("  down      Rollback all migrations")
	fmt.Println("  step N    Run N migrations (positive=up, negative=down)")
	fmt.Println("  version   Print current migration version")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  DB_HOST      Database host (default: localhost)")
	fmt.Println("  DB_PORT      Database port (default: 5432)")
	fmt.Println("  DB_USER      Database user (default: aule)")
	fmt.Println("  DB_PASSWORD  Database password (default: aule)")
	fmt.Println("  DB_NAME      Database name (default: aule)")
	fmt.Println("  DB_SSLMODE   SSL mode (default: disable)")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
