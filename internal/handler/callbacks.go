package handler

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"

	"tg-antispam/internal/crash"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
	"tg-antispam/internal/service"
)

// verificationAnswers stores pending math verification answers with group IDs
var verificationAnswers = make(map[int64]struct {
	Answer  int
	GroupID int64
})

// Add a map to track failed verification attempts
var verificationAttempts = make(map[int64]int)

// HandleCallbackQuery processes callback queries from inline keyboards
func HandleCallbackQuery(bot *telego.Bot, query telego.CallbackQuery) error {
	// Skip if no data
	if query.Data == "" {
		return nil
	}

	logger.Infof("Received callback query: %s", query.Data)

	// Handle different callback types based on prefix
	if strings.HasPrefix(query.Data, "unban:") {
		return handleUnbanCallback(bot, query)
	} else if strings.HasPrefix(query.Data, "self_unban:") {
		return handleSelfUnbanCallback(bot, query)
	} else if strings.HasPrefix(query.Data, "lang:") {
		return handleLanguageCallback(bot, query)
	} else if strings.HasPrefix(query.Data, "group:") {
		return handleGroupSelectionCallback(bot, query)
	} else if strings.HasPrefix(query.Data, "action:") {
		return handleActionSelectionCallback(bot, query)
	} else if strings.HasPrefix(query.Data, "ban:") {
		return handleBanCallback(bot, query)
	}

	return nil
}

func checkAdminQuery(bot *telego.Bot, query telego.CallbackQuery, groupID int64) (bool, error) {
	language := GetBotQueryLang(bot, &query)
	// Check if the callback sender is an admin in the chat
	if !isUserAdmin(bot, groupID, query.From.ID) {
		// Inform user they don't have permission
		err := bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            models.GetTranslation(language, "user_not_admin"),
			ShowAlert:       true,
		})
		return false, err
	}
	return true, nil
}

// handleUnbanCallback processes a request to unban a user
func handleUnbanCallback(bot *telego.Bot, query telego.CallbackQuery) error {
	groupID, userID, err := getGroupAndUserID(query.Data)
	if err != nil {
		logger.Warningf("Invalid callback data in unban callback: %s", query.Data)
		return nil
	}

	logger.Infof("Unban callback received: %+v, groupID=%d, userID=%d", query, groupID, userID)
	// Get group info for language
	language := GetBotQueryLang(bot, &query)
	isAdmin, err := checkAdminQuery(bot, query, groupID)
	if !isAdmin {
		return err
	}

	// Unrestrict the user
	UnrestrictUser(bot, groupID, userID)
	// Update ban_records to mark as unbanned
	service.MarkBanRecordUnbanned(groupID, userID, "admin")

	// Notify the admin that the action was successful
	err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            models.GetTranslation(language, "warning_user_unbanned"),
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	linkedUserName, err := getLinkedUserName(bot, userID)
	if err != nil {
		logger.Warningf("Error getting linked user name: %v", err)
		return err
	}

	// Update the message to reflect that the user was unbanned
	if query.Message != nil {
		if accessibleMsg, ok := query.Message.(*telego.Message); ok {
			// Create ban button
			banButton := telego.InlineKeyboardButton{
				Text:         models.GetTranslation(language, "ban_user"),
				CallbackData: fmt.Sprintf("ban:%d:%d", groupID, userID),
			}
			keyboard := [][]telego.InlineKeyboardButton{
				{banButton},
			}

			bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: accessibleMsg.Chat.ID},
				MessageID:   accessibleMsg.MessageID,
				Text:        fmt.Sprintf(models.GetTranslation(language, "warning_unbanned_message"), linkedUserName),
				ParseMode:   "HTML",
				ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
			})
		}
	}

	return err
}

