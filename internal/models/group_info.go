package models

import (
	"fmt"
	"sync"
)

type GroupInfo struct {
	GroupID   int64
	GroupName string
	GroupLink string
	IsAdmin   bool
	AdminID   int64
}

func (g *GroupInfo) GetLinkedGroupName() string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", g.GroupLink, g.GroupName)
}

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
