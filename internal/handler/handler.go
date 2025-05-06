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

	// Global configuration
	globalConfig *config.Config
)

// Initialize initializes the handler with configuration
func Initialize(cfg *config.Config) {
	globalConfig = cfg
}

// SetupMessageHandlers configures all bot message and update handlers
func SetupMessageHandlers(bh *th.BotHandler, bot *telego.Bot) {
	// Initialize group repository if database is enabled
	InitGroupRepository()

	// Register commands
	RegisterCommands(bh, bot)

	// Skip messages from the bot itself
	botID := bot.ID()

	// Handle new chat members
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		// Skip if no sender information or sender is a bot
		if message.From == nil || message.From.IsBot {
			return nil
		}

		groupInfo := GetGroupInfo(ctx, bot, message.Chat.ID)
		if !groupInfo.IsAdmin {
			log.Printf("bot is not an admin for chat ID: %d", message.Chat.ID)
			return nil
		}
		log.Printf("Processing message: %+v", message)

		// Use database configuration if available
		useCAS := groupInfo.EnableCAS
		shouldRestrict := false
		reason := ""

		if useCAS {
			shouldRestrict, reason = CasRequest(message.From.ID)
		}

		if shouldRestrict {
			bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: message.MessageID,
			})
			RestrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
			// Send warning only if notifications are enabled
			if groupInfo.EnableNotification {
				SendWarning(ctx.Context(), bot, groupInfo, *message.From, reason)
			}
		}

		return nil
	})

	// Handle chat member updates
	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		// Process ChatMember updates (when users join chat or change status)
		if update.ChatMember != nil {
			log.Printf("Chat member update: %+v", update.ChatMember)
			chatId := update.ChatMember.Chat.ID
			groupInfo := GetGroupInfo(ctx, bot, chatId)

			newChatMember := update.ChatMember.NewChatMember
			log.Printf("new Chat member: %+v", newChatMember)

			fromUser := update.ChatMember.From

			// Skip updates related to the bot itself
			if fromUser.ID == botID {
				log.Printf("Skipping chat member update from the bot itself")
				return nil
			}

			// Track admin who promoted the bot
			if newChatMember.MemberUser().ID == botID {
				// Check if the bot's status was changed to admin
				if newChatMember.MemberStatus() == telego.MemberStatusAdministrator {
					// Record the user who promoted the bot to admin
					log.Printf("Bot was promoted to admin in chat %d by user %d", chatId, fromUser.ID)
					groupInfo.IsAdmin = true
					groupInfo.AdminID = fromUser.ID
					// Update the group info
					UpdateGroupInfo(groupInfo)
				} else {
					groupInfo.IsAdmin = false
					// Update the group info
					UpdateGroupInfo(groupInfo)
				}
				return nil
			}

			if !groupInfo.IsAdmin {
				log.Printf("Bot not and admin for chat ID: %d", chatId)
				return nil
			}

			user := newChatMember.MemberUser()
			if newChatMember.MemberIsMember() {
				// Skip bots
				if user.IsBot {
					log.Printf("Skipping bot: %s", user.FirstName)
					return nil
				}

				// 首次入群，等待入群机器人处理
				// @TODO: 需要优化，如果没有其它机器人，则需要处理
				if !fromUser.IsBot {
					log.Printf("Skipping first time join: %s", user.FirstName)
					return nil
				}

				// 检查是否是受限制的成员
				if newChatMember.MemberStatus() == telego.MemberStatusRestricted {
					restrictedMember, ok := newChatMember.(*telego.ChatMemberRestricted)
					if ok {
						// 现在可以访问 CanSendMessages 属性
						canSendMsg := restrictedMember.CanSendMessages
						if !canSendMsg {
							return nil
						}
					}
				}

				// Check if user has permission to send messages first
				hasPermission, err := UserCanSendMessages(ctx.Context(), bot, chatId, user.ID)
				if err != nil {
					log.Printf("Error checking user permissions: %v", err)
					return nil
				}

				if !hasPermission {
					log.Printf("User %s is already restricted, skipping", user.FirstName)
					return nil
				}

				// Check if user should be restricted
				shouldRestrict, reason := ShouldRestrictUser(ctx, bot, groupInfo, user)
				if !shouldRestrict && groupInfo.EnableCAS {
					shouldRestrict, reason = CasRequest(user.ID)
				}

				if shouldRestrict {
					log.Printf("Restricting user: %s, reason: %s", user.FirstName, reason)
					RestrictUser(ctx.Context(), bot, chatId, user.ID)
					// Send warning only if notifications are enabled
					if groupInfo.EnableNotification {
						SendWarning(ctx.Context(), bot, groupInfo, user, reason)
					}
				}
			}
		}
		return nil
	}, th.AnyChatMember())

	// Handle callback queries for unban button
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		return HandleCallbackQuery(ctx, bot, query)
	})
}

