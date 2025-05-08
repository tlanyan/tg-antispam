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
	logger.Infof("GetGroupInfo called for chatID: %d", chatID)

	// First check the in-memory cache
	groupInfo := groupInfoManager.GetGroupInfo(chatID)
	if groupInfo != nil {
		logger.Infof("Found group info in cache for chatID: %d", chatID)
		return groupInfo
	}

	// If not found in cache but database is enabled, try to load it from database
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

	// If still not found, create a new one with default values
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

	// Try to get group information from Telegram
	chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
		ChatID: telego.ChatID{ID: chatID},
	})

	if err != nil {
		logger.Warningf("Error getting chat info from Telegram: %v", err)
		// Still return the default group info
		return groupInfo
	}

	// Update group name
	groupInfo.GroupName = chatInfo.Title

	// Set group link if available
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

	// Try to check admin status
	groupInfo.AdminID, groupInfo.IsAdmin = GetBotPromoterID(ctx, bot, chatID)
	logger.Infof("Group info created: %+v", groupInfo)

	// Save to cache
	groupInfoManager.AddGroupInfo(groupInfo)

	// Save to database if enabled
	if groupRepository != nil {
		if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
			logger.Warningf("Error saving group info to database: %v", err)
		}
	}

	return groupInfo
}

// UpdateGroupInfo updates group information in cache and database
func UpdateGroupInfo(groupInfo *models.GroupInfo) {
	// Update cache
	groupInfoManager.AddGroupInfo(groupInfo)

	// Update database if enabled
	if groupRepository != nil {
		if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
			logger.Warningf("Error updating group info in database: %v", err)
		}
	}
}

func GetBotPromoterID(ctx context.Context, bot *telego.Bot, chatID int64) (int64, bool) {
	// Need a bot instance to make API calls
	newBot, err := telego.NewBot(globalConfig.Bot.Token)
	if err != nil {
		logger.Warningf("Error creating temporary bot for admin check: %v", err)
		return 0, false
	}
	defer newBot.Close(ctx)

	// Get all administrators in the chat
	admins, err := newBot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		logger.Warningf("Error getting chat administrators: %v", err)
		return 0, false
	}

	botID := bot.ID()
	// First check: First determine if bot is an admin
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

		// Check if this admin can promote members
		if admin.MemberStatus() == telego.MemberStatusAdministrator {
			if adminMember, ok := admin.(*telego.ChatMemberAdministrator); ok {
				if adminMember.CanPromoteMembers {
					candidateAdmins = append(candidateAdmins, *adminMember)
				}
			}
		}
	}

	// If we found admins who can promote, return the first one
	// This is our best guess at who promoted the bot
	if len(candidateAdmins) > 0 {
		return candidateAdmins[0].User.ID, true
	}

	logger.Warningf("Could not find bot promoter in chat %d", chatID)
	return 0, false
}

// GetLinkedGroupName gets a linked HTML representation of the group name with caching
func GetGroupName(ctx context.Context, bot *telego.Bot, chatID int64) (string, string) {
	// Cache miss, fetch from API
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
		// For private groups, convert the chat ID to work with t.me links
		// Telegram requires removing the -100 prefix from supergroup IDs for links
		groupIDForLink := chatID
		if groupIDForLink < -1000000000000 {
			// Extract the actual ID from the negative number (skip the -100 prefix)
			groupIDForLink = -groupIDForLink - 1000000000000
		}
		groupLink = fmt.Sprintf("https://t.me/c/%d", groupIDForLink)
	}

	return chatInfo.Title, groupLink
}
