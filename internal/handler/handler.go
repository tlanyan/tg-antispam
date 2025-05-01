package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/models"
)

var (
	// Compiled regular expressions
	emojiRegex  = regexp.MustCompile(`[\x{1F600}-\x{1F64F}|\x{1F300}-\x{1F5FF}|\x{1F680}-\x{1F6FF}|\x{1F700}-\x{1F77F}|\x{1F780}-\x{1F7FF}|\x{1F800}-\x{1F8FF}|\x{1F900}-\x{1F9FF}|\x{1FA00}-\x{1FA6F}|\x{1FA70}-\x{1FAFF}|\x{2600}-\x{26FF}|\x{2700}-\x{27BF}]`)
	tgLinkRegex = regexp.MustCompile(`t\.me`)

	CasRecords = models.NewUserActionManager(10)
	// Cache for group information
	GroupNameCache = make(map[int64]string)
	// Global configuration
	globalConfig *config.Config
)

// Initialize initializes the handler with configuration
func Initialize(cfg *config.Config) {
	globalConfig = cfg
}

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// Skip messages from the bot itself
	botInfo, err := bot.GetMe(context.Background())
	if err != nil {
		log.Printf("Error getting bot info: %v", err)
	}

	adminID := globalConfig.Admin.UserID

	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		log.Printf("Processing message: %+v", message)

		// Skip if no sender information or sender is a bot
		if message.From == nil || message.From.IsBot {
			return nil
		}

		shouldRestrict, reason := CasRequest(message.From.ID)
		if shouldRestrict {
			bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: message.MessageID,
			})
			RestrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
			SendWarning(ctx.Context(), bot, message.Chat.ID, *message.From, reason, adminID)
		}

		return nil
	})

	// Handle chat member updates
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// Process ChatMember updates (when users join chat or change status)
		if update.ChatMember != nil {
			newChatMember := update.ChatMember.NewChatMember
			log.Printf("Chat member update: %+v", update.ChatMember)
			log.Printf("new Chat member: %+v", newChatMember)

			// Skip updates related to the bot itself
			if botInfo != nil && update.ChatMember.From.ID == botInfo.ID {
				log.Printf("Skipping chat member update from the bot itself")
				return nil
			}

			chatId := update.ChatMember.Chat.ID
			user := newChatMember.MemberUser()
			if newChatMember.MemberIsMember() {
				// Skip bots
				if user.IsBot {
					log.Printf("Skipping bot: %s", user.FirstName)
					return nil
				}

				// 首次入群，等待入群机器人处理
				if !update.ChatMember.From.IsBot {
					log.Printf("Skipping first time join: %s", user.FirstName)
					return nil
				}

				// 检查是否是受限制的成员
				if newChatMember.MemberStatus() == telego.MemberStatusRestricted {
					restrictedMember, ok := newChatMember.(*telego.ChatMemberRestricted)
					if ok {
						// 现在可以访问 CanSendMessages 属性
						canSendMsg := restrictedMember.CanSendMessages
						if canSendMsg {
							return nil
						}
					}
				}

				// Check if user has permission to send messages first
				hasPermission, err := UserCanSendMessages(ctx.Context(), bot, update.ChatMember.Chat.ID, user.ID)
				if err != nil {
					log.Printf("Error checking user permissions: %v", err)
					return nil
				}

				if !hasPermission {
					log.Printf("User %s is already restricted, skipping", user.FirstName)
					return nil
				}

				// Check if user should be restricted
				shouldRestrict, reason := ShouldRestrictUser(ctx, bot, user)
				if !shouldRestrict {
					shouldRestrict, reason = CasRequest(user.ID)
				}
				if shouldRestrict {
					log.Printf("Restricting user: %s, reason: %s", user.FirstName, reason)
					RestrictUser(ctx.Context(), bot, chatId, user.ID)
					SendWarning(ctx.Context(), bot, chatId, user, reason, adminID)
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

// ShouldRestrictUser determines if a user should be restricted
func ShouldRestrictUser(ctx context.Context, bot *telego.Bot, user telego.User) (bool, string) {

	if globalConfig.Antispam.RestrictPremiumUser && user.IsPremium {
		return true, "用户是Premium用户"
	}

	// Check for random username pattern
	if globalConfig.Antispam.CheckRandomUsername && IsRandomUsername(user.Username) {
		return true, "疑似随机用户名"
	}

	// Check for emoji in name
	if globalConfig.Antispam.CheckEmojiUsername && (HasEmoji(user.FirstName) || (user.LastName != "" && HasEmoji(user.LastName))) {
		return true, "用户名含有表情符号"
	}

	// Check bio for Telegram links
	if globalConfig.Antispam.CheckBioLinks && HasTelegramLinksInBio(ctx, bot, user.ID) {
		return true, "个人简介包含t.me链接"
	}

	return false, ""
}

// CasRequest checks if a user is in the CAS (Combot Anti-Spam) database
func CasRequest(userID int64) (bool, string) {
	// If CAS checking is disabled, return false
	if globalConfig != nil && !globalConfig.Antispam.UseCAS {
		return false, ""
	}

	// Check cache first
	if CasRecords.Contains(userID) {
		return false, ""
	}

	// Construct CAS API URL
	apiURL := fmt.Sprintf("https://api.cas.chat/check?user_id=%d", userID)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make the request
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("Error making CAS request: %v", err)
		return false, ""
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("CAS API returned non-OK status: %d", resp.StatusCode)
		return false, ""
	}

	// Decode JSON response
	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			Offenses int `json:"offenses"`
		} `json:"result"`
	}

	CasRecords.Add(userID)
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding CAS response: %v", err)
		return false, ""
	}

	log.Printf("CAS response: %+v", resp.Body)
	// Check if user is in CAS database
	if result.Ok && result.Result.Offenses > 0 {
		return true, "用户在 CAS 黑名单中"
	}

	return false, ""
}

