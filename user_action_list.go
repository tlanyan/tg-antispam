package main

import (
	"sync"
	"time"
)

// UserActionInfo represents a banned user with an expiration time
type UserActionInfo struct {
	UserID    int64
	ExpiresAt time.Time
}

// UserActionManager manages the list of banned users with expirations
type UserActionManager struct {
	users           map[int64]time.Time
	expirationHours int
	mu              sync.RWMutex
}

// NewUserActionManager creates a new banned users list
func NewUserActionManager(expirationHours int) *UserActionManager {
	list := &UserActionManager{
		users:           make(map[int64]time.Time),
		expirationHours: expirationHours,
	}

	// Start a goroutine to clean up expired entries
	go list.cleanupExpired()

	return list
}

// Add adds a user to the banned list with expiration
func (b *UserActionManager) Add(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set expiration based on configured hours
	b.users[userID] = time.Now().Add(time.Duration(b.expirationHours) * time.Hour)
}

// Remove removes a user from the banned list
func (b *UserActionManager) Remove(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.users, userID)
}

// Contains checks if a user is in the banned list
func (b *UserActionManager) Contains(userID int64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiry, exists := b.users[userID]
	if !exists {
		return false
	}

	// If expired, remove and return false
	if time.Now().After(expiry) {
		// Use a goroutine to avoid deadlock when removing while holding a read lock
		go b.Remove(userID)
		return false
	}

	return true
}

// cleanupExpired periodically removes expired entries
func (b *UserActionManager) cleanupExpired() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for userID, expiry := range b.users {
			if now.After(expiry) {
				delete(b.users, userID)
			}
		}
		b.mu.Unlock()
	}
}
