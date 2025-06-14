# 崩溃处理改进说明

## 概述

为了解决程序在 Docker 容器中频繁崩溃重启的问题，我们添加了全面的崩溃处理和堆栈信息记录功能。

## 改进内容

### 1. 新增崩溃处理模块 (`internal/crash/handler.go`)

- **RecoverWithStack**: 通用的 panic 恢复函数，记录详细的崩溃信息和堆栈
- **RecoverWithStackAndExit**: 主程序专用的 panic 恢复函数，记录信息后优雅退出
- **SafeGoroutine**: 启动带有 panic 恢复的安全 goroutine
- **logRuntimeInfo**: 记录详细的运行时信息，包括内存使用、goroutine 数量等

### 2. 主程序改进 (`cmd/main/main.go`)

- 在 main 函数开始处添加了全局崩溃处理器
- 将 HTTP 服务器启动改为使用 SafeGoroutine
- 将待删除消息处理改为使用 SafeGoroutine

### 3. Handler 模块改进

#### `internal/handler/commands.go`
- 私聊警告消息清理 goroutine 改为使用 SafeGoroutine

#### `internal/handler/restrictions.go`
- 警告消息清理 goroutine 改为使用 SafeGoroutine

#### `internal/handler/message_handlers.go`
- 待处理用户检查 goroutine 改为使用 SafeGoroutine
- 用户限制处理 goroutine 改为使用 SafeGoroutine

#### `internal/handler/callbacks.go`
- 待删除消息清理 goroutine 改为使用 SafeGoroutine

### 4. Docker 配置改进 (`Dockerfile`)

- 添加了 `ENV GOTRACEBACK=all` 环境变量，确保输出完整的堆栈跟踪信息

## 功能特性

### 崩溃信息记录
- **双重输出**: 同时记录到日志文件和标准错误输出，确保容器日志能捕获到崩溃信息
- **详细堆栈**: 完整的 Go 堆栈跟踪信息
- **运行时信息**: 包括内存使用、goroutine 数量、GC 统计等系统信息
- **时间戳**: 精确的崩溃发生时间

### 安全 Goroutine
- **自动恢复**: 所有 goroutine 都有 panic 恢复机制
- **命名标识**: 每个 goroutine 都有唯一的名称标识，便于调试
- **闭包安全**: 正确处理了变量闭包问题，避免竞态条件

### 优雅退出
- **非零退出码**: 崩溃时以状态码 1 退出，便于容器编排系统检测异常
- **延迟退出**: 给日志系统充足时间写入崩溃信息
- **信号处理**: 保持原有的信号处理机制

## 使用效果

1. **崩溃可见性**: 程序崩溃时会在容器日志中看到详细的错误信息和堆栈
2. **问题定位**: 通过堆栈信息和运行时统计，可以快速定位崩溃原因
3. **服务稳定性**: 单个 goroutine 的崩溃不会影响整个程序的运行
4. **容器监控**: 容器编排系统可以正确检测到程序异常并采取相应措施

## 日志示例

崩溃时的日志输出示例：
```
[FATAL PANIC] 2024-01-20 15:04:05 - main: runtime error: invalid memory address or nil pointer dereference
Stack trace:
goroutine 1 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x5e
main.main.func1()
	/app/cmd/main/main.go:25 +0x12c
panic({0x1043e40?, 0x1157d10?})
	/usr/local/go/src/runtime/panic.go:770 +0x132
...

Runtime Information:
- Go version: go1.24.1
- Go root: /usr/local/go
- Number of CPUs: 4
- Number of goroutines: 12
- Memory stats:
  - Heap allocated: 2048 KB
  - Heap in use: 4096 KB
  - Stack in use: 1024 KB
  - Next GC: 8192 KB
  - Num GC: 5
```

## 部署建议

1. **日志收集**: 建议配置容器日志收集系统，确保崩溃信息能被持久化保存
2. **监控告警**: 配置容器重启监控，及时发现和处理崩溃问题
3. **资源监控**: 结合运行时信息，监控内存和 goroutine 泄漏情况

## 注意事项

- 崩溃处理模块本身使用了最小依赖，避免在错误处理过程中引入额外的崩溃风险
- 所有 goroutine 都正确处理了变量闭包，避免竞态条件
- 保持了原有程序的功能逻辑不变，只是增加了安全性包装