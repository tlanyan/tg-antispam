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
)

// HandleCallbackQuery processes callback queries from inline keyboards
func HandleCallbackQuery(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	// Skip if no data
	if query.Data == "" {
		return nil
	}

	logger.Infof("Received callback query: %s", query.Data)

	// Handle different callback types based on prefix
	if strings.HasPrefix(query.Data, "unban:") {
		return handleUnbanCallback(ctx, bot, query)
	} else if strings.HasPrefix(query.Data, "lang:") {
		return handleLanguageCallback(ctx, bot, query)
	} else if strings.HasPrefix(query.Data, "group:") {
		return handleGroupSelectionCallback(ctx, bot, query)
	} else if strings.HasPrefix(query.Data, "action:") {
		return handleActionSelectionCallback(ctx, bot, query)
	}

	return nil
}

// handleUnbanCallback processes a request to unban a user
func handleUnbanCallback(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	// Parse the callback data: unban:chatID:userID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		logger.Warningf("Invalid callback data in unban callback: %s", parts)
		return nil
	}

	// Parse chat ID and user ID
	chatID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		logger.Warningf("Invalid chat ID in unban callback: %v", err)
		return nil
	}

	userID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		logger.Warningf("Invalid user ID in unban callback: %v", err)
		return nil
	}

	// Check if the callback sender is an admin in the chat
	isAdmin, err := isUserAdmin(ctx.Context(), bot, chatID, query.From.ID)
	if err != nil || !isAdmin {
		// Inform user they don't have permission
		err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "You don't have permission to unban users.",
			ShowAlert:       true,
		})
		return err
	}

	// Unrestrict the user
	UnrestrictUser(ctx.Context(), bot, chatID, userID)

	// Get group info for language
	groupInfo := service.GetGroupInfo(ctx.Context(), bot, chatID)
	language := models.LangSimplifiedChinese
	if groupInfo != nil && groupInfo.Language != "" {
		language = groupInfo.Language
	}

	// Notify the admin that the action was successful
	err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            models.GetTranslation(language, "user_unrestricted"),
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// Update the message to reflect that the user was unbanned
	if query.Message != nil {
		if accessibleMsg, ok := query.Message.(*telego.Message); ok {
			_, editErr := bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: accessibleMsg.Chat.ID},
				MessageID: accessibleMsg.MessageID,
				Text:      accessibleMsg.Text + "\n\n" + models.GetTranslation(language, "user_unrestricted"),
				ParseMode: "HTML",
			})
			if editErr != nil {
				logger.Warningf("Error editing message: %v", editErr)
			}
		}
	}

	return err
}

// handleLanguageCallback processes language selection callbacks
func handleLanguageCallback(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	// Format: lang:language:chatID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		logger.Warningf("Invalid callback data in language callback: %s", parts)
		return nil
	}

	language := parts[1]
	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		logger.Warningf("Invalid chat ID in language callback: %v", err)
		return nil
	}

	// 直接传递变量给setLanguage函数处理
	return setLanguage(ctx, bot, query, chatID, language)
}

