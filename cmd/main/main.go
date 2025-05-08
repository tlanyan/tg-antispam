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
	// Define command line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging first
	if err := logger.Setup(cfg); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	// Initialize database if enabled
	if cfg.Database.Enabled {
		if err := storage.Initialize(cfg); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		log.Println("Database connection established")
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize bot with configuration
	botService, server, err := bot.Initialize(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Initialize handler with configuration
	handler.Initialize(cfg)

	// Start HTTP server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)
	log.Println("HTTP server is ready, starting bot handler...")

	// Setup and start message handlers
	handler.SetupMessageHandlers(botService.Handler, botService.Bot)
	botService.Start()

	// Create a channel for receiving OS signals
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
