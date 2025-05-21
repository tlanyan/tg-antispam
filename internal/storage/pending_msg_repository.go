package storage

import (
	"tg-antispam/internal/models"

	"gorm.io/gorm"
)

// PendingDeletionRepository handles database operations for PendingMessageDeletion
type PendingMsgRepository struct {
	db *gorm.DB
}

// NewPendingDeletionRepository creates a new PendingDeletionRepository
func NewPendingMsgRepository(db *gorm.DB) *PendingMsgRepository {
	return &PendingMsgRepository{db: db}
}

// MigrateTable ensures the PendingMessageDeletion table exists
func (r *PendingMsgRepository) MigrateTable() error {
	return r.db.AutoMigrate(&models.PendingMessage{})
}

// AddPendingMsg adds a new pending message record
func (r *PendingMsgRepository) AddPendingMsg(pm *models.PendingMessage) error {
	return r.db.Create(pm).Error
}

// RemovePendingMsg removes a pending message record by ChatID and MessageID
func (r *PendingMsgRepository) RemovePendingMsg(chatID int64, messageID int) error {
	return r.db.Where("chat_id = ? AND message_id = ?", chatID, messageID).Delete(&models.PendingMessage{}).Error
}

// GetAllPendingMsgs retrieves all pending message records
func (r *PendingMsgRepository) GetAllPendingMsgs() ([]models.PendingMessage, error) {
	var msgs []models.PendingMessage
	result := r.db.Find(&msgs)
	return msgs, result.Error
}

func (r *PendingMsgRepository) GetPendingMsgsByUserID(userID int64, chatID int64) ([]models.PendingMessage, error) {
	var msgs []models.PendingMessage
	result := r.db.Where("user_id = ? AND chat_id = ?", userID, chatID).Find(&msgs)
	return msgs, result.Error
}
