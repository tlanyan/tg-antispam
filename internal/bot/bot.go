package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
)

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
func Initialize(ctx context.Context, cfg *config.Config) (*BotService, *WebhookServer, error) {
	// Validate configuration
	if cfg.Bot.Token == "" {
		return nil, nil, fmt.Errorf("bot token is required")
	}

	// Initialize bot
	bot, err := telego.NewBot(cfg.Bot.Token, telego.WithDefaultDebugLogger())
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

	// Set fixed secret token or generate one based on bot token
	secretToken := "secure_webhook_token_" + cfg.Bot.Token[len(cfg.Bot.Token)-6:]

	// Set up webhook handler
	bh, server, err := SetupWebhook(ctx, bot, cfg.Bot.Webhook.Endpoint, cfg.Bot.Webhook.ListenPort, cfg.Bot.Webhook.DebugPath, secretToken, cfg.Bot.Webhook.CertFile, cfg.Bot.Webhook.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup webhook: %w", err)
	}

	return &BotService{
		Bot:     bot,
		Handler: bh,
	}, server, nil
}
