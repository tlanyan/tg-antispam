package service

import (
	"context"
	"fmt"

	"tg-antispam/internal/config"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
	"tg-antispam/internal/storage"

	"github.com/mymmrac/telego"
)

var (
	groupInfoManager = models.NewGroupInfoManager()
	groupRepository  *storage.GroupRepository
	globalConfig     *config.Config
)

// Initialize initializes the service with configuration
func Initialize(cfg *config.Config) {
	globalConfig = cfg
}

// InitGroupRepository initializes the group repository if database is enabled
func InitGroupRepository() {
	if storage.DB != nil {
		groupRepository = storage.NewGroupRepository(storage.DB)
		if err := groupRepository.MigrateTable(); err != nil {
			logger.Warningf("Error migrating GroupInfo table: %v", err)
		}
		// Load existing groups from the database
		if err := storage.InitializeGroups(groupInfoManager); err != nil {
			logger.Warningf("Error loading groups from database: %v", err)
		}
	}
}

func GetGroupInfo(ctx context.Context, bot *telego.Bot, chatID int64) *models.GroupInfo {
	// First check the in-memory cache
	groupInfo := groupInfoManager.GetGroupInfo(chatID)
	if groupInfo != nil {
		return groupInfo
	}

	if groupRepository != nil {
		dbGroupInfo, err := groupRepository.GetGroupInfo(chatID)
		if err != nil {
			logger.Warningf("Error fetching group info from database: %v", err)
		} else if dbGroupInfo != nil {
			logger.Infof("Found group info in database for chatID: %d", chatID)
			groupInfo = dbGroupInfo
			// Add to cache
			groupInfoManager.AddGroupInfo(groupInfo)
			return groupInfo
		}
	}

	logger.Infof("Creating new group info for chatID: %d", chatID)
	groupInfo = &models.GroupInfo{
		GroupID:            chatID,
		IsAdmin:            false,
		AdminID:            -1,
		EnableNotification: true,
		BanPremium:         globalConfig.Antispam.BanPremium,
		BanEmojiName:       globalConfig.Antispam.BanEmojiName,
		BanRandomUsername:  globalConfig.Antispam.BanRandomUsername,
		BanBioLink:         globalConfig.Antispam.BanBioLink,
		EnableCAS:          globalConfig.Antispam.UseCAS,
		Language:           "zh_CN",
	}

	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})

	if err != nil {
		logger.Warningf("Error getting chat info from Telegram: %v", err)
		return groupInfo
	}

	groupInfo.GroupName = chatInfo.Title

	if chatInfo.Username != "" {
		groupInfo.GroupLink = fmt.Sprintf("https://t.me/%s", chatInfo.Username)
	} else {
		// For private groups, convert the chat ID to work with t.me links
		// Telegram requires removing the -100 prefix from supergroup IDs for links
		groupIDForLink := chatID
		if groupIDForLink < -1000000000000 {
			// Extract the actual ID from the negative number (skip the -100 prefix)
			groupIDForLink = -groupIDForLink - 1000000000000
		}
		groupInfo.GroupLink = fmt.Sprintf("https://t.me/c/%d", groupIDForLink)
	}

	groupInfo.AdminID, groupInfo.IsAdmin = GetBotPromoterID(ctx, bot, chatID)
	logger.Infof("Group info created: %+v", groupInfo)

	groupInfoManager.AddGroupInfo(groupInfo)

	if groupRepository != nil {
		if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
			logger.Warningf("Error saving group info to database: %v", err)
		}
	}

	return groupInfo
}

// UpdateGroupInfo updates group information in cache and database
func UpdateGroupInfo(groupInfo *models.GroupInfo) {
	groupInfoManager.AddGroupInfo(groupInfo)

	if groupRepository != nil {
		if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
			logger.Warningf("Error updating group info in database: %v", err)
		}
	}
}

func GetBotPromoterID(ctx context.Context, bot *telego.Bot, chatID int64) (int64, bool) {
	newBot, err := telego.NewBot(globalConfig.Bot.Token)
	if err != nil {
		logger.Warningf("Error creating temporary bot for admin check: %v", err)
		return 0, false
	}
	defer newBot.Close(ctx)

	admins, err := newBot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		logger.Warningf("Error getting chat administrators: %v", err)
		return 0, false
	}

	botID := bot.ID()
	botIsAdmin := false
	for _, admin := range admins {
		if admin.MemberUser().ID == botID {
			botIsAdmin = true
			break
		}
	}

	if !botIsAdmin {
		return 0, false
	}

	// Find the chat creator if available (they definitely have promotion rights)
	for _, admin := range admins {
		if admin.MemberStatus() == telego.MemberStatusCreator {
			if creator, ok := admin.(*telego.ChatMemberOwner); ok {
				return creator.User.ID, true
			}
		}
	}

	// Second: Find admins who can promote members (one of them promoted the bot)
	var candidateAdmins []telego.ChatMemberAdministrator
	for _, admin := range admins {
		// Skip the bot itself
		if admin.MemberUser().ID == botID {
			continue
		}

		if admin.MemberStatus() == telego.MemberStatusAdministrator {
			if adminMember, ok := admin.(*telego.ChatMemberAdministrator); ok {
				if adminMember.CanPromoteMembers {
					candidateAdmins = append(candidateAdmins, *adminMember)
				}
			}
		}
	}

	if len(candidateAdmins) > 0 {
		return candidateAdmins[0].User.ID, true
	}

	logger.Warningf("Could not find bot promoter in chat %d", chatID)
	return 0, false
}

// GetLinkedGroupName gets a linked HTML representation of the group name with caching
func GetGroupName(ctx context.Context, bot *telego.Bot, chatID int64) (string, string) {
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		logger.Warningf("Error getting chat info: %v", err)
		return "", ""
	}

	var groupLink string
	if chatInfo.Username != "" {
		groupLink = fmt.Sprintf("https://t.me/%s", chatInfo.Username)
	} else {
		groupIDForLink := chatID
		if groupIDForLink < -1000000000000 {
			// Extract the actual ID from the negative number (skip the -100 prefix)
			groupIDForLink = -groupIDForLink - 1000000000000
		}
		groupLink = fmt.Sprintf("https://t.me/c/%d", groupIDForLink)
	}

	return chatInfo.Title, groupLink
}
