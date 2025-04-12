package main

import (
	"context"
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

func main() {
	// Create context
	ctx := context.Background()

	// Get bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	// Initialize bot
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Get bot info
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatalf("Failed to get bot info: %v", err)
	}
	log.Printf("Authorized on account %s", botUser.Username)

	// Set up updates handler
	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}

	// Setup handler
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatalf("Failed to create bot handler: %v", err)
	}
	defer bh.Stop()

	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.NewChatMembers != nil {
			for _, newMember := range message.NewChatMembers {
				// Skip bots
				if newMember.IsBot {
					continue
				}

				// Check if user should be restricted
				if shouldRestrictUser(newMember) {
					restrictUser(ctx.Context(), bot, message.Chat.ID, newMember.ID)
					sendWarning(ctx.Context(), bot, message.Chat.ID, newMember)
				}
			}
		}
		return nil
	})

	// Start handling updates
	bh.Start()

	// Keep running until interrupted
	select {}
}

// shouldRestrictUser checks if a user should be restricted based on their name and username
func shouldRestrictUser(user telego.User) bool {
	// Check for emoji in name
	if hasEmoji(user.FirstName) || hasEmoji(user.LastName) {
		return true
	}

	// Check for random username
	if isRandomUsername(user.Username) {
		return true
	}

	return false
}

// hasEmoji checks if a string contains emoji characters
func hasEmoji(s string) bool {
	if s == "" {
		return false
	}
	return emojiRegex.MatchString(s)
}

// isRandomUsername checks if a username appears to be a random string
func isRandomUsername(username string) bool {
	if username == "" {
		return false
	}

	return false
}

// restrictUser restricts a user's permissions in a chat
func restrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
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

// sendWarning sends a warning message about the restricted user to the specified admin
func sendWarning(ctx context.Context, bot *telego.Bot, chatID int64, user telego.User) {
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
	if hasEmoji(user.FirstName) || hasEmoji(user.LastName) {
		reason = "名称中包含emoji"
	} else if isRandomUsername(user.Username) {
		reason = "用户名是无意义的随机字符串"
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

	// Create message for admin
	groupName := chatInfo.Title
	message := "⚠️ 安全提醒 [" + groupName + "]\n" +
		"用户 " + userName + " 已被限制发送消息和媒体的权限\n" +
		"原因: " + reason

	// Send message to admin
	adminMessageParams := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: adminID},
		Text:   message,
	}

	_, err = bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
	} else {
		log.Printf("Successfully sent restriction notice to admin for user %s", userName)
	}
}
