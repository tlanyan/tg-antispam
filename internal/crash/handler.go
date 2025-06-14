package crash

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"tg-antispam/internal/logger"
)

// RecoverWithStack 是一个通用的 panic 恢复函数，会记录详细的堆栈信息
func RecoverWithStack(moduleName string) {
	if r := recover(); r != nil {
		stack := debug.Stack()

		// 记录崩溃信息到日志
		logger.Errorf("PANIC in %s: %v", moduleName, r)
		logger.Errorf("Stack trace:\n%s", string(stack))

		// 同时输出到标准错误，确保在容器日志中能看到
		fmt.Fprintf(os.Stderr, "[PANIC] %s - %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), moduleName, r)
		fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", string(stack))

		// 记录运行时信息
		logRuntimeInfo()
	}
}

// RecoverWithStackAndExit 用于主程序的 panic 恢复，会记录信息后退出
func RecoverWithStackAndExit(moduleName string) {
	if r := recover(); r != nil {
		stack := debug.Stack()

		// 记录崩溃信息到日志
		logger.Errorf("FATAL PANIC in %s: %v", moduleName, r)
		logger.Errorf("Stack trace:\n%s", string(stack))

		// 同时输出到标准错误，确保在容器日志中能看到
		fmt.Fprintf(os.Stderr, "[FATAL PANIC] %s - %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), moduleName, r)
		fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", string(stack))

		// 记录运行时信息
		logRuntimeInfo()

		// 给日志系统一些时间写入文件
		time.Sleep(1 * time.Second)

		// 以非零状态码退出，这样容器编排系统可以检测到异常
		os.Exit(1)
	}
}

// SafeGoroutine 启动一个带有 panic 恢复的 goroutine
func SafeGoroutine(name string, fn func()) {
	go func() {
		defer RecoverWithStack(fmt.Sprintf("goroutine-%s", name))
		fn()
	}()
}

// logRuntimeInfo 记录运行时信息，帮助调试
func logRuntimeInfo() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := fmt.Sprintf(`
Runtime Information:
- Go version: %s
- Go root: %s
- Number of CPUs: %d
- Number of goroutines: %d
- Memory stats:
  - Heap allocated: %d KB
  - Heap in use: %d KB
  - Stack in use: %d KB
  - Next GC: %d KB
  - Num GC: %d
`,
		runtime.Version(),
		runtime.GOROOT(),
		runtime.NumCPU(),
		runtime.NumGoroutine(),
		bToKb(m.HeapAlloc),
		bToKb(m.HeapInuse),
		bToKb(m.StackInuse),
		bToKb(m.NextGC),
		m.NumGC,
	)

	logger.Error(info)
	fmt.Fprint(os.Stderr, info)
}

// bToKb 将字节转换为KB
func bToKb(b uint64) uint64 {
	return b / 1024
}

// SetupCrashHandler 设置全局的崩溃处理器
func SetupCrashHandler() {
	// 设置在程序即将退出时的清理函数
	debug.SetPanicOnFault(true)
}