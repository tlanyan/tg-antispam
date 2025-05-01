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

// Setup configures logging to output to both stdout and a rotating log file
func Setup(cfg *config.Config) error {
	logDir := cfg.Logger.Directory

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Format current date for the log filename
	currentDate := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("tg-antispam-%s.log", currentDate))

	// Configure rotating logger using config values
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.Logger.Rotation.MaxSize,
		MaxBackups: cfg.Logger.Rotation.MaxBackups,
		MaxAge:     cfg.Logger.Rotation.MaxAge,
		Compress:   cfg.Logger.Rotation.Compress,
	}

	// Create multi-writer to log to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, rotatingLogger)

	// Set standard logger output to the multi-writer
	log.SetOutput(multiWriter)

	// Set log flags to include date, time, and file information
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Printf("Logging initialized: writing to %s", logFilePath)
	return nil
}

// GetRotatingLogWriter returns a rotating log writer for custom loggers
func GetRotatingLogWriter(cfg *config.Config, prefix string) io.Writer {
	// Format current date for the log filename
	currentDate := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(cfg.Logger.Directory, fmt.Sprintf("%s-%s.log", prefix, currentDate))

	// Configure rotating logger using config values
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    cfg.Logger.Rotation.MaxSize,
		MaxBackups: cfg.Logger.Rotation.MaxBackups,
		MaxAge:     cfg.Logger.Rotation.MaxAge,
		Compress:   cfg.Logger.Rotation.Compress,
	}

	// Return multi-writer that writes to both stdout and the log file
	return io.MultiWriter(os.Stdout, rotatingLogger)
}
