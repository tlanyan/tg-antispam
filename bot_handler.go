package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

var (
	// Compiled regular expressions
	emojiRegex  = regexp.MustCompile(`[\x{1F600}-\x{1F64F}|\x{1F300}-\x{1F5FF}|\x{1F680}-\x{1F6FF}|\x{1F700}-\x{1F77F}|\x{1F780}-\x{1F7FF}|\x{1F800}-\x{1F8FF}|\x{1F900}-\x{1F9FF}|\x{1FA00}-\x{1FA6F}|\x{1FA70}-\x{1FAFF}|\x{2600}-\x{26FF}|\x{2700}-\x{27BF}]`)
	tgLinkRegex = regexp.MustCompile(`t\.me`)
)

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// Skip messages from the bot itself
	botInfo, err := bot.GetMe(context.Background())
	if err != nil {
		log.Printf("Error getting bot info: %v", err)
	}

	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		log.Printf("Processing message: %+v", message)

		// only restrict user when they join the group
		return nil
		// Skip messages from the bot itself
		// if message.From != nil && botInfo != nil && message.From.ID == botInfo.ID {
		// 	log.Printf("Skipping message from the bot itself")
		// 	return nil
		// }

		// if message.From != nil && message.From.IsPremium {
		// 	if message.From.IsBot {
		// 		log.Printf("Skipping bot: %s", message.From.FirstName)
		// 		return nil
		// 	}
		// 	log.Printf("Found premium user: %s", message.From.FirstName)

		// 	// Check if user has permission to send messages first
		// 	hasPermission, err := UserCanSendMessages(ctx.Context(), bot, message.Chat.ID, message.From.ID)
		// 	if err != nil {
		// 		log.Printf("Error checking user permissions: %v", err)
		// 		return nil
		// 	}

		// 	// Only restrict if they have permission (not already restricted)
		// 	if hasPermission {
		// 		bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
		// 			ChatID:    telego.ChatID{ID: message.Chat.ID},
		// 			MessageID: message.MessageID,
		// 		})
		// 		RestrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
		// 		SendWarning(ctx.Context(), bot, message.Chat.ID, *message.From)
		// 	} else {
		// 		log.Printf("User %s is already restricted, skipping", message.From.FirstName)
		// 	}
		// 	return nil
		// }

		// return nil
	})

	// Handle chat member updates
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// Process ChatMember updates (when users join chat or change status)
		if update.ChatMember != nil {
			log.Printf("Chat member update: %+v", update.ChatMember)
			log.Printf("new Chat member: %+v", update.ChatMember.NewChatMember)

			// Skip updates related to the bot itself
			if botInfo != nil && update.ChatMember.From.ID == botInfo.ID {
				log.Printf("Skipping chat member update from the bot itself")
				return nil
			}

			if update.ChatMember.NewChatMember.MemberIsMember() {
				newMember := update.ChatMember.NewChatMember.MemberUser()
				// Skip bots
				if newMember.IsBot {
					log.Printf("Skipping bot: %s", newMember.FirstName)
					return nil
				}

				// Check if user has permission to send messages first
				hasPermission, err := UserCanSendMessages(ctx.Context(), bot, update.ChatMember.Chat.ID, newMember.ID)
				if err != nil {
					log.Printf("Error checking user permissions: %v", err)
					return nil
				}

				if !hasPermission {
					log.Printf("User %s is already restricted, skipping", newMember.FirstName)
					return nil
				}

				// Check if user should be restricted
				if ShouldRestrictUser(ctx, bot, newMember) {
					log.Printf("Restricting user: %s", newMember.FirstName)
					RestrictUser(ctx.Context(), bot, update.ChatMember.Chat.ID, newMember.ID)
					SendWarning(ctx.Context(), bot, update.ChatMember.Chat.ID, newMember)
				}
			}
		}
		return nil
	}, th.AnyChatMember())

	// Handle callback queries for unban button
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		log.Printf("Full callback query object: %+v", query)

		// Check if it's an unban request
		if strings.HasPrefix(query.Data, "unban:") {
			log.Printf("Processing unban request with data: %s", query.Data)
			// Extract chat ID and user ID from callback data
			parts := strings.Split(query.Data, ":")
			if len(parts) != 3 {
				return nil
			}

			chatID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				log.Printf("Error parsing chat ID: %v", err)
				return nil
			}

			userID, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				log.Printf("Error parsing user ID: %v", err)
				return nil
			}

			// Unban the user
			UnrestrictUser(ctx.Context(), bot, chatID, userID)

			// Get user information
			userInfo, err := bot.GetChat(ctx.Context(), &telego.GetChatParams{
				ChatID: telego.ChatID{ID: userID},
			})
			if err != nil {
				log.Printf("Error getting user info: %v", err)
				return nil
			}

			userName := userInfo.FirstName
			if userInfo.LastName != "" {
				userName += " " + userInfo.LastName
			}

			// Create user link
			userLink := fmt.Sprintf("tg://user?id=%d", userID)
			linkedUserName := fmt.Sprintf("<a href=\"%s\">%s</a>", userLink, userName)

			// Update the message
			messageText := fmt.Sprintf("✅ <b>用户已解封</b>\n"+
				"用户 %s 已被解除限制，现在可以正常发言。", linkedUserName)

			// Check if we have a message to edit
			if query.Message != nil {
				// Access message fields correctly for MaybeInaccessibleMessage type
				chatID := query.Message.GetChat().ID
				messageID := query.Message.GetMessageID()

				bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
					ChatID:      telego.ChatID{ID: chatID},
					MessageID:   messageID,
					Text:        messageText,
					ParseMode:   "HTML",
					ReplyMarkup: nil, // Remove the button
				})
			}

			// Answer the callback query
			bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "✅ 用户已成功解封",
			})
		}

		return nil
	})
}

