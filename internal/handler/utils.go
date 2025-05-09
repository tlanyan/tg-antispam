package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
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

func getGroupAndUserID(data string) (int64, int64, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("invalid data format: %s", data)
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid group ID: %v", err)
	}

	userID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid user ID: %v", err)
	}

	return groupID, userID, nil
}

func getLinkedUserName(ctx *th.Context, bot *telego.Bot, userID int64) (string, error) {
	// Get user information
	userInfo, err := bot.GetChat(ctx.Context(), &telego.GetChatParams{
		ChatID: telego.ChatID{ID: userID},
	})
	if err != nil {
		return "", fmt.Errorf("Error getting user info: %v", err)
	}

	userName := userInfo.FirstName
	if userInfo.LastName != "" {
		userName += " " + userInfo.LastName
	}

	// Create user link
	userLink := fmt.Sprintf("tg://user?id=%d", userID)
	return fmt.Sprintf("<a href=\"%s\">%s</a>", userLink, userName), nil
}