// setLanguage updates the language setting for a group
func setLanguage(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery, chatID int64, language string) error {
	// Get the group info
	groupInfo := service.GetGroupInfo(ctx.Context(), bot, chatID)

	// Check if the user is an admin
	if groupInfo.AdminID != query.From.ID {
		isAdmin, err := isUserAdmin(ctx.Context(), bot, chatID, query.From.ID)
		if err != nil || !isAdmin {
			// Inform user they don't have permission
			err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "You don't have permission to change settings.",
				ShowAlert:       true,
			})
			return err
		}
	}

	// Update the language
	groupInfo.Language = language
	service.UpdateGroupInfo(groupInfo)

	// Notify about the change
	err := bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            models.GetTranslation(language, "language_updated"),
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// Update the settings message
	currentLang := models.GetTranslation(language, "current_language")
	welcomeText := models.GetTranslation(language, "welcome_to_settings")
	currentSettings := models.GetTranslation(language, "current_settings")

	// Build the settings message
	settingsText := fmt.Sprintf("<b>%s</b>\n\n<b>%s:</b>\n%s: %s\n",
		welcomeText,
		currentSettings,
		currentLang,
		getLanguageName(language),
	)

	// Add other settings
	premiumStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanPremium))
	casStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.EnableCAS))
	randomUsernameStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanRandomUsername))
	emojiNameStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanEmojiName))
	bioLinkStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.BanBioLink))
	notificationStatus := models.GetTranslation(language, getBoolStatusText(groupInfo.EnableNotification))

	settingsText += models.GetTranslation(language, "ban_premium") + ": " + premiumStatus + "\n"
	settingsText += models.GetTranslation(language, "use_cas") + ": " + casStatus + "\n"
	settingsText += models.GetTranslation(language, "ban_random_username") + ": " + randomUsernameStatus + "\n"
	settingsText += models.GetTranslation(language, "ban_emoji_name") + ": " + emojiNameStatus + "\n"
	settingsText += models.GetTranslation(language, "ban_bio_link") + ": " + bioLinkStatus + "\n"
	settingsText += models.GetTranslation(language, "enable_notifications") + ": " + notificationStatus

	// Create an inline keyboard for settings
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         models.GetTranslation(language, "toggle_premium"),
				CallbackData: fmt.Sprintf("action:toggle_premium:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_cas"),
				CallbackData: fmt.Sprintf("action:toggle_cas:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_random_username"),
				CallbackData: fmt.Sprintf("action:toggle_random_username:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_emoji_name"),
				CallbackData: fmt.Sprintf("action:toggle_emoji_name:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_bio_link"),
				CallbackData: fmt.Sprintf("action:toggle_bio_link:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "toggle_notifications"),
				CallbackData: fmt.Sprintf("action:toggle_notifications:%d", chatID),
			},
		},
		{
			{
				Text:         models.GetTranslation(language, "change_language"),
				CallbackData: fmt.Sprintf("action:language:%d", chatID),
			},
		},
	}

	// Update the message
	if query.Message != nil {
		if accessibleMsg, ok := query.Message.(*telego.Message); ok {
			_, editErr := bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: accessibleMsg.Chat.ID},
				MessageID:   accessibleMsg.MessageID,
				Text:        settingsText,
				ParseMode:   "HTML",
				ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
			})
			if editErr != nil {
				logger.Warningf("Error editing settings message: %v", editErr)
			}
		}
	}

	return err
}

// getBoolStatusText returns "enabled" or "disabled" based on a boolean value
func getBoolStatusText(value bool) string {
	if value {
		return "status_enabled"
	}
	return "status_disabled"
}

// getLanguageName returns the display name of a language code
func getLanguageName(langCode string) string {
	switch langCode {
	case models.LangSimplifiedChinese:
		return "简体中文"
	case models.LangTraditionalChinese:
		return "繁體中文"
	case models.LangEnglish:
		return "English"
	default:
		return "Unknown"
	}
}

// 处理群组选择回调
func handleGroupSelectionCallback(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	// 提取群组ID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		return nil
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		logger.Warningf("Invalid group ID in callback: %v", err)
		return nil
	}

	action := parts[2]
	logger.Infof("Group selection callback received: group=%d, action=%s", groupID, action)

	// 通知用户已收到请求
	err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "正在处理您的请求...",
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// 处理"添加群组"操作
	if groupID == 0 && action == "add" {
		// 使用Reply标记来提示用户输入群组ID
		language := models.LangSimplifiedChinese
		selectText := models.GetTranslation(language, "enter_group_id")

		if query.Message != nil {
			if message, ok := query.Message.(*telego.Message); ok {
				// 删除现有的inline键盘
				_, err := bot.EditMessageReplyMarkup(ctx.Context(), &telego.EditMessageReplyMarkupParams{
					ChatID:      telego.ChatID{ID: message.Chat.ID},
					MessageID:   message.MessageID,
					ReplyMarkup: &telego.InlineKeyboardMarkup{},
				})
				if err != nil {
					logger.Warningf("Error removing inline keyboard: %v", err)
				}

				// 发送新消息，带有强制回复
				_, err = bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
					ChatID:      telego.ChatID{ID: message.Chat.ID},
					Text:        selectText,
					ParseMode:   "HTML",
					ReplyMarkup: &telego.ForceReply{ForceReply: true, InputFieldPlaceholder: "-1001234567890"},
				})
				if err != nil {
					logger.Warningf("Error sending group ID input message: %v", err)
				}
			}
		}
		return nil
	}

	// 根据操作类型处理
	if query.Message != nil {
		if message, ok := query.Message.(*telego.Message); ok {
			// 获取用户语言设置
			language := models.LangSimplifiedChinese
			groupInfo := service.GetGroupInfo(ctx.Context(), bot, groupID)
			if groupInfo != nil && groupInfo.Language != "" {
				language = groupInfo.Language
			}

			switch action {
			case "settings":
				// 显示群组设置
				return showGroupSettings(ctx, bot, *message, groupID, language)
			default:
				// 对于其他操作类型，创建action回调
				callbackData := fmt.Sprintf("action:%s:%d", action, groupID)
				actionQuery := telego.CallbackQuery{
					ID:      query.ID,
					From:    query.From,
					Data:    callbackData,
					Message: query.Message,
				}
				return handleActionSelectionCallback(ctx, bot, actionQuery)
			}
		}
	}

	return nil
}

