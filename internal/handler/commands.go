package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"tg-antispam/internal/crash"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
	"tg-antispam/internal/service"
	"tg-antispam/internal/storage"
)

// registers all bot command handlers
func RegisterCommands(bot *telego.Bot, message telego.Message) (bool, error) {
	// Skip non-command messages
	if !strings.HasPrefix(message.Text, "/") {
		return false, nil
	}

	command := message.Text
	if strings.Contains(command, "@"+bot.Username()) {
		command = strings.TrimSuffix(command, "@"+bot.Username())
	}

	switch command {
	case "/help", "/start", "/start help":
		return true, sendHelpMessage(bot, message)
	case "/settings":
		return true, handleSettingsCommand(bot, message)
	case "/toggle_premium":
		return true, handleToggleCommand(bot, message, "toggle_premium")
	case "/toggle_cas":
		return true, handleToggleCommand(bot, message, "toggle_cas")
	case "/toggle_random_username":
		return true, handleToggleCommand(bot, message, "toggle_random_username")
	case "/toggle_emoji_name":
		return true, handleToggleCommand(bot, message, "toggle_emoji_name")
	case "/toggle_bio_link":
		return true, handleToggleCommand(bot, message, "toggle_bio_link")
	case "/toggle_notifications":
		return true, handleToggleCommand(bot, message, "toggle_notifications")
	case "/language_group":
		return true, handleToggleCommand(bot, message, "language_group")
	case "/language":
		return true, handleLanguageCommand(bot, message)
	case "/self_unban":
		return true, handleSelfUnbanCommand(bot, message)
	case "/ping":
		return true, handlePingCommand(bot, message)
	}

	return false, nil
}

func handlePingCommand(bot *telego.Bot, message telego.Message) error {
	_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text:   "pong",
	})
	return err
}

func checkAdminMessage(bot *telego.Bot, message telego.Message) error {
	// Check if sender is admin
	if !isUserAdmin(bot, message.Chat.ID, message.From.ID) {
		groupInfo := service.GetGroupInfo(bot, message.Chat.ID, true)
		if groupInfo.AdminID == -1 {
			service.UpdateGroupInfo(groupInfo)
		}
		language := groupInfo.Language
		botUsername, _ := getBotUsername(bot)
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text: fmt.Sprintf("%s @%s",
				models.GetTranslation(language, "user_not_admin"),
				botUsername),
		})
		return err
	}
	return nil
}

func handleLanguageCommand(bot *telego.Bot, message telego.Message) error {
	groupInfo := service.GetGroupInfo(bot, message.Chat.ID, true)
	if message.Chat.Type == "private" {
		// current chat
		if groupInfo.AdminID == -1 {
			groupInfo.AdminID = message.From.ID
			groupInfo.IsAdmin = true
			service.UpdateGroupInfo(groupInfo)
		}
	} else {
		err := checkAdminMessage(bot, message)
		if err != nil {
			return err
		}
	}
	query := telego.CallbackQuery{
		ID:      "",            // 这里没有实际的回调ID，但函数调用中不会用到
		From:    *message.From, // 解引用指针，获取用户对象
		Message: &message,
	}
	return showLanguageSelection(bot, query, message.Chat.ID, groupInfo.Language)
}

// sendHelpMessage sends help information based on chat type and language
func sendHelpMessage(bot *telego.Bot, message telego.Message) error {
	language := GetBotLang(bot, message)

	helpText := fmt.Sprintf("<b>%s</b>\n\n%s\n\n<b>%s</b>\n%s\n%s\n%s\n\n<b>%s</b>\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n\n<b>%s</b>",
		models.GetTranslation(language, "help_title"),
		models.GetTranslation(language, "help_description"),
		models.GetTranslation(language, "general_commands"),
		models.GetTranslation(language, "help_cmd_help"),
		models.GetTranslation(language, "help_cmd_self_unban"),
		models.GetTranslation(language, "help_cmd_language"),

		models.GetTranslation(language, "settings_commands"),
		models.GetTranslation(language, "help_cmd_settings"),
		models.GetTranslation(language, "help_cmd_toggle_premium"),
		models.GetTranslation(language, "help_cmd_toggle_cas"),
		models.GetTranslation(language, "help_cmd_toggle_random_username"),
		models.GetTranslation(language, "help_cmd_toggle_emoji_name"),
		models.GetTranslation(language, "help_cmd_toggle_bio_link"),
		models.GetTranslation(language, "help_cmd_toggle_notifications"),
		models.GetTranslation(language, "help_cmd_language_group"),
		models.GetTranslation(language, "help_note"),
	)

	_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      helpText,
		ParseMode: "HTML",
	})

	return err
}

