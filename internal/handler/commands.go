package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"tg-antispam/internal/models"
	"tg-antispam/internal/storage"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

// RegisterCommands registers bot commands
func RegisterCommands(bh *th.BotHandler, bot *telego.Bot) {
	// Help command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/help" {
			return sendHelpMessage(ctx, bot, message)
		}
		return nil
	})

	// Settings command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/settings" {
			return handleSettingsCommand(ctx, bot, message)
		}
		return nil
	})

	// Toggle premium command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/toggle_premium" {
			return handleTogglePremiumCommand(ctx, bot, message)
		}
		return nil
	})

	// Toggle CAS command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/toggle_cas" {
			return handleToggleCasCommand(ctx, bot, message)
		}
		return nil
	})

	// Toggle notifications command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/toggle_notifications" {
			return handleToggleNotificationsCommand(ctx, bot, message)
		}
		return nil
	})

	// Language command
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Text != "" && message.Text == "/language" {
			return handleLanguageCommand(ctx, bot, message)
		}
		return nil
	})

	// Handle language selection callbacks
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		prefix := "lang_"
		if strings.HasPrefix(query.Data, prefix) {
			language := strings.TrimPrefix(query.Data, prefix)
			return setLanguage(ctx, bot, query, language)
		}
		return nil
	})

	// Handle group selection callbacks
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		if strings.HasPrefix(query.Data, "group_") {
			return handleGroupSelection(ctx, bot, query)
		}
		return nil
	})

	// Handle action callbacks
	bh.HandleCallbackQuery(func(ctx *th.Context, query telego.CallbackQuery) error {
		if strings.HasPrefix(query.Data, "action_") {
			return handleActionCallback(ctx, bot, query)
		}
		return nil
	})

	// Handle group ID input
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		if message.Chat.Type == "private" && message.ReplyToMessage != nil {
			// Check if the message is a reply to our "enter group ID" message
			if message.ReplyToMessage.From.ID == bot.ID() &&
				strings.Contains(message.ReplyToMessage.Text, "请输入群组ID") ||
				strings.Contains(message.ReplyToMessage.Text, "請輸入群組ID") ||
				strings.Contains(message.ReplyToMessage.Text, "Please enter the Group ID") {
				return handleGroupIDInput(ctx, bot, message)
			}
		}
		return nil
	})
}

// sendHelpMessage sends help information
func sendHelpMessage(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	// Get the group's language settings if in a group, otherwise use default
	language := models.LangSimplifiedChinese
	if message.Chat.Type != "private" {
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo != nil && groupInfo.Language != "" {
			language = groupInfo.Language
		}
	}

	var helpText string
	if message.Chat.Type == "private" {
		// In private chat, show instructions for using bot in private mode
		helpText = fmt.Sprintf("<b>%s</b>\n\n%s\n\n<b>%s</b>\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n\n<b>%s</b>",
			models.GetTranslation(language, "help_title"),
			models.GetTranslation(language, "help_description"),
			models.GetTranslation(language, "help_commands"),
			models.GetTranslation(language, "help_cmd_help"),
			models.GetTranslation(language, "help_cmd_settings"),
			models.GetTranslation(language, "help_cmd_toggle_premium"),
			models.GetTranslation(language, "help_cmd_toggle_cas"),
			models.GetTranslation(language, "help_cmd_toggle_random_username"),
			models.GetTranslation(language, "help_cmd_toggle_emoji_name"),
			models.GetTranslation(language, "help_cmd_toggle_bio_link"),
			models.GetTranslation(language, "help_cmd_toggle_notifications"),
			models.GetTranslation(language, "help_cmd_language"),
			models.GetTranslation(language, "help_note"),
		)
	} else {
		// In group chat, suggest using private chat with the bot
		botUsername, _ := getBotUsername(ctx.Context(), bot)
		helpText = fmt.Sprintf("<b>%s</b>\n\n%s\n\n%s @%s",
			models.GetTranslation(language, "help_title"),
			models.GetTranslation(language, "help_description"),
			models.GetTranslation(language, "please_use_private_chat"),
			botUsername,
		)
	}

	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      helpText,
		ParseMode: "HTML",
	})

	return err
}

