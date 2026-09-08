package models

import (
	"encoding/json"
	"time"
)

// MarketEvent records marketplace activity used to generate
// transparent demand and market-behaviour signals.
//
// A market event may optionally belong to:
//   - a product
//   - an authenticated user
//   - an anonymous/session-based visitor
//
// The event itself is intentionally generic so the marketplace
// can record activities such as:
//
//   - product_view
//   - cart_add
//   - negotiation_started
//   - negotiation_accepted
//   - order_created
//   - checkout_abandoned
//
// ProductID and UserID are pointers because the database allows
// them to be NULL.
type MarketEvent struct {
	ID        int             `json:"id"`
	EventType string          `json:"event_type"`
	ProductID *int            `json:"product_id,omitempty"`
	UserID    *int            `json:"user_id,omitempty"`
	SessionID *string         `json:"session_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewMarketEvent creates a market event with an empty JSON object
// when no additional metadata is supplied.
func NewMarketEvent(eventType string) *MarketEvent {
	return &MarketEvent{
		EventType: eventType,
		Metadata:  json.RawMessage(`{}`),
	}
}

// SetMetadata replaces the event metadata with any JSON-compatible
// value.
func (e *MarketEvent) SetMetadata(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	e.Metadata = json.RawMessage(data)
	return nil
}

// HasMetadata reports whether the event contains usable metadata.
func (e *MarketEvent) HasMetadata() bool {
	return len(e.Metadata) > 0 && string(e.Metadata) != "null"
}

// IsProductEvent reports whether this event is associated with
// a marketplace product.
func (e *MarketEvent) IsProductEvent() bool {
	return e.ProductID != nil
}

// IsUserEvent reports whether this event is associated with
// an authenticated user.
func (e *MarketEvent) IsUserEvent() bool {
	return e.UserID != nil
}