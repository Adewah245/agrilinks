package models

import "time"

// ChatMessage represents one message inside a conversation.
//
// Chat messages are separate from NegotiationMessage.
// NegotiationMessage is specifically for buyer/seller negotiations,
// while ChatMessage is for normal Agro-Shield communication.
type ChatMessage struct {
	ID int `json:"id"`

	ConversationID int `json:"conversation_id"`
	SenderID       int `json:"sender_id"`

	// SenderName is populated through a user join.
	// It is not stored directly in chat_messages.
	SenderName string `json:"sender_name"`

	// Message content.
	//
	// For text messages, Message contains the text.
	// For media messages, Message may be nil.
	Message *string `json:"message,omitempty"`

	// MessageType is one of:
	// text, image, video, audio, document.
	MessageType string `json:"message_type"`

	// MediaURL is used by non-text messages.
	MediaURL *string `json:"media_url,omitempty"`

	// Editing state.
	IsEdited bool       `json:"is_edited"`
	EditedAt *time.Time `json:"edited_at,omitempty"`

	// Seen is derived by a repository query when read receipts are available.
	Seen bool `json:"seen"`

	CreatedAt time.Time `json:"created_at"`
}
