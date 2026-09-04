package chat

import (
	"sync"
	"time"
)

// Tracker handles real-time user presence with thread safety.
type Tracker struct {
	mu           sync.RWMutex
	activeUsers  map[int64]time.Time
	expiryWindow time.Duration
}

// NewTracker initializes a presence tracker with a timeout threshold (e.g., 45 seconds).
func NewTracker(expiryWindow time.Duration) *Tracker {
	return &Tracker{
		activeUsers:  make(map[int64]time.Time),
		expiryWindow: expiryWindow,
	}
}

// MarkActive updates or adds a user's last active timestamp.
func (t *Tracker) MarkActive(userID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activeUsers[userID] = time.Now()
}

// MarkInactive manually removes a user upon WebSocket disconnect or logout.
func (t *Tracker) MarkInactive(userID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.activeUsers, userID)
}

// GetOnlineUserIDs returns a slice of user IDs active within the expiry window.
func (t *Tracker) GetOnlineUserIDs() []int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	onlineIDs := make([]int64, 0, len(t.activeUsers))

	for userID, lastSeen := range t.activeUsers {
		if now.Sub(lastSeen) <= t.expiryWindow {
			onlineIDs = append(onlineIDs, userID)
		}
	}

	return onlineIDs
}
