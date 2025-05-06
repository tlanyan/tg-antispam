package main

import (
	"flag"
	"fmt"
	"log"

	"tg-antispam/internal/config"
	"tg-antispam/internal/models"
	"tg-antispam/internal/storage"

	"gorm.io/gorm"
)

func main() {
	// Define command line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	action := flag.String("action", "migrate", "Action to perform (migrate, reset, status)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if database is enabled
	if !cfg.Database.Enabled {
		log.Fatalf("Database is not enabled in configuration")
	}

	// Initialize database connection
	if err := storage.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := storage.GetDB()
	if db == nil {
		log.Fatalf("Failed to get database connection")
	}

	// Perform requested action
	switch *action {
	case "migrate":
		if err := migrateDatabase(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration completed successfully")
	case "reset":
		if err := resetDatabase(db); err != nil {
			log.Fatalf("Reset failed: %v", err)
		}
		log.Println("Database reset completed successfully")
	case "status":
		if err := checkStatus(db); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

// migrateDatabase performs database migration
func migrateDatabase(db *gorm.DB) error {
	fmt.Println("Migrating database...")

	// Migrate models
	if err := db.AutoMigrate(&models.GroupInfo{}); err != nil {
		return fmt.Errorf("failed to migrate GroupInfo model: %w", err)
	}

	return nil
}

// resetDatabase drops tables and recreates them
func resetDatabase(db *gorm.DB) error {
	fmt.Println("Resetting database...")

	// Confirm reset operation
	fmt.Print("WARNING: This will delete all data! Are you sure? (y/N): ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "y" && confirmation != "Y" {
		return fmt.Errorf("operation cancelled by user")
	}

	// Drop tables in reverse order to avoid foreign key constraints
	if err := db.Migrator().DropTable(&models.GroupInfo{}); err != nil {
		return fmt.Errorf("failed to drop GroupInfo table: %w", err)
	}

	// Recreate tables
	return migrateDatabase(db)
}

// checkStatus checks the database status
func checkStatus(db *gorm.DB) error {
	fmt.Println("Checking database status...")

	// Check if tables exist
	if db.Migrator().HasTable(&models.GroupInfo{}) {
		fmt.Println("✅ GroupInfo table exists")

		// Count records
		var count int64
		db.Model(&models.GroupInfo{}).Count(&count)
		fmt.Printf("   - Contains %d records\n", count)
	} else {
		fmt.Println("❌ GroupInfo table does not exist")
	}

	// Additional checks can be added here

	return nil
}
