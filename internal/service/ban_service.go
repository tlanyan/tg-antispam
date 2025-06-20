package service

import (
	"tg-antispam/internal/logger"
	"tg-antispam/internal/models"
)

// CreateBanRecord stores a new ban record for the user in a group
func CreateBanRecord(groupID, userID int64, reason string) {
	if banRepository != nil {
		record := &models.BanRecord{GroupID: groupID, UserID: userID, Reason: reason}
		if err := banRepository.Create(record); err != nil {
			logger.Warningf("Error creating ban record: %v", err)
		}
	}
}

// GetUserActiveBanRecords retrieves all active ban records for a user
func GetUserActiveBanRecords(userID int64, groupID int64) ([]*models.BanRecord, error) {
	if banRepository != nil {
		return banRepository.GetActiveRecordsByUser(userID, groupID)
	}
	return nil, nil
}

// UnbanUserInGroup unban user in a group
func UnbanUserInGroup(groupID, userID int64, unbannedBy string) {
	if banRepository != nil {
		if err := banRepository.UnbanUserByGroup(groupID, userID, unbannedBy); err != nil {
			logger.Warningf("Error marking ban record unbanned: %v", err)
		}
	}
}
