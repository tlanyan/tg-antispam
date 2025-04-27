package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

var (
	// Compiled regular expressions
	emojiRegex = regexp.MustCompile(`[\x{1F600}-\x{1F64F}|\x{1F300}-\x{1F5FF}|\x{1F680}-\x{1F6FF}|\x{1F700}-\x{1F77F}|\x{1F780}-\x{1F7FF}|\x{1F800}-\x{1F8FF}|\x{1F900}-\x{1F9FF}|\x{1FA00}-\x{1FA6F}|\x{1FA70}-\x{1FAFF}|\x{2600}-\x{26FF}|\x{2700}-\x{27BF}]`)
)

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		log.Printf("Processing message: %+v", message)
		if message.From != nil && message.From.IsPremium {
			if message.From.IsBot {
				log.Printf("Skipping bot: %s", message.From.FirstName)
				return nil
			}
			log.Printf("Found premium user: %s", message.From.FirstName)
			bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: message.MessageID,
			})
			RestrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
			SendWarning(ctx.Context(), bot, message.Chat.ID, *message.From)
			return nil
		}

		return nil
	})

	// Handle chat member updates
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// Process ChatMember updates (when users join chat or change status)
		if update.ChatMember != nil {
			log.Printf("Chat member update: %+v", update.ChatMember)

			// Check if new member joined (status changed to 'member')
			if update.ChatMember.NewChatMember.MemberStatus() == "member" {

				newMember := update.ChatMember.NewChatMember.MemberUser()
				log.Printf("New user detected via chat_member update: %s", newMember.FirstName)

				// Skip bots
				if newMember.IsBot {
					log.Printf("Skipping bot: %s", newMember.FirstName)
					return nil
				}

				// Check if user should be restricted
				if ShouldRestrictUser(newMember) {
					log.Printf("Restricting user: %s", newMember.FirstName)
					RestrictUser(ctx.Context(), bot, update.ChatMember.Chat.ID, newMember.ID)
					SendWarning(ctx.Context(), bot, update.ChatMember.Chat.ID, newMember)
				}
			}
		}
		return nil
	}, th.AnyChatMember())
}

// ShouldRestrictUser checks if a user should be restricted based on their name and username
func ShouldRestrictUser(user telego.User) bool {
	// Check for emoji in name
	if HasEmoji(user.FirstName) || HasEmoji(user.LastName) {
		return true
	}

	// Check for random username
	// if IsRandomUsername(user.Username) {
	// 	return true
	// }

	return user.IsPremium
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
	} else if IsRandomUsername(user.Username) {
		reason = "用户名是无意义的随机字符串"
	} else if user.IsPremium {
		reason = "用户是Premium用户"
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
		if groupIDForLink < 0 && groupIDForLink < -1000000000000 {
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

	// Send HTML message to admin
	adminMessageParams := telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: adminID},
		Text:      message,
		ParseMode: "HTML", // Enable HTML formatting
	}

	_, err = bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
	} else {
		log.Printf("Successfully sent restriction notice to admin for user %s", userName)
	}
}