// handleSettingsCommand handles the /settings command
func handleSettingsCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	language := models.LangSimplifiedChinese

	if message.Chat.Type == "private" {
		// In private chat, show group selection
		return showGroupSelection(ctx, bot, message, "settings")
	} else {
		// In group chat, get group settings
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo.Language != "" {
			language = groupInfo.Language
		}

		// Check if sender is admin
		senderIsAdmin, err := isUserAdmin(ctx.Context(), bot, message.Chat.ID, message.From.ID)
		if err != nil || !senderIsAdmin {
			botUsername, _ := getBotUsername(ctx.Context(), bot)
			_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text: fmt.Sprintf("%s @%s",
					models.GetTranslation(language, "please_use_private_chat"),
					botUsername),
			})
			return err
		}

		return showGroupSettings(ctx, bot, message, message.Chat.ID, language)
	}
}

// handleTogglePremiumCommand handles the /toggle_premium command
func handleTogglePremiumCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	if message.Chat.Type == "private" {
		// In private chat, show group selection
		return showGroupSelection(ctx, bot, message, "toggle_premium")
	} else {
		// In group chat, suggest using private chat
		language := models.LangSimplifiedChinese
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo.Language != "" {
			language = groupInfo.Language
		}

		botUsername, _ := getBotUsername(ctx.Context(), bot)
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s @%s",
				models.GetTranslation(language, "please_use_private_chat"),
				botUsername),
		})
		return err
	}
}

// handleToggleCasCommand handles the /toggle_cas command
func handleToggleCasCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	if message.Chat.Type == "private" {
		// In private chat, show group selection
		return showGroupSelection(ctx, bot, message, "toggle_cas")
	} else {
		// In group chat, suggest using private chat
		language := models.LangSimplifiedChinese
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo.Language != "" {
			language = groupInfo.Language
		}

		botUsername, _ := getBotUsername(ctx.Context(), bot)
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s @%s",
				models.GetTranslation(language, "please_use_private_chat"),
				botUsername),
		})
		return err
	}
}

// handleToggleNotificationsCommand handles the /toggle_notifications command
func handleToggleNotificationsCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	if message.Chat.Type == "private" {
		// In private chat, show group selection
		return showGroupSelection(ctx, bot, message, "toggle_notifications")
	} else {
		// In group chat, suggest using private chat
		language := models.LangSimplifiedChinese
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo.Language != "" {
			language = groupInfo.Language
		}

		botUsername, _ := getBotUsername(ctx.Context(), bot)
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s @%s",
				models.GetTranslation(language, "please_use_private_chat"),
				botUsername),
		})
		return err
	}
}

// handleLanguageCommand handles the /language command
func handleLanguageCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	if message.Chat.Type == "private" {
		// In private chat, show group selection
		return showGroupSelection(ctx, bot, message, "language")
	} else {
		// In group chat, suggest using private chat
		language := models.LangSimplifiedChinese
		groupInfo := GetGroupInfo(ctx.Context(), bot, message.Chat.ID)
		if groupInfo.Language != "" {
			language = groupInfo.Language
		}

		botUsername, _ := getBotUsername(ctx.Context(), bot)
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s @%s",
				models.GetTranslation(language, "please_use_private_chat"),
				botUsername),
		})
		return err
	}
}

// showGroupSettings shows current group settings
func showGroupSettings(ctx *th.Context, bot *telego.Bot, message telego.Message, groupID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, groupID)

	// Update language if provided in group info
	if groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// 构建设置消息
	// Get status strings based on settings
	banPremiumStatus := models.GetTranslation(language, "disabled")
	if groupInfo.BanPremium {
		banPremiumStatus = models.GetTranslation(language, "enabled")
	}

	casStatus := models.GetTranslation(language, "disabled")
	if groupInfo.EnableCAS {
		casStatus = models.GetTranslation(language, "enabled")
	}

	randomUsernameStatus := models.GetTranslation(language, "disabled")
	if groupInfo.BanRandomUsername {
		randomUsernameStatus = models.GetTranslation(language, "enabled")
	}

	emojiNameStatus := models.GetTranslation(language, "disabled")
	if groupInfo.BanEmojiName {
		emojiNameStatus = models.GetTranslation(language, "enabled")
	}

	bioLinkStatus := models.GetTranslation(language, "disabled")
	if groupInfo.BanBioLink {
		bioLinkStatus = models.GetTranslation(language, "enabled")
	}

	notificationStatus := models.GetTranslation(language, "disabled")
	if groupInfo.EnableNotification {
		notificationStatus = models.GetTranslation(language, "enabled")
	}

	// Get language name
	languageName := models.GetLanguageName(language, language)

	msgText := fmt.Sprintf(
		`<b>%s</b>

<b>%s</b> %s

<b>%s</b>
%s
%s
%s
%s
%s
%s
%s
%s
%s
%s
%s
%s`,
		fmt.Sprintf(models.GetTranslation(language, "settings_title"), groupInfo.GroupName),
		models.GetTranslation(language, "settings_bot_status"),
		models.GetTranslation(language, "settings_active"),
		models.GetTranslation(language, "settings_current"),
		fmt.Sprintf(models.GetTranslation(language, "settings_ban_premium"), banPremiumStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_cas"), casStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_random_name"), randomUsernameStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_emoji_name"), emojiNameStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_bio_link"), bioLinkStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_notifications"), notificationStatus),
		fmt.Sprintf(models.GetTranslation(language, "settings_language"), languageName),
		models.GetTranslation(language, "settings_commands"),
		models.GetTranslation(language, "settings_cmd_premium"),
		models.GetTranslation(language, "settings_cmd_cas"),
		models.GetTranslation(language, "settings_cmd_notifications"),
		models.GetTranslation(language, "settings_cmd_language"),
	)

	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      msgText,
		ParseMode: "HTML",
	})

	return err
}

