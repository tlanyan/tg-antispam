package models

import "time"

// PendingMessage represents a message scheduled for deletion.
type PendingMessage struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	ChatID    int64     `gorm:"index:idx_chat_message,unique"`
	MessageID int       `gorm:"index:idx_chat_message,unique"`
	DeleteAt  time.Time `gorm:"index"` // Optional: if you want to store when it was originally scheduled
}
