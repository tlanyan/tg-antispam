package handler

import (
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"tg-antispam/internal/config"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/service"
	"tg-antispam/internal/storage"
)

// unbannParamRegex matches the format "unban_groupID_userID"
var unbannParamRegex = regexp.MustCompile(`^unban_(-?\d+)_(\d+)$`)

// pendingUsers user and group id to pending security check
var pendingUsers = make(map[int64]int64)

// handleIncomingMessage processes new messages in chats
func handleIncomingMessage(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	// Skip if no sender information or sender is a bot
	if message.From == nil || message.From.IsBot {
		return nil
	}

	// group_id 限制：只处理指定群组
	cfg := config.Get()
	if message.Chat.Type != "private" && cfg.Bot.GroupID != -1 && message.Chat.ID != cfg.Bot.GroupID {
		return nil
	}

	// Check for math verification answers first
	if message.Chat.Type == "private" {
		return handlePrivateMessage(ctx, bot, message)
	}

	if message.Chat.ID > 0 {
		// 获取并打印调用堆栈
		stackTrace := string(debug.Stack())
		logger.Infof("private chat message call stack: %s", stackTrace)

		// Prompt user to send /help command
		_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			Text:      "请发送 /help 获取使用帮助。\nPlease send /help to get help.",
			ParseMode: "HTML",
		})
		return err
	}

	return handleGroupMessage(ctx, bot, message)
}

func handlePrivateMessage(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	// Handle /start command with parameters for self-unban
	if message.Text != "" && strings.HasPrefix(message.Text, "/start ") {
		startParam := strings.TrimPrefix(message.Text, "/start ")
		// Check if this is an unban request
		if matches := unbannParamRegex.FindStringSubmatch(startParam); matches != nil {
			groupID, _ := strconv.ParseInt(matches[1], 10, 64)
			userID, _ := strconv.ParseInt(matches[2], 10, 64)

			// Verify that the user requesting unban is the same user who was banned
			if message.From.ID != userID {
				_, err := bot.SendMessage(ctx.Context(), &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: message.Chat.ID},
					Text:      "不能为其他用户解除限制。\nYou cannot unban for other users.",
					ParseMode: "HTML",
				})
				return err
			}

			return SendMathVerificationMessage(ctx, bot, userID, groupID, nil)
		}

		// If not an unban request, continue with normal processing
	}

	// Check for pending math verification answer
	if err := HandleMathVerification(ctx, bot, message); err != nil {
		logger.Warningf("Error handling math verification: %v", err)
	}

	// @TODO: handle other commands/messages
	return nil
}

func handleGroupMessage(ctx *th.Context, bot *telego.Bot, message telego.Message) error {
	cfg := config.Get()
	groupInfo := service.GetGroupInfo(ctx, bot, message.Chat.ID, true)
	if !groupInfo.IsAdmin {
		return nil
	}
	logger.Infof("Processing message: %+v, from: %+v", message, *message.From)
	// Use database configuration if available
	shouldRestrict := false
	reason := ""

	text := ""
	// @TODO: more rules
	if strings.Contains(message.Text, "https://t.me/") || (strings.Contains(message.Text, "@") && !strings.HasPrefix(message.Text, "/")) {
		text = message.Text
	} else if message.ForwardOrigin != nil {
		text = message.Caption
	} else if message.Quote != nil {
		text = message.Quote.Text
	}

	if text != "" {
		logger.Infof("suspicious message: %s, request cas or ai check", text)
		shouldRestrict, reason = CasRequest(message.From.ID)

		if !shouldRestrict && groupInfo.EnableAicheck {
			if cfg.AiApi.GeminiApiKey != "" {
				isSpam, err := ClassifyWithGemini(cfg.AiApi.GeminiApiKey, cfg.AiApi.GeminiModel, text)
				if err != nil {
					logger.Warningf("Error classifying message with Gemini: %v", err)
				} else if isSpam {
					shouldRestrict = true
					reason = "reason_ai_spam"
				}
				logger.Infof("AI check message: %s, result: %t", text, isSpam)
			}
		}
	}

	if shouldRestrict {
		logger.Infof("suspicious message text: %s, delete and restrict user: %d", text, message.From.ID)
		bot.DeleteMessage(ctx.Context(), &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: message.Chat.ID},
			MessageID: message.MessageID,
		})
		RestrictUser(ctx.Context(), bot, message.Chat.ID, message.From.ID)
		// Record the ban event in database
		service.CreateBanRecord(message.Chat.ID, message.From.ID, reason)
		// Send warning only if notifications are enabled
		if groupInfo.EnableNotification {
			SendWarning(ctx.Context(), bot, groupInfo.GroupID, *message.From, reason)
		}
	}

	return nil
}

