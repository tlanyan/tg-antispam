package service

import (
	"time"
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

// startCacheCleanup initializes and runs the periodic user cache reset goroutine.
func startCacheCleanup(manager *models.GroupInfoManager) {
    if manager == nil {
        logger.Warningf("GroupInfoManager is nil, cannot start user cache cleanup")
        return
    }
    // Define the cleanup interval (e.g., every 24 hours)
    ticker := time.NewTicker(24 * time.Hour)

    // Start the goroutine
    go func() {
        logger.Infof("Starting GroupInfo user cache cleanup goroutine with interval: %v", 24*time.Hour)
        for range ticker.C { // Receive ticks from the ticker
            logger.Infof("Initiating GroupInfo user cache reset (group_id > 0)...")
            manager.ResetUserCache() // Call the NEW reset function on the manager instance
            logger.Infof("GroupInfo user cache reset completed (group_id > 0).")
        }
    }()
}

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

// StartCacheCleanup starts the periodic user cache reset goroutine.
func StartCacheCleanup() {
    startCacheCleanup(groupInfoManager)
}
