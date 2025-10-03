package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tg-antispam/internal/bot"
	"tg-antispam/internal/config"
	"tg-antispam/internal/crash"
	"tg-antispam/internal/handler"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/service"
	"tg-antispam/internal/storage"

	"github.com/mymmrac/telego"
)

func main() {
	// 设置崩溃处理器，确保在任何 panic 时都能记录堆栈信息
	defer crash.RecoverWithStackAndExit("main")

	// 设置全局崩溃处理
	crash.SetupCrashHandler()

	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := logger.Setup(cfg); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	if cfg.Database.Enabled {
		if err := storage.Initialize(cfg); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		service.InitRepositories()
		logger.Info("Database connection established and repositories initialized")
	} else {
		logger.Info("Database support is disabled. Repositories will not be initialized.")
	}

	// Start the cache cleanup goroutine regardless of DB status
    service.StartCacheCleanup()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	botService, server, err := bot.Initialize(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	handler.Initialize(cfg)

	crash.SafeGoroutine("http-server", func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("HTTP server error: %v", err)
		}
	})

	// Give server time to start
	time.Sleep(500 * time.Millisecond)
	log.Println("HTTP server is ready, starting bot handler...")

	handler.SetupMessageHandlers(botService.Handler, botService.Bot)
	handlePendingDeletions(botService, cfg)
	botService.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)

	// Wait for signal
	sig := <-sigChan
	logger.Infof("Received signal: %v, shutting down...", sig)

	logger.Info("Waiting for message handlers to complete...")
	// 等待所有消息处理器完成
	done := make(chan struct{})
	go func() {
		handler.WaitForHandlers()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All message handlers completed")
	case <-time.After(30 * time.Second):
		logger.Warning("Timeout waiting for message handlers, proceeding with shutdown")
	}

	logger.Info("attempting to clear in-memory pending deletions...")
	if botService.Bot != nil { // Ensure bot service is available
		handler.DeleteAllPendingInMemoryMessages(botService.Bot)
	} else {
		logger.Warning("Bot service not available for shutdown cleanup of in-memory messages.")
	}

	// Gracefully shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}

func handlePendingDeletions(botService *bot.BotService, cfg *config.Config) {
	// After bot starts, load and process pending deletions if DB is enabled
	if cfg.Database.Enabled {
		logger.Info("Loading pending message deletions from database...")
		pendingMsgs, err := service.GetAllPendingMsgs()
		if err != nil {
			logger.Errorf("Error loading pending deletions: %v", err)
		} else {
			logger.Infof("Found %d pending message deletions to process.", len(pendingMsgs))
			for _, msg := range pendingMsgs {
				msgCopy := msg // 创建副本避免闭包问题
				crash.SafeGoroutine(fmt.Sprintf("pending-deletion-%d-%d", msgCopy.ChatID, msgCopy.MessageID), func() {
					durationUntilDelete := time.Until(msgCopy.CreatedAt.Add(3 * time.Minute))
					if durationUntilDelete < 0 {
						durationUntilDelete = 0 // Delete immediately if past due
					}

					logger.Infof("Rescheduling deletion for message %d in chat %d in %v", msgCopy.MessageID, msgCopy.ChatID, durationUntilDelete)
					time.Sleep(durationUntilDelete)

					botService.Bot.DeleteMessage(context.Background(), &telego.DeleteMessageParams{
						ChatID:    telego.ChatID{ID: msgCopy.ChatID},
						MessageID: msgCopy.MessageID,
					})

					// Remove from DB after attempting deletion
					if err = service.RemovePendingMsg(msgCopy.ChatID, msgCopy.MessageID); err != nil {
						logger.Warningf("Error removing pending deletion from DB for chat %d, message %d: %v", msgCopy.ChatID, msgCopy.MessageID, err)
					}
				})
			}
		}
	}
}
