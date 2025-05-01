package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Setup configures logging to output to both stdout and a rotating log file
func Setup(logDir string) error {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Format current date for the log filename
	currentDate := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("tg-antispam-%s.log", currentDate))

	// Configure rotating logger
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // megabytes
		MaxBackups: 30, // number of backups
		MaxAge:     90, // days
		Compress:   true,
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
func GetRotatingLogWriter(logDir, prefix string) io.Writer {
	// Format current date for the log filename
	currentDate := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", prefix, currentDate))

	// Configure rotating logger
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // megabytes
		MaxBackups: 30, // number of backups
		MaxAge:     90, // days
		Compress:   true,
	}

	// Return multi-writer that writes to both stdout and the log file
	return io.MultiWriter(os.Stdout, rotatingLogger)
}
