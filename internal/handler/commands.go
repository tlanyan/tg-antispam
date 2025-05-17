package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
	"tg-antispam/internal/service"
	"tg-antispam/internal/storage"
)

// registers all bot command handlers
func RegisterCommands(ctx *th.Context, bot *telego.Bot, message telego.Message) (bool, error) {
	// Skip non-command messages
	if !strings.HasPrefix(message.Text, "/") {
		return false, nil
	}

	switch message.Text {
	case "/help", "/start", "/start help":
		return true, sendHelpMessage(ctx, bot, message)
	case "/settings":
		return true, handleSettingsCommand(ctx, bot, message)
	case "/toggle_premium":
		return true, handleToggleCommand(ctx, bot, message, "toggle_premium")
	case "/toggle_cas":
		return true, handleToggleCommand(ctx, bot, message, "toggle_cas")
	case "/toggle_random_username":
		return true, handleToggleCommand(ctx, bot, message, "toggle_random_username")
	case "/toggle_emoji_name":
		return true, handleToggleCommand(ctx, bot, message, "toggle_emoji_name")
	case "/toggle_bio_link":
		return true, handleToggleCommand(ctx, bot, message, "toggle_bio_link")
	case "/toggle_notifications":
		return true, handleToggleCommand(ctx, bot, message, "toggle_notifications")
	case "/language_group":
		return true, handleToggleCommand(ctx, bot, message, "language_group")
	case "/language":
		return true, handleLanguageCommand(ctx, bot, message)
	case "/self_unban":
		return true, handleSelfUnbanCommand(ctx, bot, message)
	}

	if message.Chat.Type == "private" && message.ReplyToMessage != nil {
		// Check if the message is a reply to our "enter group ID" message
		if message.ReplyToMessage.From.ID == bot.ID() &&
			(strings.Contains(message.ReplyToMessage.Text, "请输入群组ID") ||
				strings.Contains(message.ReplyToMessage.Text, "請輸入群組ID") ||
				strings.Contains(message.ReplyToMessage.Text, "Please enter the Group ID")) {
			return true, handleGroupIDInput(ctx, bot, message)
		}
	}
	return false, nil
}

func handleLanguageCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	groupInfo := service.GetGroupInfo(ctx.Context(), bot, message.Chat.ID, true)
	// current chat
	if groupInfo.AdminID == -1 {
		groupInfo.AdminID = message.From.ID
		groupInfo.IsAdmin = true
		service.UpdateGroupInfo(groupInfo)
	}
	query := telego.CallbackQuery{
		ID:      "",            // 这里没有实际的回调ID，但函数调用中不会用到
		From:    *message.From, // 解引用指针，获取用户对象
		Message: &message,
	}
	return showLanguageSelection(ctx, bot, query, message.Chat.ID, groupInfo.Language)
}

// sendHelpMessage sends help information based on chat type and language
func sendHelpMessage(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	language := GetBotChatLang(ctx, bot, message.From.ID, message.Chat.ID)

	var helpText string
	if message.Chat.Type == "private" {
		helpText = fmt.Sprintf("<b>%s</b>\n\n%s\n\n<b>%s</b>\n%s\n%s\n%s\n\n<b>%s</b>\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n\n<b>%s</b>",
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
	language := GetBotChatLang(ctx, bot, message.From.ID, message.Chat.ID)

	logger.Debugf("handleSettingsCommand called for message: %+v", message)
	if message.Chat.Type == "private" {
		return showGroupSelection(ctx, bot, message, "settings")
	} else {
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

// handleToggleCommand is a generic handler for all toggle commands
func handleToggleCommand(ctx *th.Context, bot *telego.Bot, message telego.Message, action string) error {
	logger.Infof("handleToggleCommand called, action: %s, for message: %+v", action, message)
	if message.Chat.Type == "private" {
		return showGroupSelection(ctx, bot, message, action)
	} else {
		language := GetBotChatLang(ctx, bot, message.From.ID, message.Chat.ID)

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

// handleGroupIDInput processes user input when adding a group by ID
func handleGroupIDInput(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	logger.Debugf("handleGroupIDInput called for message: %+v", message)
	// Get the ID from the message
	groupID, err := strconv.ParseInt(strings.TrimSpace(message.Text), 10, 64)
	if err != nil {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "无效的群组ID，请输入数字ID",
			ParseMode: "HTML",
		})
		return err
	}

	logger.Infof("handleGroupIDInput called for groupID=%d", groupID)
	chatInfo, err := bot.GetChat(ctx.Context(), &telego.GetChatParams{
		ChatID: telego.ChatID{ID: groupID},
	})
	if err != nil {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "无法获取群组信息，请确保机器人已经加入该群组，并且您输入了正确的群组ID",
			ParseMode: "HTML",
		})
		return err
	}

	// Check if the chat is a group or supergroup
	if chatInfo.Type != "group" && chatInfo.Type != "supergroup" {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "ID不是群组，请输入正确的群组ID",
			ParseMode: "HTML",
		})
		return err
	}

	// Check if the bot is an admin in the group
	botAdmins, err := bot.GetChatAdministrators(ctx.Context(), &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: groupID},
	})
	if err != nil {
		logger.Warningf("Error getting chat administrators: %v", err)
	}

	botID := bot.ID()
	botIsAdmin := false
	for _, admin := range botAdmins {
		if admin.MemberUser().ID == botID {
			botIsAdmin = true
			break
		}
	}

	if !botIsAdmin {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "机器人不是群组管理员，请先将机器人设为管理员",
			ParseMode: "HTML",
		})
		return err
	}

	isAdmin, err := isUserAdmin(ctx.Context(), bot, groupID, message.From.ID)
	if err != nil || !isAdmin {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "您不是该群组的管理员，无法管理该群组",
			ParseMode: "HTML",
		})
		return err
	}

	groupInfo := service.GetGroupInfo(ctx.Context(), bot, groupID, true)
	groupInfo.AdminID = message.From.ID
	service.UpdateGroupInfo(groupInfo)

	_, err = bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: message.Chat.ID},
		Text: fmt.Sprintf("✅ 成功添加群组: <b>%s</b>\n\n请使用 /settings 命令管理该群组",
			chatInfo.Title),
		ParseMode: "HTML",
	})

	return err
}

