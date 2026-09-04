package models

import "time"

// Negotiation represents a price negotiation between a buyer and a product owner.
type Negotiation struct {
	ID int

	// Product being negotiated.
	ProductID int

	// Buyer participating in the negotiation.
	BuyerID int

	// Owner of the product.
	OwnerType string
	OwnerID   int

	// Quantity requested by the buyer.
	Quantity float64

	// Negotiation lifecycle status.
	Status string // "open", "accepted", "rejected", "expired"

	// Negotiation round tracking.
	RoundCount int
	MaxRounds  int

	// Negotiation timestamps.
	CreatedAt time.Time
	ExpiresAt time.Time

	// Populated for display only.
	ProductName   string
	BuyerName     string
	SellerName    string
	BuyerPhotoURL string
	SellerPhotoURL string
}

// TimeLeft returns the amount of time remaining before the negotiation expires.
func (n *Negotiation) TimeLeft() time.Duration {
	d := time.Until(n.ExpiresAt)

	if d < 0 {
		return 0
	}

	return d
}

// IsExpired reports whether the negotiation has passed its expiration time.
func (n *Negotiation) IsExpired() bool {
	return time.Now().After(n.ExpiresAt)
}

// RoundsLeft returns the number of negotiation rounds still available.
func (n *Negotiation) RoundsLeft() int {
	left := n.MaxRounds - n.RoundCount

	if left < 0 {
		return 0
	}

	return left
}