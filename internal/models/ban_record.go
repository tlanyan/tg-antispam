package models

import "time"

// BanRecord stores information about user bans and unbans
// It records the group, user, reason, and unban status
// along with creation and update timestamps.
type BanRecord struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	GroupID    int64  `gorm:"index;not null"`
	UserID     int64  `gorm:"index;not null"`
	Reason     string `gorm:"type:text"`
	IsUnbanned bool   `gorm:"default:false"`
	UnbannedBy string `gorm:"default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
