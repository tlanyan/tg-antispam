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
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	action := flag.String("action", "migrate", "Action to perform (migrate, reset, status)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if !cfg.Database.Enabled {
		log.Fatalf("Database is not enabled in configuration")
	}

	if err := storage.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := storage.GetDB()
	if db == nil {
		log.Fatalf("Failed to get database connection")
	}

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

func migrateDatabase(db *gorm.DB) error {
	fmt.Println("Migrating database...")

	if err := db.AutoMigrate(&models.GroupInfo{}); err != nil {
		return fmt.Errorf("failed to migrate GroupInfo model: %w", err)
	}

	if err := db.AutoMigrate(&models.BanRecord{}); err != nil {
		return fmt.Errorf("failed to migrate BanRecord model: %w", err)
	}

	return nil
}

func resetDatabase(db *gorm.DB) error {
	fmt.Println("Resetting database...")

	fmt.Print("WARNING: This will delete all data! Are you sure? (y/N): ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "y" && confirmation != "Y" {
		return fmt.Errorf("operation cancelled by user")
	}

	if err := db.Migrator().DropTable(&models.GroupInfo{}); err != nil {
		return fmt.Errorf("failed to drop GroupInfo table: %w", err)
	}

	return migrateDatabase(db)
}

func checkStatus(db *gorm.DB) error {
	fmt.Println("Checking database status...")

	if db.Migrator().HasTable(&models.GroupInfo{}) {
		fmt.Println("✅ GroupInfo table exists")

		var count int64
		db.Model(&models.GroupInfo{}).Count(&count)
		fmt.Printf("   - Contains %d records\n", count)
	} else {
		fmt.Println("❌ GroupInfo table does not exist")
	}

	// others...

	return nil
}
