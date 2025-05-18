package service

import (
	"tg-antispam/internal/config"
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
	"tg-antispam/internal/storage"
)

var (
	groupInfoManager     = models.NewGroupInfoManager()
	groupRepository      *storage.GroupRepository
	banRepository        *storage.BanRepository
	pendingMsgRepository *storage.PendingMsgRepository
	globalConfig         *config.Config
)

// Initialize initializes the service with configuration
func Initialize(cfg *config.Config) {
	globalConfig = cfg
}

// InitRepositories initializes the repositories if database is enabled
func InitRepositories() {
	if storage.DB != nil {
		groupRepository = storage.NewGroupRepository(storage.DB)
		if err := groupRepository.MigrateTable(); err != nil {
			logger.Warningf("Error migrating GroupInfo table: %v", err)
		}
		// Load existing groups from the database
		if err := storage.InitializeGroups(groupInfoManager); err != nil {
			logger.Warningf("Error loading groups from database: %v", err)
		}
		// Initialize BanRecord table
		banRepository = storage.NewBanRepository(storage.DB)
		if err := banRepository.MigrateTable(); err != nil {
			logger.Warningf("Error migrating BanRecord table: %v", err)
		}
		// Initialize PendingMessageDeletion table
		pendingMsgRepository = storage.NewPendingMsgRepository(storage.DB)
		if err := pendingMsgRepository.MigrateTable(); err != nil {
			logger.Warningf("Error migrating PendingMessageDeletion table: %v", err)
		}
	}
}
