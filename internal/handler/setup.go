package handler

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/service"
)

var globalConfig *config.Config

func Initialize(cfg *config.Config) {
	globalConfig = cfg
	service.Initialize(cfg)
}

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	service.InitRepositories()

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		ok, err := RegisterCommands(ctx, bot, message)
		if ok {
			return err
		}

		return handleIncomingMessage(ctx, bot, message)
	})

	bh.HandleChannelPost(func(ctx *th.Context, message telego.Message) error {
		return handleIncomingMessage(ctx, bot, message)
	})

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		return handleChatMemberUpdate(ctx, bot, update)
	}, th.AnyChatMember())

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		return handleMyChatMemberUpdate(ctx, bot, update)
	}, th.AnyMyChatMember())

	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		return HandleCallbackQuery(ctx, bot, query)
	})
}
