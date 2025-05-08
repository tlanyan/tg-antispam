package storage

import (
	"fmt"
	"time"

	"tg-antispam/internal/config"
	customlogger "tg-antispam/internal/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// DB is the global database connection
	DB *gorm.DB
)

// Initialize sets up the database connection based on configuration
func Initialize(cfg *config.Config) error {
	// Skip if database is disabled
	if !cfg.Database.Enabled {
		customlogger.Warning("Database support is disabled")
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.Charset,
	)

	customlogger.Infof("Connecting to database: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 创建使用我们自定义logger的GORM日志适配器
	dbLogger := NewCustomGormLogger(cfg.Logger.Level)

	var err error
	// Initialize database with our custom logger
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	customlogger.Infof("Database connection established successfully")
	return nil
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}

// IsEnabled returns true if database support is enabled
func IsEnabled(cfg *config.Config) bool {
	return cfg.Database.Enabled
}
