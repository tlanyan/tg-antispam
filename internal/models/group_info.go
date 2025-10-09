package models

import (
	"fmt"
	"sync"
	"time"
	"tg-antispam/internal/logger"
)

// GroupInfo represents group information and settings
type GroupInfo struct {
	ID                 uint  `gorm:"primaryKey;autoIncrement"`
	GroupID            int64 `gorm:"uniqueIndex;not null"`
	GroupName          string
	GroupLink          string
	AdminID            int64
	IsAdmin            bool
	WaitSec            int    `gorm:"default:3"`
	EnableNotification bool   `gorm:"default:true"`
	BanPremium         bool   `gorm:"default:true"`
	BanRandomUsername  bool   `gorm:"default:true"`
	BanEmojiName       bool   `gorm:"default:true"`
	BanBioLink         bool   `gorm:"default:true"`
	EnableCAS          bool   `gorm:"default:true"`
	EnableAicheck      bool   `gorm:"default:false"`
	Language           string `gorm:"default:zh_CN"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (g *GroupInfo) GetLinkedGroupName() string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", g.GroupLink, g.GroupName)
}

// GroupInfoManager manages cached group info
type GroupInfoManager struct {
	GroupInfoMap   map[int64]*GroupInfo
	GroupInfoMapMu sync.RWMutex
}

func NewGroupInfoManager() *GroupInfoManager {
	return &GroupInfoManager{
		GroupInfoMap:   make(map[int64]*GroupInfo),
		GroupInfoMapMu: sync.RWMutex{},
	}
}

func (g *GroupInfoManager) GetGroupInfo(chatID int64) *GroupInfo {
	g.GroupInfoMapMu.RLock()
	defer g.GroupInfoMapMu.RUnlock()
	return g.GroupInfoMap[chatID]
}

func (g *GroupInfoManager) AddGroupInfo(groupInfo *GroupInfo) {
	g.GroupInfoMapMu.Lock()
	defer g.GroupInfoMapMu.Unlock()
	g.GroupInfoMap[groupInfo.GroupID] = groupInfo
}

func (g *GroupInfoManager) RemoveGroupInfo(groupID int64) {
	g.GroupInfoMapMu.Lock()
	defer g.GroupInfoMapMu.Unlock()
	delete(g.GroupInfoMap, groupID)
}

// ResetUserCache clears only the in-memory cache entries where GroupID > 0 (representing users).
func (g *GroupInfoManager) ResetUserCache() {
    g.GroupInfoMapMu.Lock()
    defer g.GroupInfoMapMu.Unlock()
    
    removedCount := 0
    // Iterate through the map and delete entries with GroupID > 0
    for groupID := range g.GroupInfoMap {
        if groupID > 0 {
            delete(g.GroupInfoMap, groupID)
            removedCount++
        }
    }
    logger.Infof("User entries (group_id > 0) removed from GroupInfo cache: %d", removedCount)
}
