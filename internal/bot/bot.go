package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
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
	logger.Infof("Authorized on account %s", botUser.Username)

	// Set bot commands for menu in different languages
	setLocalizedCommands(ctx, bot)

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

// setLocalizedCommands sets bot commands in different languages
func setLocalizedCommands(ctx context.Context, bot *telego.Bot) {
	// Define command list
	commandKeys := []struct {
		Command string
		DescKey string
	}{
		{Command: "help", DescKey: "cmd_desc_help"},
		{Command: "settings", DescKey: "cmd_desc_settings"},
		{Command: "self_unban", DescKey: "cmd_desc_self_unban"},
		{Command: "language", DescKey: "cmd_desc_language"},
	}

	// Map of language codes to Telegram language codes
	langCodes := map[string]string{
		models.LangEnglish:            "en",
		models.LangSimplifiedChinese:  "zh",
		models.LangTraditionalChinese: "zh_TW",
	}

	// Set commands for each supported language
	for lang, telegramLang := range langCodes {
		var commands []telego.BotCommand

		for _, cmd := range commandKeys {
			commands = append(commands, telego.BotCommand{
				Command:     cmd.Command,
				Description: models.GetTranslation(lang, cmd.DescKey),
			})
		}

		params := &telego.SetMyCommandsParams{
			Commands:     commands,
			LanguageCode: telegramLang,
		}
		if err := setCommandsWithRetry(ctx, bot, 1, params); err != nil {
			logger.Infof("Warning: Failed to set bot commands for %s: %v", lang, err)
		}
	}

	// Set default commands (without language code) using Simplified Chinese
	var defaultCommands []telego.BotCommand
	for _, cmd := range commandKeys {
		defaultCommands = append(defaultCommands, telego.BotCommand{
			Command:     cmd.Command,
			Description: models.GetTranslation(models.LangSimplifiedChinese, cmd.DescKey),
		})
	}

	params := &telego.SetMyCommandsParams{
		Commands: defaultCommands,
	}
	if err := setCommandsWithRetry(ctx, bot, 3, params); err != nil {
		logger.Infof("Warning: Failed to set default bot commands: %v", err)
	}
}

// setCommandsWithRetry tries to set bot commands, retrying up to 3 times with a 5-minute delay on failure.
func setCommandsWithRetry(ctx context.Context, bot *telego.Bot, maxRetries int, params *telego.SetMyCommandsParams) error {
	const retryDelay = 5 * time.Minute
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = bot.SetMyCommands(ctx, params)
		if err == nil {
			return nil
		}
		if attempt < maxRetries {
			logger.Infof("Warning: Failed to set bot commands (attempt %d/%d): %v. Retrying in %v", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return fmt.Errorf("failed to set bot commands after %d attempts: %w", maxRetries, err)
}