// ShouldRestrictUser determines if a user should be restricted
func ShouldRestrictUser(ctx context.Context, bot *telego.Bot, groupInfo *models.GroupInfo, user telego.User) (bool, string) {

	if groupInfo.BanPremium && user.IsPremium {
		return true, "reason_premium_user"
	}

	// Check for random username pattern
	if groupInfo.BanRandomUsername && IsRandomUsername(user.Username) {
		return true, "reason_random_username"
	}

	// Check for emoji in name
	if groupInfo.BanEmojiName && (HasEmoji(user.FirstName) || (user.LastName != "" && HasEmoji(user.LastName))) {
		return true, "reason_emoji_name"
	}

	// Check bio for links
	if groupInfo.BanBioLink && HasLinksInBio(ctx, bot, user.ID) {
		return true, "reason_bio_link"
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

	log.Printf("CAS response: %+v", result)
	// Check if user is in CAS database
	if result.Ok && result.Result.Offenses > 0 {
		return true, "reason_cas_blacklisted"
	}

	return false, ""
}

// HasLinksInBio checks if user's bio contains links
func HasLinksInBio(ctx context.Context, bot *telego.Bot, userID int64) bool {
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

// SendWarning sends a warning message about the restricted user
func SendWarning(ctx context.Context, bot *telego.Bot, groupInfo *models.GroupInfo, user telego.User, reason string) {
	if groupInfo.AdminID <= 0 {
		log.Printf("Admin ID is not set, do not send warning")
		return
	}

	// Get user's linked name
	linkedUserName := GetLinkedUserName(user)
	linkedGroupName := groupInfo.GetLinkedGroupName()
	if linkedGroupName == "" {
		log.Printf("failed to get Group name, do not send warning")
		return
	}

	// Get language setting
	language := models.LangSimplifiedChinese
	if groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// Create HTML formatted message for admin using translations
	message := fmt.Sprintf(
		"%s\n%s\n%s",
		fmt.Sprintf(models.GetTranslation(language, "warning_title"), linkedGroupName),
		fmt.Sprintf(models.GetTranslation(language, "warning_restricted"), linkedUserName),
		fmt.Sprintf(models.GetTranslation(language, "warning_reason"), models.GetTranslation(language, reason)),
	)

	// Create unban button with callback data containing chat ID and user ID
	unbanCallbackData := fmt.Sprintf("unban:%d:%d", groupInfo.GroupID, user.ID)
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         models.GetTranslation(language, "warning_unban_button"),
				CallbackData: unbanCallbackData,
			},
		},
	}
	inlineKeyboard := telego.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	// Send HTML message to admin with the unban button
	adminMessageParams := telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: groupInfo.AdminID},
		Text:        message,
		ParseMode:   "HTML", // Enable HTML formatting
		ReplyMarkup: &inlineKeyboard,
	}

	_, err := bot.SendMessage(ctx, &adminMessageParams)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
		groupInfo.AdminID = -1
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

// HandleCallbackQuery processees callback query data for unban actions
func HandleCallbackQuery(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
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

		// Get group information for language settings
		groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)
		language := models.LangSimplifiedChinese
		if groupInfo != nil && groupInfo.Language != "" {
			language = groupInfo.Language
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

		// Update the message with translated text
		messageText := fmt.Sprintf(models.GetTranslation(language, "warning_unbanned_message"), linkedUserName)

		// Check if we have a message to edit
		if query.Message != nil {
			// Access message fields correctly for MaybeInaccessibleMessage type
			msgChatID := query.Message.GetChat().ID
			messageID := query.Message.GetMessageID()

			bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: msgChatID},
				MessageID:   messageID,
				Text:        messageText,
				ParseMode:   "HTML",
				ReplyMarkup: nil, // Remove the button
			})
		}

		// Answer the callback query with translated text
		bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            models.GetTranslation(language, "warning_user_unbanned"),
		})
	}

	return nil
}
