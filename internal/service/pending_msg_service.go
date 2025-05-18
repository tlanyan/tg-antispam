package service

import "tg-antispam/internal/storage"

// GetPendingMsgRepository returns the pending message repository
func GetPendingMsgRepository() *storage.PendingMsgRepository {
	return pendingMsgRepository
}