// handleSettingsCommand handles the /settings command
func handleSettingsCommand(bot *telego.Bot, message telego.Message) error {
	logger.Debugf("handleSettingsCommand called for message: %+v", message)
	if message.Chat.Type == "private" {
		return showGroupSelection(bot, message, "settings")
	} else {
		err := checkAdminMessage(bot, message)
		if err != nil {
			return err
		}
		return showGroupSettings(bot, message, message.Chat.ID)
	}
}

// handleToggleCommand is a generic handler for all toggle commands
func handleToggleCommand(bot *telego.Bot, message telego.Message, action string) error {
	logger.Infof("handleToggleCommand called, action: %s, for message: %+v", action, message)
	if message.Chat.Type == "private" {
		return showGroupSelection(bot, message, action)
	} else {
		err := checkAdminMessage(bot, message)
		if err != nil {
			return err
		}
		callbackData := fmt.Sprintf("action:%s:%d", action, message.Chat.ID)
		query := telego.CallbackQuery{
			ID:      "",
			From:    *message.From,
			Data:    callbackData,
			Message: &message,
		}
		return HandleCallbackQuery(bot, query)
	}
}

// showGroupSelection displays a list of groups for the user to select from
func showGroupSelection(bot *telego.Bot, message telego.Message, action string) error {
	logger.Infof("showGroupSelection called, action: %s, message: %+v", action, message)
	userID := message.From.ID
	language := GetBotLang(bot, message)

	var adminGroups []*models.GroupInfo
	var err error

	// 检查数据库是否已启用
	if storage.DB != nil {
		// 获取用户作为管理员的群组
		groupRepo := storage.NewGroupRepository(storage.DB)
		adminGroups, err = groupRepo.GetGroupsByAdminID(userID)
		if err != nil {
			logger.Warningf("Error getting admin groups: %v", err)
			return err
		}
	}

	// 如果没有找到群组，提示用户输入群组ID
	if len(adminGroups) == 0 {
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "empty_group_list"),
			ParseMode: "HTML",
		})
		if err != nil {
			logger.Warningf("Error sending message: %v", err)
		}
		return err
	}

	// 如果只有一个群组，直接执行动作，无需显示选择菜单
	if len(adminGroups) == 1 {
		group := adminGroups[0]
		logger.Infof("Only one group found, directly executing action %s for group %d", action, group.GroupID)

		switch action {
		case "settings":
			return showGroupSettings(bot, message, group.GroupID)
		case "toggle_premium", "toggle_cas", "toggle_random_username", "toggle_emoji_name", "toggle_bio_link", "toggle_notifications", "language_group":
			// 模拟回调数据处理，创建一个回调查询对象
			callbackData := fmt.Sprintf("action:%s:%d", action, group.GroupID)
			query := telego.CallbackQuery{
				ID:      "",            // 这里没有实际的回调ID，但函数调用中不会用到
				From:    *message.From, // 解引用指针，获取用户对象
				Data:    callbackData,
				Message: &message,
			}
			// 调用处理回调的函数
			return HandleCallbackQuery(bot, query)
		}
	}

	// 创建一个群组选择的内联键盘
	var rows [][]telego.InlineKeyboardButton
	for _, group := range adminGroups {
		// 截断群组名称，避免太长
		groupName := group.GroupName
		if len(groupName) > 30 {
			groupName = groupName[:27] + "..."
		}

		rows = append(rows, []telego.InlineKeyboardButton{
			{
				Text:         groupName,
				CallbackData: fmt.Sprintf("group:%s:%d", action, group.GroupID),
			},
		})
	}

	// 发送选择消息
	selectText := models.GetTranslation(language, "select_group")
	msg, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: message.Chat.ID},
		Text:        selectText,
		ParseMode:   "HTML",
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
	if err != nil {
		logger.Warningf("Error sending group selection message: %v", err)
	}
	logger.Infof("Sent selection message: %+v", msg)

	return err
}

