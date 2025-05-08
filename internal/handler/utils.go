package handler

import (
	"context"

	"github.com/mymmrac/telego"
)

// isUserAdmin checks if a user is an admin in a chat
func isUserAdmin(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) (bool, error) {
	admins, err := bot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		return false, err
	}

	for _, admin := range admins {
		if admin.MemberUser().ID == userID {
			return true, nil
		}
	}

	return false, nil
}

// getBotUsername retrieves the bot's username
func getBotUsername(ctx context.Context, bot *telego.Bot) (string, error) {
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		return "", err
	}
	return botUser.Username, nil
}
