package handler

import (
	"context"
	"fmt"
	"log"

	"tg-antispam/internal/models"

	"github.com/mymmrac/telego"
)

var (
	groupInfoManager = models.NewGroupInfoManager()
)

func GetGroupInfo(ctx context.Context, bot *telego.Bot, chatID int64) *models.GroupInfo {
	groupInfo := groupInfoManager.GetGroupInfo(chatID)
	if groupInfo == nil {
		groupInfo = &models.GroupInfo{
			GroupID: chatID,
			IsAdmin: false,
			AdminID: -1,
		}

		groupInfo.AdminID, groupInfo.IsAdmin = GetBotPromoterID(ctx, bot, chatID)
		groupInfo.GroupName, groupInfo.GroupLink = GetGroupName(ctx, bot, chatID)
		log.Printf("Group info: %+v", groupInfo)
		groupInfoManager.AddGroupInfo(groupInfo)
	}
	return groupInfo
}

func GetBotPromoterID(ctx context.Context, bot *telego.Bot, chatID int64) (int64, bool) {
	// Need a bot instance to make API calls
	newBot, err := telego.NewBot(globalConfig.Bot.Token)
	if err != nil {
		log.Printf("Error creating temporary bot for admin check: %v", err)
		return 0, false
	}
	defer newBot.Close(ctx)

	// Get all administrators in the chat
	admins, err := newBot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		log.Printf("Error getting chat administrators: %v", err)
		return 0, false
	}

	botID := bot.ID()
	// First check: First determine if bot is an admin
	botIsAdmin := false
	for _, admin := range admins {
		if admin.MemberUser().ID == botID {
			botIsAdmin = true
			break
		}
	}

	if !botIsAdmin {
		return 0, false
	}

	// Find the chat creator if available (they definitely have promotion rights)
	for _, admin := range admins {
		if admin.MemberStatus() == telego.MemberStatusCreator {
			if creator, ok := admin.(*telego.ChatMemberOwner); ok {
				return creator.User.ID, true
			}
		}
	}

	// Second: Find admins who can promote members (one of them promoted the bot)
	var candidateAdmins []telego.ChatMemberAdministrator
	for _, admin := range admins {
		// Skip the bot itself
		if admin.MemberUser().ID == botID {
			continue
		}

		// Check if this admin can promote members
		if admin.MemberStatus() == telego.MemberStatusAdministrator {
			if adminMember, ok := admin.(*telego.ChatMemberAdministrator); ok {
				if adminMember.CanPromoteMembers {
					candidateAdmins = append(candidateAdmins, *adminMember)
				}
			}
		}
	}

	// If we found admins who can promote, return the first one
	// This is our best guess at who promoted the bot
	if len(candidateAdmins) > 0 {
		return candidateAdmins[0].User.ID, true
	}

	log.Printf("Could not find bot promoter in chat %d", chatID)
	return 0, false
}

// GetLinkedGroupName gets a linked HTML representation of the group name with caching
func GetGroupName(ctx context.Context, bot *telego.Bot, chatID int64) (string, string) {
	// Cache miss, fetch from API
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		log.Printf("Error getting chat info: %v", err)
		return "", ""
	}

	var groupLink string
	if chatInfo.Username != "" {
		groupLink = fmt.Sprintf("https://t.me/%s", chatInfo.Username)
	} else {
		// For private groups, convert the chat ID to work with t.me links
		// Telegram requires removing the -100 prefix from supergroup IDs for links
		groupIDForLink := chatID
		if groupIDForLink < -1000000000000 {
			// Extract the actual ID from the negative number (skip the -100 prefix)
			groupIDForLink = -groupIDForLink - 1000000000000
		}
		groupLink = fmt.Sprintf("https://t.me/c/%d", groupIDForLink)
	}

	return chatInfo.Title, groupLink
}