// getBoolStatusText returns "enabled" or "disabled" based on a boolean value
func getBoolStatusText(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

// buildGroupSettingsMessageParts constructs the settings message text and keyboard
func buildGroupSettingsMessageParts(groupInfo *models.GroupInfo, language string, groupID int64) (string, [][]telego.InlineKeyboardButton) {
	// 构建设置消息
	settingsText := fmt.Sprintf("<b>%s</b>\n\n<b>%s</b> %s\n\n<b>%s</b>\n",
		fmt.Sprintf(models.GetTranslation(language, "settings_title"), groupInfo.GroupName),
		models.GetTranslation(language, "settings_bot_status"),
		models.GetTranslation(language, "settings_active"),
		models.GetTranslation(language, "settings_current"),
	)

	// 添加当前设置
	premiumStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanPremium))
	casStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.EnableCAS))
	randomUsernameStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanRandomUsername))
	emojiNameStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanEmojiName))
	bioLinkStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanBioLink))
	notificationsStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.EnableNotification))
	langName := getLanguageName(groupInfo.Language)

	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_ban_premium"), premiumStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_cas"), casStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_random_username"), randomUsernameStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_emoji_name"), emojiNameStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_bio_link"), bioLinkStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_notifications"), notificationsStatus) + "\n"
	settingsText += fmt.Sprintf(models.GetTranslation(language, "settings_language"), langName) + "\n"

	// 创建设置按钮
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         models.GetTranslation(language, "toggle_premium"),
				CallbackData: fmt.Sprintf("action:toggle_premium:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_cas"),
				CallbackData: fmt.Sprintf("action:toggle_cas:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_random_username"),
				CallbackData: fmt.Sprintf("action:toggle_random_username:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_emoji_name"),
				CallbackData: fmt.Sprintf("action:toggle_emoji_name:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_bio_link"),
				CallbackData: fmt.Sprintf("action:toggle_bio_link:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_notifications"),
				CallbackData: fmt.Sprintf("action:toggle_notifications:%d", groupID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "change_language"),
				CallbackData: fmt.Sprintf("action:language_group:%d", groupID),
			},
		},
	}
	return settingsText, keyboard
}

// showGroupSettings displays the settings for a group
func showGroupSettings(bot *telego.Bot, message telego.Message, groupID int64) error {
	// 获取群组信息
	groupInfo := service.GetGroupInfo(bot, groupID, true)
	if groupInfo.AdminID == -1 {
		groupInfo.AdminID = message.From.ID
		service.UpdateGroupInfo(groupInfo)
	}

	if !groupInfo.IsAdmin {
		// 机器人不是管理员
		msg, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(groupInfo.Language, "bot_not_admin"),
			ParseMode: "HTML",
		})
		if err != nil {
			logger.Warningf("Error sending bot not admin message: %v", err)
		}
		logger.Infof("Sent bot not admin message: %+v", msg)
		return err
	}

	settingsText, keyboard := buildGroupSettingsMessageParts(groupInfo, groupInfo.Language, groupID)

	// 发送设置消息
	msg, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: message.Chat.ID},
		Text:        settingsText,
		ParseMode:   "HTML",
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		logger.Warningf("Error sending settings message: %v", err)
	}
	logger.Infof("Sent settings message: %+v", msg)

	return err
}

func PrivateChatWarning(bot *telego.Bot, message telego.Message) error {
	language := GetBotLang(bot, message)
	msg, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: message.Chat.ID},
		Text:      models.GetTranslation(language, "use_private_chat"),
		ParseMode: "HTML",
	})
	if err != nil {
		logger.Warningf("Error sending use private chat message: %v", err)
	} else {
		crash.SafeGoroutine("private-chat-warning-cleanup", func() {
			time.Sleep(time.Minute * 3)
			bot.DeleteMessage(context.Background(), &telego.DeleteMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				MessageID: msg.MessageID,
			})
		})
	}
	return err
}

// handleSelfUnbanCommand guides a user through self-unban based on their ban records
func handleSelfUnbanCommand(bot *telego.Bot, message telego.Message) error {
	if message.Chat.Type != "private" {
		return PrivateChatWarning(bot, message)
	}

	language := GetBotLang(bot, message)
	userID := message.From.ID
	records, err := service.GetUserActiveBanRecords(userID, -1)
	if err != nil {
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "get_ban_records_error"),
			ParseMode: "HTML",
		})
		return err
	}
	if len(records) == 0 {
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "no_ban_records"),
			ParseMode: "HTML",
		})
		return err
	}
	if len(records) == 1 {
		// Directly start math verification for the single ban record
		query := telego.CallbackQuery{
			ID:      "",
			From:    *message.From,
			Data:    "",
			Message: &message,
		}
		return SendMathVerificationMessage(bot, userID, records[0].GroupID, &query)
	}
	// Multiple records: ask user to choose which group to unban
	var buttons [][]telego.InlineKeyboardButton
	for _, rec := range records {
		grp := service.GetGroupInfo(bot, rec.GroupID, false)
		if grp == nil {
			continue
		}
		label := fmt.Sprintf("%d", rec.GroupID)
		if grp != nil && grp.GroupName != "" {
			label = grp.GroupName
		}
		btn := telego.InlineKeyboardButton{
			Text:         label,
			CallbackData: fmt.Sprintf("self_unban:%d:%d", rec.GroupID, userID),
		}
		buttons = append(buttons, []telego.InlineKeyboardButton{btn})
	}
	markup := &telego.InlineKeyboardMarkup{InlineKeyboard: buttons}
	_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: message.Chat.ID},
		Text:        models.GetTranslation(language, "select_group_to_unban"),
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	return err
}