// togglePremiumBanning toggles premium user banning
func togglePremiumBanning(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.BanPremium = !groupInfo.BanPremium

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.BanPremium {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_premium")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// toggleCasVerification toggles CAS verification
func toggleCasVerification(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.EnableCAS = !groupInfo.EnableCAS

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.EnableCAS {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_cas")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// toggleRandomUsername toggles random username check
func toggleRandomUsername(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.BanRandomUsername = !groupInfo.BanRandomUsername

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.BanRandomUsername {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_random_username")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// toggleEmojiUsername toggles emoji username check
func toggleEmojiName(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.BanEmojiName = !groupInfo.BanEmojiName

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.BanEmojiName {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_emoji_name")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// toggleBioLink toggles bio link check
func toggleBioLink(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.BanBioLink = !groupInfo.BanBioLink

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.BanBioLink {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_bio_link")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// toggleNotifications toggles admin notifications
func toggleNotifications(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// 获取群组设置
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Update the setting
	groupInfo.EnableNotification = !groupInfo.EnableNotification

	// Save changes
	UpdateGroupInfo(groupInfo)

	// Get status string
	status := models.GetTranslation(language, "disabled_text")
	if groupInfo.EnableNotification {
		status = models.GetTranslation(language, "enabled_text")
	}

	// Create confirmation message
	translatedSettingName := models.GetTranslation(language, "setting_notifications")
	msgText := fmt.Sprintf(models.GetTranslation(language, "setting_updated"), translatedSettingName, status)

	// Send confirmation message
	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: messageID},
		Text:   msgText,
	})

	return err
}

// isUserAdmin checks if user is an admin in the chat
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

// showLanguageOptions shows language selection options
func showLanguageOptions(ctx *th.Context, bot *telego.Bot, chatID int64, messageID int64, language string) error {
	// Create keyboard with language options
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         models.GetLanguageName(language, models.LangSimplifiedChinese),
				CallbackData: fmt.Sprintf("lang_%s_%d", models.LangSimplifiedChinese, chatID),
			},
		},
		{
			{
				Text:         models.GetLanguageName(language, models.LangTraditionalChinese),
				CallbackData: fmt.Sprintf("lang_%s_%d", models.LangTraditionalChinese, chatID),
			},
		},
		{
			{
				Text:         models.GetLanguageName(language, models.LangEnglish),
				CallbackData: fmt.Sprintf("lang_%s_%d", models.LangEnglish, chatID),
			},
		},
	}

	// Translate message using current language
	msgText := fmt.Sprintf("%s\n%s",
		models.GetTranslation(language, "language_select"),
		fmt.Sprintf(models.GetTranslation(language, "for_group"), GetGroupInfo(ctx.Context(), bot, chatID).GroupName))

	_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: messageID},
		Text:        msgText,
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})

	return err
}

// setLanguage sets the bot language for a group
func setLanguage(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery, language string) error {
	// Check if the message is accessible
	if query.Message == nil {
		return fmt.Errorf("message is not accessible")
	}

	// Split the callback data to get language and chatID
	parts := strings.Split(language, "_")
	if len(parts) < 2 {
		return fmt.Errorf("invalid callback data format")
	}

	lang := parts[0]
	chatIDStr := parts[1]

	// Parse the chat ID
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Safely access message fields
	var messageID int
	var queryChatID int64

	if message, ok := query.Message.(*telego.Message); ok {
		messageID = message.MessageID
		queryChatID = message.Chat.ID
	} else {
		return fmt.Errorf("can't access message fields")
	}

	// Get the group info
	groupInfo := GetGroupInfo(ctx.Context(), bot, chatID)

	// Verify the callback query sender is an admin
	senderIsAdmin, err := isUserAdmin(ctx.Context(), bot, chatID, query.From.ID)
	if err != nil || !senderIsAdmin {
		// Answer query silently
		_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "只有群组管理员才能更改设置。",
			ShowAlert:       true,
		})
		return err
	}

	// Validate language
	validLanguages := []string{models.LangSimplifiedChinese, models.LangTraditionalChinese, models.LangEnglish}
	isValid := false
	for _, validLang := range validLanguages {
		if lang == validLang {
			isValid = true
			break
		}
	}

	if !isValid {
		lang = models.LangSimplifiedChinese
	}

	// Get the old language for translations
	oldLang := groupInfo.Language
	if oldLang == "" {
		oldLang = models.LangSimplifiedChinese
	}

	// Update the language
	groupInfo.Language = lang
	UpdateGroupInfo(groupInfo)

	// Get language name for the response message
	languageName := models.GetLanguageName(lang, lang)

	// Create success message using the new language
	successMsg := fmt.Sprintf(models.GetTranslation(lang, "language_updated"), languageName)

	// Edit the message to show the selection
	_, _ = bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: queryChatID},
		MessageID: messageID,
		Text:      successMsg,
	})

	// Answer the callback query
	_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return nil
}

