package storage

import (
	"time"

	"tg-antispam/internal/models"

	"gorm.io/gorm"
)

// BanRepository handles database operations for BanRecord
type BanRepository struct {
	db *gorm.DB
}

// NewBanRepository creates a new BanRepository
func NewBanRepository(db *gorm.DB) *BanRepository {
	return &BanRepository{db: db}
}

// MigrateTable ensures the BanRecord table exists
func (r *BanRepository) MigrateTable() error {
	return r.db.AutoMigrate(&models.BanRecord{})
}

// Create inserts a new BanRecord
func (r *BanRepository) Create(record *models.BanRecord) error {
	return r.db.Create(record).Error
}

// GetActiveRecordsByUser returns all non-unbanned records for a user
func (r *BanRepository) GetActiveRecordsByUser(userID int64, groupID int64) ([]*models.BanRecord, error) {
	var records []*models.BanRecord
	var result *gorm.DB
	if groupID != -1 {
		result = r.db.Where("user_id = ? AND group_id = ? AND is_unbanned = ?", userID, groupID, false).Find(&records)
	} else {
		result = r.db.Where("user_id = ? AND is_unbanned = ?", userID, false).Find(&records)
	}
	return records, result.Error
}

// UnbanUserByGroup unban user in a group
func (r *BanRepository) UnbanUserByGroup(groupID, userID int64, unbannedBy string) error {
	result := r.db.Model(&models.BanRecord{}).
		Where("group_id = ? AND user_id = ? AND is_unbanned = ?", groupID, userID, false).
		Updates(map[string]interface{}{"is_unbanned": true, "updated_at": time.Now(), "unbanned_by": unbannedBy})
	return result.Error
}
