package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mymmrac/telego"
)

func main() {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	// Get webhook configuration from environment variables
	webhookPoint := os.Getenv("WEBHOOK_POINT")
	if webhookPoint == "" {
		log.Fatal("WEBHOOK_POINT environment variable not set (e.g. https://example.com/webhook)")
	}

	// Parse URL to get path component
	parsedURL, err := url.Parse(webhookPoint)
	if err != nil {
		log.Fatalf("Invalid WEBHOOK_POINT: %v", err)
	}
	webhookPath := parsedURL.Path
	if webhookPath == "" {
		webhookPath = "/webhook"
		log.Printf("No path specified in WEBHOOK_POINT, using default path: %s", webhookPath)
	}

	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8443" // Default listen port
		log.Printf("Using default listen port: %s", listenPort)
	}

	webhookListen := "0.0.0.0:" + listenPort
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	if (certFile == "" || keyFile == "") && !strings.HasPrefix(webhookPoint, "https://") {
		log.Fatal("HTTPS configuration required: Set CERT_FILE and KEY_FILE env vars or use a HTTPS proxy")
	}

	// Initialize bot
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Get bot info
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatalf("Failed to get bot info: %v", err)
	}
	log.Printf("Authorized on account %s", botUser.Username)

	// Delete any existing webhook
	err = bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{})
	if err != nil {
		log.Fatalf("Failed to delete existing webhook: %v", err)
	}

	// Set fixed secret token instead of random generation
	secretToken := "secure_webhook_token_" + botToken[len(botToken)-6:]

	// Set up webhook handler
	bh, server, err := SetupWebhook(ctx, bot, webhookPoint, webhookPath, webhookListen, secretToken, certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to setup webhook: %v", err)
	}
	defer bh.Stop()

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", webhookListen)
		log.Printf("Bot webhook path: %s, Debug path: /debug", webhookPath)

		var err error
		if certFile != "" && keyFile != "" {
			log.Printf("Using TLS with cert: %s, key: %s", certFile, keyFile)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("WARNING: Running without TLS. Make sure you have a HTTPS proxy in front of this server")
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)
	log.Println("HTTP server is ready, starting bot handler...")

	// Start bot handler after server is ready
	bh.Start()

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