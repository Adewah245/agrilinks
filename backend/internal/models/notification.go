package models

import "time"

// ============================================================
// NOTIFICATION MODEL
//
// Responsibility:
// - Represents one notification in Agro-Shield.
// - Matches the notifications table in PostgreSQL.
// - Carries notification data between our application layers.
// ============================================================

type Notification struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}