package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"tg-antispam/internal/config"
)

type LogLevel string

const (
	LevelDebug   LogLevel = "DEBUG"
	LevelInfo    LogLevel = "INFO"
	LevelWarning LogLevel = "WARNING"
	LevelError   LogLevel = "ERROR"
	LevelFatal   LogLevel = "FATAL"
)

// levelOrder defines the severity order of log levels
var levelOrder = map[LogLevel]int{
	LevelDebug:   0,
	LevelInfo:    1,
	LevelWarning: 2,
	LevelError:   3,
	LevelFatal:   4,
}

// customLogger is a wrapper for the standard logger that allows timezone and format customization
type customLogger struct {
	logger      *log.Logger
	timezone    *time.Location
	format      string
	timeFormat  string
	multiWriter io.Writer
	level       LogLevel
}

var globalLogger *customLogger

// ParseLogLevel converts a string to a LogLevel
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(level) {
	case string(LevelDebug):
		return LevelDebug, nil
	case string(LevelInfo):
		return LevelInfo, nil
	case string(LevelWarning):
		return LevelWarning, nil
	case string(LevelError):
		return LevelError, nil
	case string(LevelFatal):
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// ShouldLog determines if a log message at the given level should be logged
func (l *customLogger) ShouldLog(level LogLevel) bool {
	levelValue, ok := levelOrder[level]
	if !ok {
		// If level is unknown, default to allowing the log
		return true
	}

	minLevelValue, ok := levelOrder[l.level]
	if !ok {
		// If minLevel is unknown, default to INFO
		minLevelValue = levelOrder[LevelInfo]
	}

	return levelValue >= minLevelValue
}

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

// createCustomLogger creates a logger with the specified timezone and format
func createCustomLogger(multiWriter io.Writer, timezone *time.Location, format string, timeFormat string, level LogLevel) *customLogger {
	logger := log.New(multiWriter, "", 0) // We'll handle flags ourselves for more flexibility
	return &customLogger{
		logger:      logger,
		timezone:    timezone,
		format:      format,
		timeFormat:  timeFormat,
		multiWriter: multiWriter,
		level:       level,
	}
}

// formatLogLine formats a log line according to the configured format
func (l *customLogger) formatLogLine(level LogLevel, calldepth int, message string) string {
	now := time.Now().In(l.timezone)
	timeStr := now.Format(l.timeFormat)

	// Get file and line info
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	shortFile := filepath.Base(file)

	// Replace placeholders in the format string
	result := l.format
	result = strings.ReplaceAll(result, "%{level}", string(level))
	result = strings.ReplaceAll(result, "%{time}", timeStr)
	result = strings.ReplaceAll(result, "%{file}", shortFile)
	result = strings.ReplaceAll(result, "%{line}", fmt.Sprintf("%d", line))
	result = strings.ReplaceAll(result, "%{message}", message)

	return result
}

// log outputs a log message if the level is enabled
func (l *customLogger) log(level LogLevel, calldepth int, message string) {
	if !l.ShouldLog(level) {
		return
	}
	logLine := l.formatLogLine(level, calldepth+1, message)
	l.logger.Output(calldepth, logLine)
}

func (l *customLogger) Debug(v ...interface{}) {
	if !l.ShouldLog(LevelDebug) {
		return
	}
	message := fmt.Sprint(v...)
	l.log(LevelDebug, 3, message)
}

func (l *customLogger) Debugf(format string, v ...interface{}) {
	if !l.ShouldLog(LevelDebug) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.log(LevelDebug, 3, message)
}

func (l *customLogger) Info(v ...interface{}) {
	if !l.ShouldLog(LevelInfo) {
		return
	}
	message := fmt.Sprint(v...)
	l.log(LevelInfo, 3, message)
}

func (l *customLogger) Infof(format string, v ...interface{}) {
	if !l.ShouldLog(LevelInfo) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.log(LevelInfo, 3, message)
}

func (l *customLogger) Warning(v ...interface{}) {
	if !l.ShouldLog(LevelWarning) {
		return
	}
	message := fmt.Sprint(v...)
	l.log(LevelWarning, 3, message)
}

func (l *customLogger) Warningf(format string, v ...interface{}) {
	if !l.ShouldLog(LevelWarning) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.log(LevelWarning, 3, message)
}

func (l *customLogger) Error(v ...interface{}) {
	if !l.ShouldLog(LevelError) {
		return
	}
	message := fmt.Sprint(v...)
	l.log(LevelError, 3, message)
}

func (l *customLogger) Errorf(format string, v ...interface{}) {
	if !l.ShouldLog(LevelError) {
		return
	}
	message := fmt.Sprintf(format, v...)
	l.log(LevelError, 3, message)
}

func (l *customLogger) Fatal(v ...interface{}) {
	message := fmt.Sprint(v...)
	l.log(LevelFatal, 3, message)
	os.Exit(1)
}

func (l *customLogger) Fatalf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.log(LevelFatal, 3, message)
	os.Exit(1)
}