// handleBanCallback processes a request to ban a user
func handleBanCallback(bot *telego.Bot, query telego.CallbackQuery) error {
	groupID, userID, err := getGroupAndUserID(query.Data)
	if err != nil {
		logger.Warningf("Invalid callback data in ban callback: %s", query.Data)
		return nil
	}

	logger.Infof("Ban callback received: %+v, groupID=%d, userID=%d", query, groupID, userID)
	// Check if the callback sender is an admin in the chat
	isAdmin, err := checkAdminQuery(bot, query, groupID)
	if !isAdmin {
		return err
	}

	// Get group info for language
	language := GetBotQueryLang(bot, &query)

	// Restrict the user (ban from sending messages and media)
	err = bot.RestrictChatMember(context.Background(), &telego.RestrictChatMemberParams{
		ChatID:      telego.ChatID{ID: groupID},
		UserID:      userID,
		Permissions: telego.ChatPermissions{},
	})
	if err != nil {
		logger.Warningf("Error restricting user: %v", err)
		return err
	}

	// Notify the admin that the action was successful
	err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            models.GetTranslation(language, "user_banned"),
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	linkedUserName, err := getLinkedUserName(bot, userID)
	if err != nil {
		logger.Warningf("Error getting linked user name: %v", err)
		return err
	}

	// Update the message to reflect that the user was banned
	if query.Message != nil {
		if accessibleMsg, ok := query.Message.(*telego.Message); ok {
			// Create unban button
			unbanButton := telego.InlineKeyboardButton{
				Text:         models.GetTranslation(language, "unban_user"),
				CallbackData: fmt.Sprintf("unban:%d:%d", groupID, userID),
			}
			keyboard := [][]telego.InlineKeyboardButton{
				{unbanButton},
			}

			bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: accessibleMsg.Chat.ID},
				MessageID:   accessibleMsg.MessageID,
				Text:        fmt.Sprintf(models.GetTranslation(language, "warning_banned_message"), linkedUserName),
				ParseMode:   "HTML",
				ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
			})
		}
	}

	return err
}

// handleLanguageCallback processes language selection callbacks
func handleLanguageCallback(bot *telego.Bot, query telego.CallbackQuery) error {
	// Format: lang:language:chatID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		logger.Warningf("Invalid callback data in language callback: %s", parts)
		return nil
	}

	language := parts[1]
	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		logger.Warningf("Invalid chat ID in language callback: %v", err)
		return nil
	}

	// 直接传递变量给setLanguage函数处理
	return setLanguage(bot, query, groupID, language)
}

// setLanguage updates the language setting for a group
func setLanguage(bot *telego.Bot, query telego.CallbackQuery, groupID int64, language string) error {
	logger.Infof("Setting language for group: %d, language: %s", groupID, language)
	// Get the group info
	groupInfo := service.GetGroupInfo(bot, groupID, false)
	if groupInfo == nil {
		logger.Warningf("Group info not found: %d", groupID)
		return nil
	}
	// Check if the user is an admin
	if groupInfo.AdminID != query.From.ID {
		isAdmin, err := checkAdminQuery(bot, query, groupID)
		if !isAdmin {
			return err
		}
	}

	// Update the language
	groupInfo.Language = language
	service.UpdateGroupInfo(groupInfo)

	// Notify about the change
	err := bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            fmt.Sprintf(models.GetTranslation(language, "language_updated"), getLanguageName(language)),
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// show group settings with new language
	if groupID < 0 {
		settingsText, keyboard := buildGroupSettingsMessageParts(groupInfo, language, groupID)

		// Update the message
		if query.Message != nil {
			if accessibleMsg, ok := query.Message.(*telego.Message); ok {
				_, editErr := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
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
	} else {
		if query.Message != nil {
			if accessibleMsg, ok := query.Message.(*telego.Message); ok {
				_, editErr := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:    telego.ChatID{ID: accessibleMsg.Chat.ID},
					MessageID: accessibleMsg.MessageID,
					Text:      fmt.Sprintf(models.GetTranslation(language, "language_updated"), getLanguageName(language)),
					ParseMode: "HTML",
				})
				if editErr != nil {
					logger.Warningf("Error editing select language message: %v", editErr)
				}
			}
		}
	}

	return err
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
func handleGroupSelectionCallback(bot *telego.Bot, query telego.CallbackQuery) error {
	// 提取群组ID
	parts := strings.Split(query.Data, ":")
	if len(parts) != 3 {
		return nil
	}

	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		logger.Warningf("Invalid group ID in callback: %v", err)
		return nil
	}

	action := parts[1]
	logger.Infof("Group selection callback received: group=%d, action=%s", groupID, action)

	// 通知用户已收到请求
	err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "正在处理您的请求...",
	})
	if err != nil {
		logger.Warningf("Error answering callback query: %v", err)
	}

	// 根据操作类型处理
	if query.Message != nil {
		if message, ok := query.Message.(*telego.Message); ok {
			switch action {
			case "settings":
				// 显示群组设置
				return showGroupSettings(bot, *message, groupID)
			default:
				// 对于其他操作类型，创建action回调
				callbackData := fmt.Sprintf("action:%s:%d", action, groupID)
				actionQuery := telego.CallbackQuery{
					ID:      query.ID,
					From:    query.From,
					Data:    callbackData,
					Message: query.Message,
				}
				return handleActionSelectionCallback(bot, actionQuery)
			}
		}
	}

	return nil
}

