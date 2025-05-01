package bot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// Config represents bot configuration
type Config struct {
	Token        string
	WebhookPoint string
	ListenPort   string
	CertFile     string
	KeyFile      string
}

// BotService represents the Telegram bot service
type BotService struct {
	Bot     *telego.Bot
	Handler *th.BotHandler
}

// Start starts the bot handler
func (b *BotService) Start() {
	b.Handler.Start()
}

// Stop stops the bot handler
func (b *BotService) Stop() {
	b.Handler.Stop()
}

// Initialize initializes the bot and webhook
func Initialize(ctx context.Context, config Config) (*BotService, *WebhookServer, error) {
	// Validate configuration
	if config.Token == "" {
		return nil, nil, fmt.Errorf("bot token is required")
	}

	if config.WebhookPoint == "" {
		return nil, nil, fmt.Errorf("webhook point is required")
	}

	// Set default values
	listenPort := config.ListenPort
	if listenPort == "" {
		listenPort = "8443" // Default listen port
		log.Printf("Using default listen port: %s", listenPort)
	}

	// Parse URL to get path component
	parsedURL, err := url.Parse(config.WebhookPoint)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid webhook point: %w", err)
	}

	webhookPath := parsedURL.Path
	if webhookPath == "" {
		webhookPath = "/webhook"
		log.Printf("No path specified in webhook point, using default path: %s", webhookPath)
	}

	webhookListen := "0.0.0.0:" + listenPort

	// Validate HTTPS setup
	if (config.CertFile == "" || config.KeyFile == "") && !strings.HasPrefix(config.WebhookPoint, "https://") {
		return nil, nil, fmt.Errorf("HTTPS configuration required: set CERT_FILE and KEY_FILE env vars or use a HTTPS proxy")
	}

	// Initialize bot
	bot, err := telego.NewBot(config.Token, telego.WithDefaultDebugLogger())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize bot: %w", err)
	}

	// Get bot info
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bot info: %w", err)
	}
	log.Printf("Authorized on account %s", botUser.Username)

	// Delete any existing webhook
	err = bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete existing webhook: %w", err)
	}

	// Set fixed secret token instead of random generation
	secretToken := "secure_webhook_token_" + config.Token[len(config.Token)-6:]

	// Set up webhook handler
	bh, server, err := SetupWebhook(ctx, bot, config.WebhookPoint, webhookPath, webhookListen, secretToken, config.CertFile, config.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup webhook: %w", err)
	}

	return &BotService{
		Bot:     bot,
		Handler: bh,
	}, server, nil
}