// handleChatMemberUpdate processes updates to chat members
func handleChatMemberUpdate(ctx *th.Context, bot *telego.Bot, update telego.Update) error {
	botID := bot.ID()
	// Process ChatMember updates (when users join chat or change status)
	if update.ChatMember == nil {
		return nil
	}

	chatId := update.ChatMember.Chat.ID
	// group_id 限制：只处理指定群组
	cfg := config.Get()
	if cfg.Bot.GroupID != -1 && chatId != cfg.Bot.GroupID {
		return nil
	}

	fromUser := update.ChatMember.From
	// Skip updates related to the bot itself
	if fromUser.ID == botID {
		return nil
	}

	newChatMember := update.ChatMember.NewChatMember
	logger.Infof("new Chat member: %+v", newChatMember)
	groupInfo := service.GetGroupInfo(ctx, bot, chatId, true)
	// Track admin who promoted the bot
	if newChatMember.MemberUser().ID == botID {
		// Check if the bot's status was changed to admin
		if newChatMember.MemberStatus() == telego.MemberStatusAdministrator {
			logger.Infof("Bot was promoted to admin in chat %d by user %d", chatId, fromUser.ID)
			groupInfo.IsAdmin = true
			groupInfo.AdminID = fromUser.ID
			// Update the group info
			service.UpdateGroupInfo(groupInfo)
		} else {
			groupInfo.IsAdmin = false
			// Update the group info
			service.UpdateGroupInfo(groupInfo)
		}
		return nil
	}

	if !groupInfo.IsAdmin {
		logger.Infof("Bot not and admin for chat ID: %d", chatId)
		return nil
	}

	return checkRestrictedUser(ctx, bot, chatId, newChatMember, fromUser)
}

func checkRestrictedUser(ctx *th.Context, bot *telego.Bot, chatId int64, newChatMember telego.ChatMember, fromUser telego.User) error {
	if newChatMember.MemberStatus() == telego.MemberStatusLeft || newChatMember.MemberStatus() == telego.MemberStatusBanned {
		return nil
	}

	user := newChatMember.MemberUser()
	if newChatMember.MemberIsMember() {
		// Skip bots
		if user.IsBot {
			logger.Infof("Skipping bot update: %s", user.FirstName)
			return nil
		}

		// 首次入群，等待入群机器人处理，如果没有入群机器人则封禁
		if !fromUser.IsBot && newChatMember.MemberStatus() == telego.MemberStatusMember {
			if _, ok := pendingUsers[user.ID]; !ok {
				pendingUsers[user.ID] = chatId
				go func() {
					time.Sleep(1 * time.Second)
					if _, ok := pendingUsers[user.ID]; ok {
						reason := "reason_join_group"
						restrictUser(ctx, bot, chatId, user, reason)
						delete(pendingUsers, user.ID)
					}
				}()
			}

			return nil
		}

		// 如果其它机器人封禁了用户，则不限制用户
		if newChatMember.MemberStatus() == telego.MemberStatusRestricted && user.ID != bot.ID() {
			if restrictedMember, ok := newChatMember.(*telego.ChatMemberRestricted); ok {
				// 现在可以访问 CanSendMessages 属性
				canSendMsg := restrictedMember.CanSendMessages
				if !canSendMsg {
					delete(pendingUsers, user.ID)
					return nil
				}
			}
		}

		// Check if user should be restricted
		groupInfo := service.GetGroupInfo(ctx, bot, chatId, false)
		shouldRestrict, reason := ShouldRestrictUser(ctx, bot, groupInfo, user)
		if !shouldRestrict && groupInfo.EnableCAS {
			shouldRestrict, reason = CasRequest(user.ID)
		}

		if !shouldRestrict {
			reason = "reason_join_group"
		}
		restrictUser(ctx, bot, chatId, user, reason)
	}
	return nil
}

func restrictUser(ctx *th.Context, bot *telego.Bot, chatId int64, user telego.User, reason string) {
	logger.Infof("Restricting user: %s, reason: %s", user.FirstName, reason)
	RestrictUser(ctx.Context(), bot, chatId, user.ID)
	// Send warning only if notifications are enabled
	groupInfo := service.GetGroupInfo(ctx, bot, chatId, false)
	if groupInfo.EnableNotification {
		SendWarning(ctx.Context(), bot, groupInfo.GroupID, user, reason)
	}
}

// handleMyChatMemberUpdate processes updates to the bot's own chat member status
func handleMyChatMemberUpdate(ctx *th.Context, bot *telego.Bot, update telego.Update) error {
	// Process MyChatMember updates (when users block/unblock the bot in private chats)
	if update.MyChatMember == nil {
		return nil
	}

	logger.Infof("MyChatMember update: %+v", update.MyChatMember)

	// Only process private chat updates (when a user blocks/unblocks the bot)
	if update.MyChatMember.Chat.Type == "private" {
		userID := update.MyChatMember.From.ID
		newStatus := update.MyChatMember.NewChatMember.MemberStatus()

		// When a user blocks/stops the bot (MemberStatusLeft for left, "kicked" for blocked/stopped)
		if newStatus == telego.MemberStatusLeft || newStatus == "kicked" {
			logger.Infof("User %d has blocked/stopped the bot", userID)

			// Disable notifications for all groups associated with this admin
			if storage.DB != nil {
				// Create a temporary repository
				groupRepository := storage.NewGroupRepository(storage.DB)

				groups, err := groupRepository.GetGroupsByAdminID(userID)
				if err != nil {
					logger.Warningf("Error getting groups for admin %d: %v", userID, err)
					return nil
				}

				logger.Infof("Found %d groups for admin %d", len(groups), userID)

				// Update each group's notification setting in both cache and DB
				for _, group := range groups {
					group.EnableNotification = false
					service.UpdateGroupInfo(group)

					logger.Infof("Disabled notifications for group %d", group.GroupID)
				}

				// Update all groups in DB at once
				err = groupRepository.DisableNotificationsForAdmin(userID)
				if err != nil {
					logger.Warningf("Error disabling notifications for admin %d: %v", userID, err)
				}
			}
		}
	}
	return nil
}