// getBotUsername gets the bot's username
func getBotUsername(ctx context.Context, bot *telego.Bot) (string, error) {
	botUser, err := bot.GetMe(ctx)
	if err != nil {
		return "", err
	}
	return botUser.Username, nil
}

// showGroupSelection displays a list of groups the admin manages
func showGroupSelection(ctx *th.Context, bot *telego.Bot, message telego.Message, action string) error {
	language := models.LangSimplifiedChinese

	// If database is enabled, get admin's groups
	if storage.DB != nil {
		// Get repository
		repo := storage.NewGroupRepository(storage.DB)

		// Get all groups
		groups, err := repo.GetAllGroupInfo()
		if err != nil {
			_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   "获取群组列表失败，请稍后再试。",
			})
			return err
		}

		// Filter groups where user is admin
		var adminGroups []*models.GroupInfo
		for _, group := range groups {
			isAdmin, err := isUserAdmin(ctx.Context(), bot, group.GroupID, message.From.ID)
			if err == nil && isAdmin {
				adminGroups = append(adminGroups, group)
			}
		}

		if len(adminGroups) == 0 {
			_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: message.Chat.ID},
				Text:   models.GetTranslation(language, "no_admin_groups"),
			})
			return err
		}

		// Create keyboard with group options
		var keyboard [][]telego.InlineKeyboardButton
		for _, group := range adminGroups {
			if group.Language != "" {
				language = group.Language
			}

			keyboard = append(keyboard, []telego.InlineKeyboardButton{
				{
					Text:         group.GroupName,
					CallbackData: fmt.Sprintf("group_%s_%d", action, group.GroupID),
				},
			})
		}

		msgText := models.GetTranslation(language, "select_group")
		_, err = bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: message.Chat.ID},
			Text:        msgText,
			ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
		})

		return err
	} else {
		// Database not enabled, ask for group ID
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s\n%s: %s",
				models.GetTranslation(language, "enter_group_id"),
				models.GetTranslation(language, "action"),
				action),
			ReplyMarkup: &telego.ForceReply{ForceReply: true, InputFieldPlaceholder: "123456789"},
		})
		return err
	}
}