// 处理设置项操作回调
func handleActionSelectionCallback(bot *telego.Bot, query telego.CallbackQuery) error {
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

	logger.Infof("Action selection callback received: %+v, groupID=%d, action: %s", query, groupID, action)

	if query.ID != "" {
		// 通知用户已收到请求
		err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "正在处理您的请求...",
		})
		if err != nil {
			logger.Warningf("Error answering callback query: %v", err)
		}
	}

	// 获取群组信息
	groupInfo := service.GetGroupInfo(bot, groupID, false)
	if groupInfo == nil {
		logger.Warningf("Group info not found: %d", groupID)
		return nil
	}

	// 检查用户是否是管理员
	if groupInfo.AdminID != query.From.ID {
		isAdmin, err := checkAdminQuery(bot, query, groupID)
		if !isAdmin {
			return err
		}
	}

	// 处理不同的操作
	language := GetBotQueryLang(bot, &query)

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

	case "language_group":
		// 显示语言选择界面
		return showLanguageSelection(bot, query, groupID, language)
	}

	// 保存群组设置
	service.UpdateGroupInfo(groupInfo)

	// 通知用户设置已更新
	if updateMessage != "" {
		err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
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
			return showGroupSettings(bot, *message, groupID)
		}
	}

	return nil
}

// showLanguageSelection displays language selection options
func showLanguageSelection(bot *telego.Bot, query telego.CallbackQuery, groupID int64, language string) error {
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

	if query.ID != "" {
		if query.Message != nil {
			if message, ok := query.Message.(*telego.Message); ok {
				_, err := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
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
	} else {
		var message telego.Message
		switch msg := query.Message.(type) {
		case *telego.Message:
			message = *msg
		default:
			logger.Warningf("Unexpected message type in language selection: %T", msg)
			return nil
		}

		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: message.Chat.ID},
			Text:        selectText,
			ParseMode:   "HTML",
			ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard},
		})
		if err != nil {
			logger.Warningf("Error sending language selection message: %v", err)
		}
	}
	return nil
}

func SendMathVerificationMessage(bot *telego.Bot, userID int64, groupID int64, query *telego.CallbackQuery) error {
	// Generate a random math problem
	num1 := rand.Intn(100)
	num2 := rand.Intn(100)
	operators := []string{"+", "-", "*"}
	operator := operators[rand.Intn(len(operators))]

	// Calculate the correct answer
	var correctAnswer int
	switch operator {
	case "+":
		correctAnswer = num1 + num2
	case "-":
		correctAnswer = num1 - num2
	case "*":
		correctAnswer = num1 * num2
	}

	// Store the answer in a temporary map (in a real implementation, you'd want to use a proper storage solution)
	verificationAnswers[userID] = struct {
		Answer  int
		GroupID int64
	}{correctAnswer, groupID}
	verificationAttempts[userID] = 0

	// Get group info for language
	language := GetBotQueryLang(bot, query)

	logger.Infof("Send math verification message to user: %d, groupID: %d", userID, groupID)
	// Send the math problem to the user
	_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: userID},
		Text:      fmt.Sprintf(models.GetTranslation(language, "math_verification"), num1, operator, num2),
		ParseMode: "HTML",
	})

	if err != nil {
		logger.Warningf("Error sending math verification message: %v", err)
		return err
	}

	// Answer the callback query
	if query != nil {
		err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		if err != nil {
			logger.Warningf("Error answering callback query: %v", err)
		}
	}

	return nil
}