// HasTelegramLinksInBio checks if user's bio contains t.me links
func HasTelegramLinksInBio(ctx context.Context, bot *telego.Bot, userID int64) bool {
	user, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: userID},
	})

	if err != nil {
		log.Printf("Error getting user info: %v", err)
		return false
	}

	if user.Bio != "" {
		return tgLinkRegex.MatchString(user.Bio)
	}

	return false
}

// HasEmoji checks if string contains emoji
func HasEmoji(s string) bool {
	return emojiRegex.MatchString(s)
}

// IsRandomUsername checks if username matches common spam patterns
func IsRandomUsername(username string) bool {
	if username == "" {
		return false
	}

	if username == "" {
		return false
	}

	consonantsRegex := regexp.MustCompile(`[bcdfghjklmnpqrstvwxyz]{5}`)
	if consonantsRegex.MatchString(strings.ToLower(username)) {
		return true
	}

	digitsRegex := regexp.MustCompile(`\d{7}`)
	if digitsRegex.MatchString(username) {
		return true
	}

	return false
}

// RestrictUser restricts a user from sending messages
func RestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	untilDate := int64(0) // forever

	// Create boolean variables for permissions
	falseValue := false

	restrictParams := &telego.RestrictChatMemberParams{
		ChatID: telego.ChatID{ID: chatID},
		UserID: userID,
		Permissions: telego.ChatPermissions{
			CanSendMessages:       &falseValue,
			CanSendAudios:         &falseValue,
			CanSendDocuments:      &falseValue,
			CanSendPhotos:         &falseValue,
			CanSendVideos:         &falseValue,
			CanSendVideoNotes:     &falseValue,
			CanSendVoiceNotes:     &falseValue,
			CanSendPolls:          &falseValue,
			CanSendOtherMessages:  &falseValue,
			CanAddWebPagePreviews: &falseValue,
			CanChangeInfo:         &falseValue,
			CanInviteUsers:        &falseValue,
			CanPinMessages:        &falseValue,
			CanManageTopics:       &falseValue,
		},
		UntilDate: untilDate,
	}

	err := bot.RestrictChatMember(ctx, restrictParams)
	if err != nil {
		log.Printf("Error restricting user: %v", err)
	} else {
		log.Printf("Successfully restricted user %d in chat %d", userID, chatID)
	}
}