// ShouldRestrictUser checks if a user should be restricted based on their name and username
func ShouldRestrictUser(ctx context.Context, bot *telego.Bot, user telego.User) bool {
	// Check for emoji in name
	if HasEmoji(user.FirstName) || HasEmoji(user.LastName) {
		return true
	}

	if user.IsPremium {
		return true
	}

	// Check for random username
	if IsRandomUsername(user.Username) {
		return true
	}

	// Check for t.me links in bio
	if HasTelegramLinksInBio(ctx, bot, user.ID) {
		return true
	}

	return false
}

// HasTelegramLinksInBio checks if a user's bio contains t.me links
func HasTelegramLinksInBio(ctx context.Context, bot *telego.Bot, userID int64) bool {
	// Get full user info to access bio
	userChat, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: userID}, // User's private chat
	})

	if err != nil {
		log.Printf("Error getting user info for ID %d: %v", userID, err)
		return false
	}

	// Check if we can access the user's bio
	if userChat.Bio != "" {
		return tgLinkRegex.MatchString(userChat.Bio) || strings.Contains(strings.ToLower(userChat.Bio), "t.me")
	}

	return false
}

// HasEmoji checks if a string contains emoji characters
func HasEmoji(s string) bool {
	if s == "" {
		return false
	}
	return emojiRegex.MatchString(s)
}

// IsRandomUsername checks if a username appears to be a random string
func IsRandomUsername(username string) bool {
	if username == "" {
		return false
	}

	// Check for 5 consecutive consonants
	consonantsRegex := regexp.MustCompile(`[bcdfghjklmnpqrstvwxyz]{5}`)
	if consonantsRegex.MatchString(strings.ToLower(username)) {
		return true
	}

	// Check for 7 consecutive digits
	digitsRegex := regexp.MustCompile(`\d{7}`)
	if digitsRegex.MatchString(username) {
		return true
	}

	return false
}

// RestrictUser restricts a user's permissions in a chat
func RestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	// Create chat permissions that restrict sending messages and media
	canSendMessages := false
	canSendMedia := false
	canSendPolls := false
	canSendOther := false
	canAddWebPreview := false

	permissions := telego.ChatPermissions{
		CanSendMessages:       &canSendMessages,
		CanSendAudios:         &canSendMedia,
		CanSendDocuments:      &canSendMedia,
		CanSendPhotos:         &canSendMedia,
		CanSendVideos:         &canSendMedia,
		CanSendVideoNotes:     &canSendMedia,
		CanSendVoiceNotes:     &canSendMedia,
		CanSendPolls:          &canSendPolls,
		CanSendOtherMessages:  &canSendOther,
		CanAddWebPagePreviews: &canAddWebPreview,
	}

	// Create restriction config
	params := telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: chatID},
		UserID:      userID,
		Permissions: permissions,
		UntilDate:   0, // 0 means restrict indefinitely
	}

	// Apply restriction
	err := bot.RestrictChatMember(ctx, &params)
	if err != nil {
		log.Printf("Error restricting user %d: %v", userID, err)
	} else {
		log.Printf("Successfully restricted user %d in chat %d", userID, chatID)
	}
}

