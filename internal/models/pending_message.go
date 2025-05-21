package models

import "time"

// PendingMessage represents a message scheduled for deletion.
type PendingMessage struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID    int64 `gorm:"index:idx_user_chat"`
	ChatID    int64 `gorm:"index:idx_user_chat;index:idx_chat_message,unique"`
	MessageID int   `gorm:"index:idx_chat_message,unique"`
}
