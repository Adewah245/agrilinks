package models

import "time"

// NegotiationMessage represents one permanent message or offer
// inside a negotiation.
//
// Normal messages contain text and no offer price.
// Offer messages contain an offer price and an offer status.
// Existing messages and offers are never overwritten.
type NegotiationMessage struct {
	ID            int       `json:"id"`
	NegotiationID int       `json:"negotiation_id"`
	SenderID      int       `json:"sender_id"`

	// Message type:
	// "message" or "offer".
	MessageType string `json:"message_type"`

	// Message is optional because an offer may contain no
	// explanatory text.
	Message *string `json:"message,omitempty"`

	// OfferPrice is NULL for ordinary messages and populated
	// for offer messages.
	OfferPrice *float64 `json:"offer_price,omitempty"`

	// Offer status:
	// "none", "pending", "accepted", or "rejected".
	OfferStatus string `json:"offer_status"`

	CreatedAt time.Time `json:"created_at"`
}
