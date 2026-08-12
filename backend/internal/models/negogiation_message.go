package models

import "time"

type NegotiationMessage struct {
	ID            int
	NegotiationID int
	SenderID      int
	OfferPrice    float64
	Message       string
	CreatedAt     time.Time
}