func (l *customLogger) Printf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

func (l *customLogger) Print(v ...interface{}) {
	l.Info(v...)
}

// Println is a custom implementation of Println for our logger (INFO level)
func (l *customLogger) Println(v ...interface{}) {
	message := fmt.Sprintln(v...)
	if len(message) > 0 && message[len(message)-1] == '\n' {
		message = message[:len(message)-1]
	}
	l.Info(message)
}

// Setup configures logging to output to both stdout and a rotating log file
func Setup(cfg *config.Config) error {
	logDir := cfg.Logger.Directory

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	timezone, err := time.LoadLocation(cfg.Logger.Timezone)
	if err != nil {
		return fmt.Errorf("failed to load timezone %s: %w", cfg.Logger.Timezone, err)
	}

	level, err := ParseLogLevel(cfg.Logger.Level)
	if err != nil {
		level = LevelInfo
		fmt.Printf("Warning: invalid log level '%s', defaulting to INFO\n", cfg.Logger.Level)
	}

	logFilePath := createLogFilePath(logDir, "tg-antispam")
	rotatingLogger := createRotatingLogger(logFilePath, cfg)
	multiWriter := createMultiWriter(rotatingLogger)

	globalLogger = createCustomLogger(multiWriter, timezone, cfg.Logger.Format, cfg.Logger.TimeFormat, level)

	log.SetOutput(multiWriter)
	log.SetFlags(0)

	globalLogger.Infof("Logging initialized: writing to %s with timezone %s, minimum log level: %s",
		logFilePath, cfg.Logger.Timezone, level)
	return nil
}

// GetRotatingLogWriter returns a rotating log writer for custom loggers
func GetRotatingLogWriter(cfg *config.Config, prefix string) io.Writer {
	logFilePath := createLogFilePath(cfg.Logger.Directory, prefix)
	rotatingLogger := createRotatingLogger(logFilePath, cfg)
	return createMultiWriter(rotatingLogger)
}

func Debug(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(v...)
	} else {
		log.Print(v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debugf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Info(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(v...)
	} else {
		log.Print(v...)
	}
}

func Infof(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Infof(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Warning(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warning(v...)
	} else {
		log.Print(v...)
	}
}

func Warningf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warningf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Error(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(v...)
	} else {
		log.Print(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Errorf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Printf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Infof(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func Print(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(v...)
	} else {
		log.Print(v...)
	}
}

func Println(v ...interface{}) {
	if globalLogger != nil {
		message := fmt.Sprintln(v...)
		if len(message) > 0 && message[len(message)-1] == '\n' {
			message = message[:len(message)-1]
		}
		globalLogger.Info(message)
	} else {
		log.Println(v...)
	}
}

func Fatal(v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Fatal(v...)
	} else {
		log.Fatal(v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Fatalf(format, v...)
	} else {
		log.Fatalf(format, v...)
	}
}
