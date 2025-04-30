package main

import (
	"log"
	"sync"
	"time"
)

// BannedUserInfo represents a banned user with an expiration time
type BannedUserInfo struct {
	UserID    int64
	ExpiresAt time.Time
}

// BannedUserManager manages the list of banned users with expirations
type BannedUserManager struct {
	users map[int64]time.Time
	mu    sync.RWMutex
}

// NewBannedUserManager creates a new banned users list
func NewBannedUserManager() *BannedUserManager {
	list := &BannedUserManager{
		users: make(map[int64]time.Time),
	}

	// Start a goroutine to clean up expired entries
	go list.cleanupExpired()

	return list
}

// Add adds a user to the banned list with 2-hour expiration
func (b *BannedUserManager) Add(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set expiration to 2 hours from now
	b.users[userID] = time.Now().Add(2 * time.Hour)
	log.Printf("Added user %d to banned list, expires at %v", userID, b.users[userID])
}

// Remove removes a user from the banned list
func (b *BannedUserManager) Remove(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.users, userID)
	log.Printf("Removed user %d from banned list", userID)
}

// Contains checks if a user is in the banned list
func (b *BannedUserManager) Contains(userID int64) bool {
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
func (b *BannedUserManager) cleanupExpired() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for userID, expiry := range b.users {
			if now.After(expiry) {
				delete(b.users, userID)
				log.Printf("Expired ban for user %d", userID)
			}
		}
		b.mu.Unlock()
	}
}

// Global banned users list instance
var BannedUsers = NewBannedUserManager()
