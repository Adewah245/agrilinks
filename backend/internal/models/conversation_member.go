package models

import "time"

// ============================================================
// CONVERSATION MEMBER MODEL
//
// Responsibility:
// - Represents a user belonging to a conversation.
// - Also carries basic user information when members are loaded.
//
// IMPORTANT:
//
// UserID comes from conversation_members.
//
// FullName and PhotoURL come from the users table
// // ConversationMember is loaded with a JOIN.
//
// Example:
//
// Maize Farmers Association
//
//     conversation_id = 5
//
//     user_id = 1  → James Adah
//     user_id = 7  → John Oche
//     user_id = 12 → Peter Ameh
//
// Every row still represents ONE member.
// ============================================================

// ConversationMember represents a user who belongs to a conversation.
type ConversationMember struct {
	ID int `json:"id"`

	ConversationID int `json:"conversation_id"`

	UserID int `json:"user_id"`

	// FullName and PhotoURL are retrieved from the users table.
	FullName string `json:"full_name"`

	PhotoURL string `json:"photo_url"`

	JoinedAt time.Time `json:"joined_at"`
}