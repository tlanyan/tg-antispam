package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tg-antispam/internal/bot"
	"tg-antispam/internal/config"
	"tg-antispam/internal/handler"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/storage"
)

func main() {
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
		log.Println("Database connection established")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	botService, server, err := bot.Initialize(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	handler.Initialize(cfg)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)
	log.Println("HTTP server is ready, starting bot handler...")

	handler.SetupMessageHandlers(botService.Handler, botService.Bot)
	botService.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down...", sig)

	// Gracefully shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}
