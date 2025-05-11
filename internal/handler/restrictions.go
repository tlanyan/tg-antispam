package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
)

var (
	// Compiled regular expressions
	emojiRegex  = regexp.MustCompile(`[\x{1F600}-\x{1F64F}|\x{1F300}-\x{1F5FF}|\x{1F680}-\x{1F6FF}|\x{1F700}-\x{1F77F}|\x{1F780}-\x{1F7FF}|\x{1F800}-\x{1F8FF}|\x{1F900}-\x{1F9FF}|\x{1FA00}-\x{1FA6F}|\x{1FA70}-\x{1FAFF}|\x{2600}-\x{26FF}|\x{2700}-\x{27BF}]`)
	tgLinkRegex = regexp.MustCompile(`t\.me`)

	CasRecords = models.NewUserActionManager(10)
)

// ShouldRestrictUser determines if a user should be restricted
func ShouldRestrictUser(ctx context.Context, bot *telego.Bot, groupInfo *models.GroupInfo, user telego.User) (bool, string) {
	if groupInfo.BanPremium && user.IsPremium {
		return true, "reason_premium_user"
	}

	if groupInfo.BanEmojiName && HasEmoji(user.FirstName) {
		return true, "reason_emoji_name"
	}

	if groupInfo.BanRandomUsername && user.Username != "" && IsRandomUsername(user.Username) {
		return true, "reason_random_username"
	}

	if groupInfo.BanBioLink && HasLinksInBio(ctx, bot, user.ID) {
		return true, "reason_bio_link"
	}

	return false, ""
}

// CasRequest checks if a user is listed in the Combot Anti-Spam System (CAS)
func CasRequest(userID int64) (bool, string) {
	if CasRecords.Contains(userID) {
		return false, ""
	}

	// Make request to CAS API
	casResp, err := http.Get("https://api.cas.chat/check?user_id=" + strconv.FormatInt(userID, 10))
	if err != nil {
		logger.Warningf("Error checking CAS for user %d: %v", userID, err)
		return false, ""
	}
	defer casResp.Body.Close()

	if casResp.StatusCode != 200 {
		logger.Warningf("CAS API returned status code %d for user %d", casResp.StatusCode, userID)
		return false, ""
	}

	// Parse response
	var casResult struct {
		Ok     bool `json:"ok"`
		Result struct {
			Offenses  int   `json:"offenses"`
			TimeAdded int64 `json:"time_added"`
		} `json:"result"`
	}

	if err := json.NewDecoder(casResp.Body).Decode(&casResult); err != nil {
		logger.Warningf("Error decoding CAS response for user %d: %v", userID, err)
		return false, ""
	}

	// Cache the result
	if casResult.Ok {
		CasRecords.Add(userID)
	}

	return casResult.Ok, "reason_cas"
}

// HasLinksInBio checks if a user has t.me links in their bio
func HasLinksInBio(ctx context.Context, bot *telego.Bot, userID int64) bool {
	chat, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: userID},
	})

	if err != nil {
		logger.Warningf("Error getting chat info for user %d: %v", userID, err)
		return false
	}

	return chat.Bio != "" && tgLinkRegex.MatchString(chat.Bio)
}

// HasEmoji checks if a string contains emoji
func HasEmoji(s string) bool {
	return emojiRegex.MatchString(s)
}

// IsRandomUsername checks if a username appears to be randomly generated
func IsRandomUsername(username string) bool {
	if len(username) < 5 {
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

// RestrictUser restricts a user in a chat
func RestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	permissions := telego.ChatPermissions{}

	err := bot.RestrictChatMember(ctx, &telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: chatID},
		UserID:      userID,
		Permissions: permissions,
	})

	if err != nil {
		logger.Warningf("Error restricting user %d in chat %d: %v", userID, chatID, err)
	} else {
		logger.Infof("Successfully restricted user %d in chat %d", userID, chatID)
	}
}

// GetLinkedUserName returns an HTML formatted string for a user's name with a link to their profile
func GetLinkedUserName(user telego.User) string {
	displayName := user.FirstName
	if user.LastName != "" {
		displayName += " " + user.LastName
	}

	// Handle '&', '<', '>' for HTML safety
	displayName = strings.ReplaceAll(displayName, "&", "&amp;")
	displayName = strings.ReplaceAll(displayName, "<", "&lt;")
	displayName = strings.ReplaceAll(displayName, ">", "&gt;")

	// Create link to user's profile
	return fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", user.ID, displayName)
}

