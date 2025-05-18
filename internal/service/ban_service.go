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

// GetActiveBanRecordsByUser retrieves all active (not unbanned) ban records for a user
func GetActiveBanRecordsByUser(userID int64) ([]*models.BanRecord, error) {
	if banRepository != nil {
		return banRepository.GetActiveByUser(userID)
	}
	return nil, nil
}

// MarkBanRecordUnbanned marks a user's ban record as unbanned for a specific group
func MarkBanRecordUnbanned(groupID, userID int64, unbannedBy string) {
	if banRepository != nil {
		if err := banRepository.MarkUnbanned(groupID, userID, unbannedBy); err != nil {
			logger.Warningf("Error marking ban record unbanned: %v", err)
		}
	}
}
