package handler

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"tg-antispam/internal/logger"
)

// 统计信息
var (
	totalMessagesProcessed int64
	totalChatMemberUpdates int64
	totalCallbackQueries   int64
	totalErrors            int64
	totalTimeouts          int64
	startTime              = time.Now()
)

// incrementCounter 安全地增加计数器
func incrementCounter(counter *int64) {
	atomic.AddInt64(counter, 1)
}

// GetProcessingStats 获取处理统计信息
func GetProcessingStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	activeHandlers := GetActiveHandlersCount()
	uptime := time.Since(startTime)

	return map[string]interface{}{
		"uptime_seconds":            int64(uptime.Seconds()),
		"total_messages":            atomic.LoadInt64(&totalMessagesProcessed),
		"total_chat_member_updates": atomic.LoadInt64(&totalChatMemberUpdates),
		"total_callback_queries":    atomic.LoadInt64(&totalCallbackQueries),
		"total_errors":              atomic.LoadInt64(&totalErrors),
		"total_timeouts":            atomic.LoadInt64(&totalTimeouts),
		"active_handlers":           activeHandlers,
		"max_concurrent_messages":   cap(messageProcessingSemaphore),
		"memory_usage_mb":           bToMb(m.Alloc),
		"total_alloc_mb":            bToMb(m.TotalAlloc),
		"sys_memory_mb":             bToMb(m.Sys),
		"gc_runs":                   m.NumGC,
		"goroutines":                runtime.NumGoroutine(),
	}
}

// LogProcessingStats 定期记录处理统计信息
func LogProcessingStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := GetProcessingStats()
		logger.Infof("Processing stats: %+v", stats)

		// 如果活跃处理器数量过多，记录警告
		if activeHandlers := stats["active_handlers"].(int); activeHandlers > 80 {
			logger.Warningf("High number of active handlers: %d", activeHandlers)
		}

		// 如果错误率过高，记录警告
		totalMessages := stats["total_messages"].(int64)
		totalErrors := stats["total_errors"].(int64)
		if totalMessages > 0 && float64(totalErrors)/float64(totalMessages) > 0.1 {
			logger.Warningf("High error rate: %.2f%% (%d errors out of %d messages)",
				float64(totalErrors)/float64(totalMessages)*100, totalErrors, totalMessages)
		}
	}
}

// StartStatusMonitoring 启动状态监控
func StartStatusMonitoring() {
	go LogProcessingStats()
}

// bToMb 将字节转换为MB
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// GetDetailedStatus 获取详细状态信息（用于调试）
func GetDetailedStatus() string {
	stats := GetProcessingStats()
	return fmt.Sprintf(`
=== TG-AntiSpam Processing Status ===
Uptime: %d seconds
Messages Processed: %d
Chat Member Updates: %d
Callback Queries: %d
Errors: %d
Timeouts: %d
Active Handlers: %d/%d
Memory Usage: %d MB
Total Allocated: %d MB
System Memory: %d MB
GC Runs: %d
Goroutines: %d
=====================================`,
		stats["uptime_seconds"],
		stats["total_messages"],
		stats["total_chat_member_updates"],
		stats["total_callback_queries"],
		stats["total_errors"],
		stats["total_timeouts"],
		stats["active_handlers"],
		stats["max_concurrent_messages"],
		stats["memory_usage_mb"],
		stats["total_alloc_mb"],
		stats["sys_memory_mb"],
		stats["gc_runs"],
		stats["goroutines"],
	)
}
