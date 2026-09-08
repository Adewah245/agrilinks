package models

import "time"

// Conversation represents a general Agro-Shield chat conversation.
//
// A conversation can be:
//   - private: one user communicating with another user
//   - group: multiple users communicating in one conversation
//
// This model is for general chat and is completely separate from
// negotiation conversations.
type Conversation struct {
	ID int `json:"id"`

	// Conversation type.
	//
	// Possible values:
	//   - private
	//   - group
	ConversationType string `json:"conversation_type"`

	// Title is primarily used for group conversations.
	// It is nullable in the database.
	Title *string `json:"title,omitempty"`

	// User who created the conversation.
	// The database allows this to become NULL if the user is deleted.
	CreatedBy *int `json:"created_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ============================================================
	// DERIVED / JOINED INFORMATION
	//
	// These fields are not stored directly in conversations.
	// They are populated by repository queries when needed.
	// ============================================================

	// Information about the other participant in a private
	// conversation.
	OtherUserID   int    `json:"other_user_id,omitempty"`
	OtherName     string `json:"other_name,omitempty"`
	OtherPhotoURL string `json:"other_photo_url,omitempty"`

	// Number of unread messages for the current user.
	UnreadCount int `json:"unread_count,omitempty"`
}
