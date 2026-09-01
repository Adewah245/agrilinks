package models

import "time"

// ============================================================
// NEGOTIATION MESSAGE MODEL
//
// Responsibility:
// - Represents one message inside a negotiation.
// - A message can be normal chat.
// - A message can also contain a price offer.
// - Keeps track of the type of message.
// - Keeps track of offer status when the message is an offer.
//
// Message types:
//
//   "chat"  → normal conversation message
//   "offer" → message containing a price offer
//
// Offer statuses:
//
//   "pending"  → offer is waiting for a response
//   "accepted" → offer was accepted
//   "rejected" → offer was rejected
//   "none"     → normal chat message has no offer status
//
// This model is used by:
//
// Repository → Service → Handler → Frontend
// ============================================================

type NegotiationMessage struct {
	ID            int       `json:"id"`
	NegotiationID int       `json:"negotiation_id"`
	SenderID      int       `json:"sender_id"`

	MessageType string `json:"message_type"`

	OfferPrice float64 `json:"offer_price"`
	Message    string  `json:"message"`
	OfferStatus string  `json:"offer_status"`

	CreatedAt time.Time `json:"created_at"`
}