// SendWarning sends a warning message about the restricted user to the specified admin
func SendWarning(ctx context.Context, bot *telego.Bot, chatID int64, user telego.User) {
	// Get admin ID from environment variable
	adminIDStr := os.Getenv("TELEGRAM_ADMIN_ID")
	if adminIDStr == "" {
		log.Println("TELEGRAM_ADMIN_ID environment variable not set, not sending notification")
		return
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid TELEGRAM_ADMIN_ID format: %v", err)
		return
	}

	userName := user.FirstName
	if user.LastName != "" {
		userName += " " + user.LastName
	}

	var reason string
	if HasEmoji(user.FirstName) || HasEmoji(user.LastName) {
		reason = "名称中包含emoji"
	} else if user.IsPremium {
		reason = "用户是Premium用户"
	} else if IsRandomUsername(user.Username) {
		reason = "用户名是无意义的随机字符串"
	} else if HasTelegramLinksInBio(ctx, bot, user.ID) {
		reason = "用户简介包含t.me链接"
	} else {
		reason = "符合垃圾用户特征"
	}

	// Get group information
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		log.Printf("Error getting chat info: %v", err)
		return
	}

	// Create chat and user links
	// For public groups, use the username if available
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

	// Create user link - always works even if user has no username
	userLink := fmt.Sprintf("tg://user?id=%d", user.ID)

	// Format group name and user name with links using HTML
	linkedGroupName := fmt.Sprintf("<a href=\"%s\">%s</a>", groupLink, chatInfo.Title)
	linkedUserName := fmt.Sprintf("<a href=\"%s\">%s</a>", userLink, userName)

	// Create HTML formatted message for admin
	message := fmt.Sprintf("⚠️ <b>安全提醒</b> [%s]\n"+
		"用户 %s 已被限制发送消息和媒体的权限\n"+
		"<b>原因</b>: %s",
		linkedGroupName, linkedUserName, reason)

	// Create unban button with callback data containing chat ID and user ID
	unbanCallbackData := fmt.Sprintf("unban:%d:%d", chatID, user.ID)
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         "解除限制",
				CallbackData: unbanCallbackData,
			},
		},
	}
	inlineKeyboard := telego.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	// Send HTML message to admin with the unban button
	adminMessageParams := telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: adminID},
		Text:        message,
		ParseMode:   "HTML", // Enable HTML formatting
		ReplyMarkup: &inlineKeyboard,
	}

	_, err = bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
	} else {
		log.Printf("Successfully sent restriction notice to admin for user %s", userName)
	}
}

// UnrestrictUser removes restrictions from a user in a chat
func UnrestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	// Create chat permissions that allow sending messages and media
	canSendMessages := true
	canSendMedia := true
	canSendPolls := true
	canSendOther := true
	canAddWebPreview := true

	permissions := telego.ChatPermissions{
		CanSendMessages:       &canSendMessages,
		CanSendAudios:         &canSendMedia,
		CanSendDocuments:      &canSendMedia,
		CanSendPhotos:         &canSendMedia,
		CanSendVideos:         &canSendMedia,
		CanSendVideoNotes:     &canSendMedia,
		CanSendVoiceNotes:     &canSendMedia,
		CanSendPolls:          &canSendPolls,
		CanSendOtherMessages:  &canSendOther,
		CanAddWebPagePreviews: &canAddWebPreview,
	}

	// Create unrestriction config
	params := telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: chatID},
		UserID:      userID,
		Permissions: permissions,
		UntilDate:   0, // 0 means permanent
	}

	// Apply unrestriction
	err := bot.RestrictChatMember(ctx, &params)
	if err != nil {
		log.Printf("Error unrestricting user %d: %v", userID, err)
	} else {
		log.Printf("Successfully unrestricted user %d in chat %d", userID, chatID)
	}
}

// UserCanSendMessages checks if a user has permission to send messages in a chat
func UserCanSendMessages(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) (bool, error) {
	// Get member info
	memberInfo, err := bot.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: telego.ChatID{ID: chatID},
		UserID: userID,
	})
	if err != nil {
		return false, fmt.Errorf("error getting member info: %w", err)
	}

	// Check if user has permission to send messages based on member status
	switch memberInfo.MemberStatus() {
	case telego.MemberStatusRestricted:
		// For restricted users, we need to check if they can send messages
		restrictedMember, ok := memberInfo.(*telego.ChatMemberRestricted)
		if !ok {
			return false, fmt.Errorf("unexpected member type")
		}
		return restrictedMember.CanSendMessages, nil
	case telego.MemberStatusMember, telego.MemberStatusAdministrator, telego.MemberStatusCreator:
		// Regular members, admins and creators can send messages by default
		return true, nil
	default:
		// Left or kicked users cannot send messages
		return false, nil
	}
}
