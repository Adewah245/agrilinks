package models

import "time"

// ConversationMember represents one user belonging to a conversation.
//
// The actual membership data comes from conversation_members.
// FullName and PhotoURL are derived from the users table when the
// repository performs a JOIN.
type ConversationMember struct {
	ID int `json:"id"`

	ConversationID int `json:"conversation_id"`
	UserID         int `json:"user_id"`

	// MemberRole can be:
	//   - member
	//   - admin
	//   - owner
	MemberRole string `json:"member_role"`

	JoinedAt time.Time `json:"joined_at"`

	// LeftAt is NULL while the user is still an active member.
	LeftAt *time.Time `json:"left_at,omitempty"`

	// LastReadAt is the point up to which this member has read the
	// conversation. It drives both seen receipts and unread counts.
	LastReadAt time.Time `json:"last_read_at"`

	// ============================================================
	// DERIVED / JOINED USER INFORMATION
	//
	// These fields are not stored in conversation_members.
	// They come from the users table.
	// ============================================================

	FullName string `json:"full_name"`
	PhotoURL string `json:"photo_url"`
}
