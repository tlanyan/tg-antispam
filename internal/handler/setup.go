package handler

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/service"
)

// Global configuration shared by all handler files
var globalConfig *config.Config

// Initialize initializes the handler with configuration
func Initialize(cfg *config.Config) {
	globalConfig = cfg
	// Also initialize the service layer
	service.Initialize(cfg)
}

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// Initialize group repository if database is enabled
	service.InitGroupRepository()

	// Register commands
	RegisterCommands(bh, bot)

	// Skip messages from the bot itself
	botID := bot.ID()

	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return handleIncomingMessage(ctx, bot, message)
	})

	// Handle chat member updates
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		return handleChatMemberUpdate(ctx, bot, update, botID)
	}, th.AnyChatMember())

	// Handle MyChatMember updates (when a user changes the bot's status)
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		return handleMyChatMemberUpdate(ctx, bot, update)
	}, th.AnyMyChatMember())

	// Handle callback queries for unban button
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		return HandleCallbackQuery(ctx, bot, query)
	})
}
