package service

import (
	"tg-antispam/internal/models"
)

func GetAllPendingMsgs() ([]models.PendingMessage, error) {
	if pendingMsgRepository == nil {
		return []models.PendingMessage{}, nil
	}
	return pendingMsgRepository.GetAllPendingMsgs()
}

func GetPendingMsgsByUserID(userID int64, chatID int64) ([]models.PendingMessage, error) {
	if pendingMsgRepository == nil {
		return []models.PendingMessage{}, nil
	}
	return pendingMsgRepository.GetPendingMsgsByUserID(userID, chatID)
}

func AddPendingMsg(pendingMsg *models.PendingMessage) error {
	if pendingMsgRepository == nil {
		return nil
	}
	return pendingMsgRepository.AddPendingMsg(pendingMsg)
}

func RemovePendingMsg(chatID int64, messageID int) error {
	if pendingMsgRepository == nil {
		return nil
	}
	return pendingMsgRepository.RemovePendingMsg(chatID, messageID)
}