// 处理设置项操作回调
func handleActionSelectionCallback(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery) error {
	// 提取操作和群组ID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		return nil
	}

	action := parts[1]
	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		logger.Warningf("Invalid group ID in action callback: %v", err)
		return nil
	}

	logger.Infof("Action selection callback received: action=%s, group=%d", action, groupID)

	// 通知用户已收到请求
	err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "正在处理您的请求...",
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// 获取群组信息
	groupInfo := service.GetGroupInfo(ctx.Context(), bot, groupID)
	if groupInfo == nil {
		logger.Warningf("Group info not found: %d", groupID)
		return nil
	}

	// 检查用户是否是管理员
	if groupInfo.AdminID != query.From.ID {
		isAdmin, err := isUserAdmin(ctx.Context(), bot, groupID, query.From.ID)
		if err != nil || !isAdmin {
			// 通知用户没有权限
			err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "您没有权限更改设置",
				ShowAlert:       true,
			})
			return err
		}
	}

	// 处理不同的操作
	language := groupInfo.Language
	if language == "" {
		language = models.LangSimplifiedChinese
	}

	var updateMessage string

	switch action {
	case "toggle_premium":
		// 切换Premium用户封禁设置
		groupInfo.BanPremium = !groupInfo.BanPremium
		if groupInfo.BanPremium {
			updateMessage = models.GetTranslation(language, "premium_ban_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "premium_ban_disabled")
		}

	case "toggle_cas":
		// 切换CAS验证设置
		groupInfo.EnableCAS = !groupInfo.EnableCAS
		if groupInfo.EnableCAS {
			updateMessage = models.GetTranslation(language, "cas_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "cas_disabled")
		}

	case "toggle_random_username":
		// 切换随机用户名封禁设置
		groupInfo.BanRandomUsername = !groupInfo.BanRandomUsername
		if groupInfo.BanRandomUsername {
			updateMessage = models.GetTranslation(language, "random_username_ban_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "random_username_ban_disabled")
		}

	case "toggle_emoji_name":
		// 切换表情名字封禁设置
		groupInfo.BanEmojiName = !groupInfo.BanEmojiName
		if groupInfo.BanEmojiName {
			updateMessage = models.GetTranslation(language, "emoji_name_ban_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "emoji_name_ban_disabled")
		}

	case "toggle_bio_link":
		// 切换简介链接封禁设置
		groupInfo.BanBioLink = !groupInfo.BanBioLink
		if groupInfo.BanBioLink {
			updateMessage = models.GetTranslation(language, "bio_link_ban_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "bio_link_ban_disabled")
		}

	case "toggle_notifications":
		// 切换通知设置
		groupInfo.EnableNotification = !groupInfo.EnableNotification
		if groupInfo.EnableNotification {
			updateMessage = models.GetTranslation(language, "notifications_enabled")
		} else {
			updateMessage = models.GetTranslation(language, "notifications_disabled")
		}

	case "language":
		// 显示语言选择界面
		return showLanguageSelection(ctx, bot, query, groupID, language)
	}

	// 保存群组设置
	service.UpdateGroupInfo(groupInfo)

	// 通知用户设置已更新
	if updateMessage != "" {
		err = bot.AnswerCallbackQuery(ctx.Context(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            updateMessage,
		})
		if err != nil {
			logger.Warningf("Error answering callback query: %v", err)
		}
	}

	// 更新设置消息
	if query.Message != nil {
		if message, ok := query.Message.(*telego.Message); ok {
			return showGroupSettings(ctx, bot, *message, groupID, language)
		}
	}

	return nil
}

// showLanguageSelection displays language selection options
func showLanguageSelection(ctx *th.Context, bot *telego.Bot, query telego.CallbackQuery, groupID int64, language string) error {
	// 创建语言选择键盘
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{
				Text:         "简体中文",
				CallbackData: fmt.Sprintf("lang:%s:%d", models.LangSimplifiedChinese, groupID),
			},
		},
		{
			{
				Text:         "繁體中文",
				CallbackData: fmt.Sprintf("lang:%s:%d", models.LangTraditionalChinese, groupID),
			},
		},
		{
			{
				Text:         "English",
				CallbackData: fmt.Sprintf("lang:%s:%d", models.LangEnglish, groupID),
			},
		},
	}

	// 发送或更新消息
	selectText := models.GetTranslation(language, "select_language")

	if query.Message != nil {
		if message, ok := query.Message.(*telego.Message); ok {
			_, err := bot.EditMessageText(ctx.Context(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: message.Chat.ID},
				MessageID:   message.MessageID,
				Text:        selectText,
				ParseMode:   "HTML",
				ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
			})
			if err != nil {
				logger.Warningf("Error editing message for language selection: %v", err)
			}
		}
	}

	return nil
}