func GetLinkedUserName(user telego.User) string {
	// Get user display name
	userName := user.FirstName
	if user.LastName != "" {
		userName += " " + user.LastName
	}

	// Add username if available
	displayName := userName
	if user.Username != "" {
		displayName = fmt.Sprintf("%s (@%s)", userName, user.Username)
	}

	// Create user link
	userLink := fmt.Sprintf("tg://user?id=%d", user.ID)
	linkedUserName := fmt.Sprintf("<a href=\"%s\">%s</a>", userLink, displayName)
	return linkedUserName
}

// GetLinkedGroupName gets a linked HTML representation of the group name with caching
func GetLinkedGroupName(ctx context.Context, bot *telego.Bot, chatID int64) string {
	// Check cache first
	cachedName, exists := GroupNameCache[chatID]
	if exists {
		return cachedName
	}

	// Cache miss, fetch from API
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		log.Printf("Error getting chat info: %v", err)
		return ""
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

	// Format group name with link using HTML
	linkedGroupName := fmt.Sprintf("<a href=\"%s\">%s</a>", groupLink, chatInfo.Title)
	GroupNameCache[chatID] = linkedGroupName
	return linkedGroupName
}

// SendWarning sends a warning message about the restricted user
func SendWarning(ctx context.Context, bot *telego.Bot, chatID int64, user telego.User, reason string, adminID int64) {
	if adminID < 0 {
		log.Printf("Admin ID is not set, not sending warning")
		return
	}

	linkedUserName := GetLinkedUserName(user)
	linkedGroupName := GetLinkedGroupName(ctx, bot, chatID)
	if linkedGroupName == "" {
		log.Printf("Group name is not set, not sending warning")
		return
	}

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

	_, err := bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
	} else {
		log.Printf("Successfully sent restriction notice to admin for user %s", linkedUserName)
	}
}

// UnrestrictUser removes restrictions from a user
func UnrestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	// Set permissions to allow sending messages
	trueValue := true
	falseValue := false

	restrictParams := &telego.RestrictChatMemberParams{
		ChatID: telego.ChatID{ID: chatID},
		UserID: userID,
		Permissions: telego.ChatPermissions{
			CanSendMessages:       &trueValue,
			CanSendAudios:         &trueValue,
			CanSendDocuments:      &trueValue,
			CanSendPhotos:         &trueValue,
			CanSendVideos:         &trueValue,
			CanSendVideoNotes:     &trueValue,
			CanSendVoiceNotes:     &trueValue,
			CanSendPolls:          &trueValue,
			CanSendOtherMessages:  &trueValue,
			CanAddWebPagePreviews: &trueValue,
			CanChangeInfo:         &falseValue,
			CanInviteUsers:        &trueValue,
			CanPinMessages:        &falseValue,
			CanManageTopics:       &falseValue,
		},
	}

	err := bot.RestrictChatMember(ctx, restrictParams)
	if err != nil {
		log.Printf("Error unrestricting user: %v", err)
	} else {
		log.Printf("Successfully unrestricted user %d in chat %d", userID, chatID)
	}

	// Remove from CAS cache if exists
	if CasRecords.Contains(userID) {
		CasRecords.Remove(userID)
		log.Printf("Removed user %d from CAS cache", userID)
	}
}

// UserCanSendMessages checks if user has permission to send messages
func UserCanSendMessages(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) (bool, error) {
	// Get chat member info to check their current permissions
	chatMemberParams := &telego.GetChatMemberParams{
		ChatID: telego.ChatID{ID: chatID},
		UserID: userID,
	}

	member, err := bot.GetChatMember(ctx, chatMemberParams)
	if err != nil {
		return false, fmt.Errorf("error getting chat member: %w", err)
	}

	// Check if it's a restricted member
	if member.MemberStatus() == telego.MemberStatusRestricted {
		restrictedMember, ok := member.(*telego.ChatMemberRestricted)
		if ok {
			return restrictedMember.CanSendMessages, nil
		}
		return false, fmt.Errorf("failed to convert member to restricted type")
	}

	// If not restricted, they can send messages
	return true, nil
}
