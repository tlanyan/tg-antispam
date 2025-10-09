package service

import (
	"context"
	"fmt"
	"time"

	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"

	"github.com/mymmrac/telego"
)

// GetGroupInfo gets group info from cache or database
func GetGroupInfo(bot *telego.Bot, groupID int64, create bool) *models.GroupInfo {
	// First check the in-memory cache
	groupInfo := groupInfoManager.GetGroupInfo(groupID)
	if groupInfo != nil {
		return groupInfo
	}

	if groupRepository != nil {
		dbGroupInfo, err := groupRepository.GetGroupInfo(groupID)
		if err != nil {
			logger.Warningf("Error fetching group info from database: %v", err)
		} else if dbGroupInfo != nil {
			logger.Infof("Found group info in database for groupID: %d", groupID)
			groupInfo = dbGroupInfo
			// Add to cache
			groupInfoManager.AddGroupInfo(groupInfo)
			return groupInfo
		}
	}

	if !create {
		return nil
	}

	logger.Infof("Creating new group info for groupID: %d", groupID)
	groupInfo = &models.GroupInfo{
		GroupID:            groupID,
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

	// get group name and link from telegram
	if groupID < 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		chatInfo, err := bot.GetChat(ctx, &telego.GetChatParams{
			ChatID: telego.ChatID{ID: groupID},
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
			groupIDForLink := groupID
			if groupIDForLink < -1000000000000 {
				// Extract the actual ID from the negative number (skip the -100 prefix)
				groupIDForLink = -groupIDForLink - 1000000000000
			}
			groupInfo.GroupLink = fmt.Sprintf("https://t.me/c/%d", groupIDForLink)
		}

		groupInfo.AdminID, groupInfo.IsAdmin = GetBotPromoterID(bot, groupID)
		
		// Only update the database if the GroupID represents a group (GroupID < 0)
		if groupRepository != nil {
			if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
				logger.Warningf("Error saving group info to database for groupID %d: %v", groupID, err)
			}
		}
		
	} else {
	    // For user records (GroupID > 0), only add to cache, do not save to DB.
		logger.Debugf("Skipped saving user record (group_id > 0) to database during GetGroupInfo: %d", groupID)
	}

	logger.Infof("Group info created: %+v", groupInfo)

	groupInfoManager.AddGroupInfo(groupInfo)

	return groupInfo
}

// UpdateGroupInfo updates group information in cache and database
// For records representing users (GroupID > 0), it only updates the cache, not the database.
func UpdateGroupInfo(groupInfo *models.GroupInfo) {
	groupInfoManager.AddGroupInfo(groupInfo)

    if groupRepository != nil && groupInfo.GroupID < 0 {
        if err := groupRepository.CreateOrUpdateGroupInfo(groupInfo); err != nil {
            logger.Warningf("Error updating group info in database for groupID %d: %v", groupInfo.GroupID, err)
        }
    } else if groupRepository != nil && groupInfo.GroupID > 0 {
        logger.Debugf("Skipped updating database for user record (group_id > 0): %d", groupInfo.GroupID)
    }
}

// DeleteGroupInfo removes group information from both cache and database
func DeleteGroupInfo(groupID int64) error {
    // 1. Remove from memory cache
    groupInfoManager.RemoveGroupInfo(groupID)
    logger.Infof("Removed group info for groupID: %d from cache", groupID)

    // 2. Remove from database (if repository is available)
    if groupRepository != nil {
        if err := groupRepository.DeleteGroupInfo(groupID); err != nil {
            logger.Warningf("Error deleting group info from database for groupID %d: %v", groupID, err)
            // 如果数据库删除失败，可以选择是否回滚缓存删除（这里不回滚）
            return err
        }
        logger.Infof("Deleted group info for groupID: %d from database", groupID)
    } else {
        logger.Warning("Database repository is not available, only removed from cache.")
    }

    return nil
}

func GetBotPromoterID(bot *telego.Bot, chatID int64) (int64, bool) {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admins, err := bot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: chatID},
	})
	if err != nil {
		logger.Warningf("Error getting chat administrators for chat %d: %v", chatID, err)
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
func GetGroupName(bot *telego.Bot, chatID int64) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