// SendWarning sends a warning message about the restricted user
func SendWarning(ctx context.Context, bot *telego.Bot, groupInfo *models.GroupInfo, user telego.User, reason string) {
	language := groupInfo.Language

	// Get user display name with HTML link
	userLink := GetLinkedUserName(user)

	linkedGroupName := groupInfo.GetLinkedGroupName()
	if linkedGroupName == "" {
		logger.Infof("failed to get Group name, do not send warning")
		return
	}

	// Construct message with appropriate translation
	message := fmt.Sprintf(
		"%s\n%s\n%s",
		fmt.Sprintf(models.GetTranslation(language, "warning_title"), linkedGroupName),
		fmt.Sprintf(models.GetTranslation(language, "warning_restricted"), userLink),
		fmt.Sprintf(models.GetTranslation(language, "warning_reason"), models.GetTranslation(language, reason)),
	)

	// Send notification to admin chat if it exists
	if groupInfo.AdminID > 0 {
		// Create admin unban button
		adminUnbanButton := telego.InlineKeyboardButton{
			Text:         models.GetTranslation(groupInfo.Language, "warning_unban_button"),
			CallbackData: fmt.Sprintf("unban:%d:%d", groupInfo.GroupID, user.ID),
		}
		adminMarkup := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{adminUnbanButton},
			},
		}

		_, err := bot.SendMessage(ctx, &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: groupInfo.AdminID},
			Text:        message,
			ParseMode:   "HTML",
			ReplyMarkup: adminMarkup,
		})
		if err != nil {
			logger.Warningf("Error sending warning message to admin: %v", err)
		}
	}

	// Get bot username to create the deep link
	botInfo, err := bot.GetMe(ctx)
	if err != nil {
		logger.Warningf("Error getting bot info: %v", err)
		return
	}

	// Send notification to the group with a link to the bot's private chat
	selfUnbanMessage := fmt.Sprintf(
		models.GetTranslation(language, "warning_self_unban_instruction"),
		userLink,
	)

	// Create URL button to open private chat with the bot
	// Using deep linking to pass group ID and user ID
	startParam := fmt.Sprintf("unban_%d_%d", groupInfo.GroupID, user.ID)
	botURL := fmt.Sprintf("https://t.me/%s?start=%s", botInfo.Username, startParam)

	selfUnbanButton := telego.InlineKeyboardButton{
		Text: models.GetTranslation(groupInfo.Language, "warning_self_unban_button"),
		URL:  botURL,
	}
	selfUnbanMarkup := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{selfUnbanButton},
		},
	}

	// Send notification to the group
	msg, err := bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: groupInfo.GroupID},
		Text:        selfUnbanMessage,
		ParseMode:   "HTML",
		ReplyMarkup: selfUnbanMarkup,
	})

	if err != nil {
		logger.Warningf("Error sending warning message to group: %v", err)
		return
	}

	// Delete the group message after 3 minutes
	go func() {
		time.Sleep(3 * time.Minute)
		err := bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: groupInfo.GroupID},
			MessageID: msg.MessageID,
		})
		if err != nil {
			logger.Warningf("Error deleting warning message: %v", err)
		}
	}()
}

// UnrestrictUser removes restrictions from a user in a chat
func UnrestrictUser(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) {
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})

	permissions := telego.ChatPermissions{}
	if err == nil && chatInfo.Permissions != nil {
		permissions = *chatInfo.Permissions
	}

	err = bot.RestrictChatMember(ctx, &telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: chatID},
		UserID:      userID,
		Permissions: permissions,
	})

	if err != nil {
		logger.Warningf("Error unrestricting user %d in chat %d: %v", userID, chatID, err)
	} else {
		logger.Infof("Successfully unrestricted user %d in chat %d", userID, chatID)
	}
}

// UserCanSendMessages checks if a user has permission to send messages in a chat
func UserCanSendMessages(ctx context.Context, bot *telego.Bot, chatID int64, userID int64) (bool, error) {
	// Get the user's current status in the chat
	chatMember, err := bot.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: telego.ChatID{ID: chatID},
		UserID: userID,
	})

	if err != nil {
		return false, err
	}

	// If user is a restricted member, check the permission
	if chatMember.MemberStatus() == telego.MemberStatusRestricted {
		if restrictedMember, ok := chatMember.(*telego.ChatMemberRestricted); ok {
			return restrictedMember.CanSendMessages, nil
		}
	}

	// Users with admin or creator status can always send messages
	if chatMember.MemberStatus() == telego.MemberStatusAdministrator || chatMember.MemberStatus() == telego.MemberStatusCreator {
		return true, nil
	}

	// Regular members can send messages unless otherwise specified
	return chatMember.MemberStatus() == telego.MemberStatusMember, nil
}
