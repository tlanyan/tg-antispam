package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"tg-antispam/internal/config"
)

// createLogFilePath generates a log file path with the current date
func createLogFilePath(logDir, prefix string) string {
	currentDate := time.Now().Format("2006-01-02")
	return filepath.Join(logDir, fmt.Sprintf("%s-%s.log", prefix, currentDate))
}

// createRotatingLogger creates a lumberjack rotating logger
func createRotatingLogger(logFilePath string, cfg *config.Config) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.Logger.Rotation.MaxSize,
		MaxBackups: cfg.Logger.Rotation.MaxBackups,
		MaxAge:     cfg.Logger.Rotation.MaxAge,
		Compress:   cfg.Logger.Rotation.Compress,
	}
}

// createMultiWriter creates a writer that outputs to both stdout and log file
func createMultiWriter(rotatingLogger io.Writer) io.Writer {
	return io.MultiWriter(os.Stdout, rotatingLogger)
}

// Setup configures logging to output to both stdout and a rotating log file
func Setup(cfg *config.Config) error {
	logDir := cfg.Logger.Directory

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFilePath := createLogFilePath(logDir, "tg-antispam")
	rotatingLogger := createRotatingLogger(logFilePath, cfg)
	multiWriter := createMultiWriter(rotatingLogger)

	// Set standard logger output to the multi-writer
	log.SetOutput(multiWriter)

	// Set log flags to include date, time, and file information
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Printf("Logging initialized: writing to %s", logFilePath)
	return nil
}

// GetRotatingLogWriter returns a rotating log writer for custom loggers
func GetRotatingLogWriter(cfg *config.Config, prefix string) io.Writer {
	logFilePath := createLogFilePath(cfg.Logger.Directory, prefix)
	rotatingLogger := createRotatingLogger(logFilePath, cfg)
	return createMultiWriter(rotatingLogger)
}