// handleSelfUnbanCallback processes a request for a user to unban themselves
func handleSelfUnbanCallback(bot *telego.Bot, query telego.CallbackQuery) error {
	groupID, userID, err := getGroupAndUserID(query.Data)
	if err != nil {
		logger.Warningf("Invalid callback data in self-unban callback: %s", query.Data)
		return nil
	}

	// Verify that the user clicking the button is the restricted user
	if query.From.ID != userID {
		err = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "只有被限制的用户才能使用此功能。",
			ShowAlert:       true,
		})
		return err
	}

	logger.Infof("handleSelfUnbanCallback, userID: %d, groupID: %d", userID, groupID)

	return SendMathVerificationMessage(bot, userID, groupID, &query)
}

// HandleMathVerification processes the user's answer to the math verification
func HandleMathVerification(bot *telego.Bot, message telego.Message) error {
	userID := message.From.ID
	// Check if the user has a pending verification
	expectedAnswer, exists := verificationAnswers[userID]
	if !exists {
		return nil // No pending verification
	}
	groupID := expectedAnswer.GroupID
	groupInfo := service.GetGroupInfo(bot, groupID, false)
	if groupInfo == nil {
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "无法确定需要解封的群组。请联系管理员解决。\nUnable to determine which group to unban. Please contact the administrator to resolve.",
			ParseMode: "HTML",
		})
		logger.Warningf("Group info not found: %d", groupID)
		return err
	}

	// Parse the user's answer
	userAnswer, err := strconv.Atoi(strings.TrimSpace(message.Text))
	if err != nil {
		// Not a valid number, ignore
		return nil
	}

	// Get group info for language
	language := GetBotLang(bot, message)

	// Check if the answer is correct
	if userAnswer == expectedAnswer.Answer {
		delete(verificationAnswers, userID)
		delete(verificationAttempts, userID)

		UnrestrictUser(bot, groupID, userID)
		service.MarkBanRecordUnbanned(groupID, userID, "self")

		crash.SafeGoroutine(fmt.Sprintf("cleanup-pending-messages-%d-%d", userID, groupID), func() {
			msgs, err := service.GetPendingMsgsByUserID(userID, groupID)
			if err != nil {
				logger.Warningf("Error getting pending messages: %v", err)
			} else {
				for _, msg := range msgs {
					bot.DeleteMessage(context.Background(), &telego.DeleteMessageParams{
						ChatID:    telego.ChatID{ID: msg.ChatID},
						MessageID: int(msg.MessageID),
					})
					service.RemovePendingMsg(msg.ChatID, msg.MessageID)
				}
			}
		})

		// Send success message
		_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      models.GetTranslation(language, "math_verification_success"),
			ParseMode: "HTML",
		})
	} else {
		// Handle failed attempt count and potentially resend verification
		count := verificationAttempts[userID] + 1
		verificationAttempts[userID] = count
		if count >= 3 {
			err = SendMathVerificationMessage(bot, userID, expectedAnswer.GroupID, nil)
		} else {
			// Send failure message
			_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: message.Chat.ID},
				Text:      models.GetTranslation(language, "math_verification_failed"),
				ParseMode: "HTML",
			})
		}
	}

	if err != nil {
		logger.Warningf("Error sending verification result message: %v", err)
	}

	return err
}