// showGroupSelection displays a list of groups for the user to select from
func showGroupSelection(ctx *th.Context, bot *telego.Bot, message telego.Message, action string) error {
	logger.Infof("showGroupSelection called, action: %s, message: %+v", action, message)
	userID := message.From.ID
	language := GetBotChatLang(ctx, bot, message.From.ID, message.Chat.ID)

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
		msg, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "empty_group_list"),
			ParseMode: "HTML",
		})
		if err != nil {
			logger.Warningf("Error sending message: %v", err)
		}
		logger.Infof("Sent message: %+v", msg)
		return err
	}

	// 如果只有一个群组，直接执行动作，无需显示选择菜单
	if len(adminGroups) == 1 {
		group := adminGroups[0]
		logger.Infof("Only one group found, directly executing action %s for group %d", action, group.GroupID)

		switch action {
		case "settings":
			return showGroupSettings(ctx, bot, message, group.GroupID, language)
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
			return HandleCallbackQuery(ctx, bot, query)
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
	msg, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
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
func showGroupSettings(ctx *th.Context, bot *telego.Bot, message telego.Message, groupID int64, language string) error {
	logger.Infof("showGroupSettings called for message: %+v, groupID=%d", message, groupID)
	// 获取群组信息
	groupInfo := service.GetGroupInfo(ctx.Context(), bot, groupID, false)
	if groupInfo == nil || !groupInfo.IsAdmin {
		// 机器人不是管理员
		msg, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "bot_not_admin"),
			ParseMode: "HTML",
		})
		if err != nil {
			logger.Warningf("Error sending bot not admin message: %v", err)
		}
		logger.Infof("Sent bot not admin message: %+v", msg)
		return err
	}

	settingsText, keyboard := buildGroupSettingsMessageParts(groupInfo, language, groupID)

	// 发送设置消息
	msg, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
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

// handleSelfUnbanCommand guides a user through self-unban based on their ban records
func handleSelfUnbanCommand(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	userID := message.From.ID
	records, err := service.GetActiveBanRecordsByUser(userID)
	if err != nil {
		logger.Warningf("Error fetching ban records for user %d: %v", userID, err)
		return nil
	}
	if len(records) == 0 {
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "您没有待解封的记录。\nYou have no ban records to unban.",
			ParseMode: "HTML",
		})
		return err
	}
	if len(records) == 1 {
		// Directly start math verification for the single ban record
		return SendMathVerificationMessage(ctx, bot, userID, records[0].GroupID, nil)
	}
	// Multiple records: ask user to choose which group to unban
	var buttons [][]telego.InlineKeyboardButton
	for _, rec := range records {
		grp := service.GetGroupInfo(ctx.Context(), bot, rec.GroupID, false)
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
	_, err = bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: message.Chat.ID},
		Text:        "请选择要解除限制的群组：\nPlease select the group to unban from:",
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
	return err
}