// handleGroupSelection handles group selection from keyboard
func handleGroupSelection(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	if query.Message == nil {
		return fmt.Errorf("message is not accessible")
	}

	// Parse the callback data: group_action_groupID
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return fmt.Errorf("invalid callback data format")
	}

	action := parts[1]
	groupIDStr := parts[2]

	// Parse the group ID
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	// Get group info
	groupInfo := GetGroupInfo(ctx.Context(), bot, groupID)
	language := models.LangSimplifiedChinese
	if groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// Verify the user is an admin
	senderIsAdmin, err := isUserAdmin(ctx.Context(), bot, groupID, query.From.ID)
	if err != nil || !senderIsAdmin {
		// Answer query silently
		_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            models.GetTranslation(language, "user_not_admin"),
			ShowAlert:       true,
		})
		return err
	}

	// Get the message chat ID safely
	var messageChatID int64
	if message, ok := query.Message.(*telego.Message); ok {
		messageChatID = message.Chat.ID
	} else {
		return fmt.Errorf("can't access message chat ID")
	}

	// Process the action
	switch action {
	case "settings":
		// Show settings for the selected group
		if message, ok := query.Message.(*telego.Message); ok {
			err = showGroupSettings(ctx, bot, *message, groupID, language)
		} else {
			err = fmt.Errorf("can't access message")
		}
	case "toggle_premium":
		// Toggle premium user banning for the group
		return togglePremiumBanning(ctx, bot, groupID, messageChatID, language)
	case "toggle_cas":
		// Toggle CAS verification for the group
		return toggleCasVerification(ctx, bot, groupID, messageChatID, language)
	case "toggle_random_username":
		// Toggle random username check for the group
		return toggleRandomUsername(ctx, bot, groupID, messageChatID, language)
	case "toggle_emoji_name":
		// Toggle emoji username check for the group
		return toggleEmojiName(ctx, bot, groupID, messageChatID, language)
	case "toggle_bio_link":
		// Toggle bio link check for the group
		return toggleBioLink(ctx, bot, groupID, messageChatID, language)

	case "toggle_notifications":
		// Toggle admin notifications for the group
		return toggleNotifications(ctx, bot, groupID, messageChatID, language)
	case "language":
		// Show language options for the group
		return showLanguageOptions(ctx, bot, groupID, messageChatID, language)
	default:
		err = fmt.Errorf("unknown action: %s", action)
	}

	// Answer the callback query
	_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return err
}

// handleGroupIDInput handles the group ID input from user
func handleGroupIDInput(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	language := models.LangSimplifiedChinese

	// Get the action from the reply message
	replyText := message.ReplyToMessage.Text
	actionLine := strings.Split(replyText, "\n")[1]
	actionParts := strings.Split(actionLine, ": ")

	if len(actionParts) != 2 {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   models.GetTranslation(language, "invalid_format"),
		})
		return err
	}

	action := actionParts[1]

	// Parse the group ID
	groupID, err := strconv.ParseInt(message.Text, 10, 64)
	if err != nil {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   models.GetTranslation(language, "invalid_group_id"),
		})
		return err
	}

	// Verify the group exists and user is admin
	isAdmin, err := isUserAdmin(ctx.Context(), bot, groupID, message.From.ID)
	if err != nil {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   models.GetTranslation(language, "group_not_found"),
		})
		return err
	}

	if !isAdmin {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   models.GetTranslation(language, "user_not_admin"),
		})
		return err
	}

	// Get group info
	groupInfo := GetGroupInfo(ctx.Context(), bot, groupID)
	if groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// Process the action
	switch action {
	case "settings":
		// Show settings for the selected group
		return showGroupSettings(ctx, bot, message, groupID, language)
	case "toggle_premium":
		// Toggle premium user banning for the group
		return togglePremiumBanning(ctx, bot, groupID, message.Chat.ID, language)
	case "toggle_cas":
		// Toggle CAS verification for the group
		return toggleCasVerification(ctx, bot, groupID, message.Chat.ID, language)
	case "toggle_notifications":
		// Toggle admin notifications for the group
		return toggleNotifications(ctx, bot, groupID, message.Chat.ID, language)
	case "language":
		// Show language options for the group
		return showLanguageOptions(ctx, bot, groupID, message.Chat.ID, language)
	default:
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   models.GetTranslation(language, "unknown_action"),
		})
		return err
	}
}

// handleActionCallback handles action callbacks
func handleActionCallback(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	if query.Message == nil {
		return fmt.Errorf("message is not accessible")
	}

	// Parse the callback data: action_type_groupID
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return fmt.Errorf("invalid callback data format")
	}

	actionType := parts[1]
	groupIDStr := parts[2]

	// Parse the group ID
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	// Get group info
	groupInfo := GetGroupInfo(ctx.Context(), bot, groupID)
	language := models.LangSimplifiedChinese
	if groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// Verify the user is an admin
	senderIsAdmin, err := isUserAdmin(ctx.Context(), bot, groupID, query.From.ID)
	if err != nil || !senderIsAdmin {
		// Answer query silently
		_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            models.GetTranslation(language, "user_not_admin"),
			ShowAlert:       true,
		})
		return err
	}

	// Get the message and messageID safely
	var messageID int
	var chatID int64
	if message, ok := query.Message.(*telego.Message); ok {
		messageID = message.MessageID
		chatID = message.Chat.ID
	} else {
		return fmt.Errorf("can't access message details")
	}

	// Process the action
	switch actionType {
	case "done":
		// Mark action as done
		_, _ = bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      models.GetTranslation(language, "action_completed"),
		})
	default:
		err = fmt.Errorf("unknown action type: %s", actionType)
	}

	// Answer the callback query
	_ = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return err
}
