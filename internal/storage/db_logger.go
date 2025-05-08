package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	customlogger "tg-antispam/internal/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// CustomGormLogger 是我们自定义的GORM日志适配器
// 它实现了gorm/logger.Interface接口，但使用我们的自定义logger
type CustomGormLogger struct {
	LogLevel                  logger.LogLevel
	SlowThreshold             time.Duration
	SkipCallerLookup          bool
	IgnoreRecordNotFoundError bool
}

// NewCustomGormLogger 创建一个新的GORM日志适配器
func NewCustomGormLogger(level string) logger.Interface {
	var logLevel logger.LogLevel

	// 将我们的日志级别映射到GORM的日志级别
	switch level {
	case "DEBUG":
		logLevel = logger.Info // GORM的Debug太详细，使用Info级别更合适
	case "INFO":
		logLevel = logger.Info
	case "WARNING", "ERROR":
		logLevel = logger.Warn
	case "FATAL":
		logLevel = logger.Error
	default:
		logLevel = logger.Info
	}

	return &CustomGormLogger{
		LogLevel:                  logLevel,
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode 设置日志级别
func (l *CustomGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 输出信息级别日志
func (l *CustomGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		customlogger.Infof(msg, data...)
	}
}

// Warn 输出警告级别日志
func (l *CustomGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		customlogger.Warningf(msg, data...)
	}
}

// Error 输出错误级别日志
func (l *CustomGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		customlogger.Errorf(msg, data...)
	}
}

// Trace 记录SQL执行情况
func (l *CustomGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 获取调用位置
	var source string
	if !l.SkipCallerLookup {
		source = utils.FileWithLineNum()
	}

	// 根据执行结果决定日志级别
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		if source != "" {
			customlogger.Errorf("[%.3fms] [%s] %s; error=%v", float64(elapsed.Nanoseconds())/1e6, source, sql, err)
		} else {
			customlogger.Errorf("[%.3fms] %s; error=%v", float64(elapsed.Nanoseconds())/1e6, sql, err)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if source != "" {
			customlogger.Warningf("[%.3fms] [%s] %s; %s, rows=%v", float64(elapsed.Nanoseconds())/1e6, source, sql, slowLog, rows)
		} else {
			customlogger.Warningf("[%.3fms] %s; %s, rows=%v", float64(elapsed.Nanoseconds())/1e6, sql, slowLog, rows)
		}
	case l.LogLevel == logger.Info:
		if source != "" {
			customlogger.Debugf("[%.3fms] [%s] %s; rows=%v", float64(elapsed.Nanoseconds())/1e6, source, sql, rows)
		} else {
			customlogger.Debugf("[%.3fms] %s; rows=%v", float64(elapsed.Nanoseconds())/1e6, sql, rows)
		}
	}
}
