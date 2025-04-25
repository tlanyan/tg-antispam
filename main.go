package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

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

	// Get webhook configuration from environment variables
	webhookHost := os.Getenv("WEBHOOK_HOST")
	if webhookHost == "" {
		log.Fatal("WEBHOOK_HOST environment variable not set (e.g. https://example.com)")
	}

	webhookPath := os.Getenv("WEBHOOK_PATH")
	if webhookPath == "" {
		webhookPath = "/webhook" // Default webhook path
		log.Printf("Using default webhook path: %s", webhookPath)
	}

	webhookPort := os.Getenv("WEBHOOK_PORT")
	if webhookPort == "" {
		webhookPort = "8443" // Default port for Telegram webhooks
		log.Printf("Using default webhook port: %s", webhookPort)
	}

	webhookListen := "0.0.0.0:" + webhookPort
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	if (certFile == "" || keyFile == "") && !strings.HasPrefix(webhookHost, "https://") {
		log.Fatal("HTTPS configuration required: Set CERT_FILE and KEY_FILE env vars or use a HTTPS proxy")
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

	// Delete any existing webhook
	err = bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{})
	if err != nil {
		log.Fatalf("Failed to delete existing webhook: %v", err)
	}

	// Set up webhook
	webhookURL := webhookHost + webhookPath
	log.Printf("Setting webhook to: %s", webhookURL)

	setWebhookParams := &telego.SetWebhookParams{
		URL: webhookURL,
		AllowedUpdates: []string{"message", "edited_message", "channel_post", "edited_channel_post"},
	}

	err = bot.SetWebhook(ctx, setWebhookParams)
	if err != nil {
		log.Fatalf("Failed to set webhook: %v", err)
	}

	// Create HTTP server mux
	mux := http.NewServeMux()

	// Set up updates handler via webhook
	updates, err := bot.UpdatesViaWebhook(ctx,
		telego.WebhookHTTPServeMux(mux, webhookPath, bot.SecretToken()),
	)
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
		if message.From != nil && message.From.IsPremium {
			bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: message.MessageID,
			})
			restrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
			sendWarning(ctx.Context(), bot, message.Chat.ID, *message.From)
			return nil
		}

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

	// Start HTTP server
	log.Printf("Starting HTTP server on %s", webhookListen)
	if certFile != "" && keyFile != "" {
		log.Fatal(http.ListenAndServeTLS(webhookListen, certFile, keyFile, mux))
	} else {
		log.Printf("WARNING: Running without TLS. Make sure you have a HTTPS proxy in front of this server")
		log.Fatal(http.ListenAndServe(webhookListen, mux))
	}
}

// shouldRestrictUser checks if a user should be restricted based on their name and username
func shouldRestrictUser(user telego.User) bool {
	// Check for emoji in name
	if hasEmoji(user.FirstName) || hasEmoji(user.LastName) {
		return true
	}

	// Check for random username
	// if isRandomUsername(user.Username) {
	// 	return true
	// }

	return user.IsPremium
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
