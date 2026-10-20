package models

import "time"

// Negotiation represents a price negotiation between a buyer and
// the owner of a product.
//
// The product owner is not stored directly on the negotiation.
// Ownership is derived through ProductID -> Product.OwnerType/OwnerID.
type Negotiation struct {
	ID int

	// Product being negotiated.
	ProductID int

	// Buyer participating in the negotiation.
	BuyerID int

	// Quantity requested by the buyer.
	Quantity float64

	// Offer pricing.
	//
	// InitialOfferPrice is the buyer's first offer.
	// CurrentOfferPrice is the latest offer in the negotiation.
	InitialOfferPrice float64
	CurrentOfferPrice float64

	// Negotiation lifecycle.
	Status string // "open", "accepted", "rejected", "expired", "closed"

	// Negotiation round tracking.
	RoundCount int
	MaxRounds  int

	// Negotiation timestamps.
	ExpiresAt  time.Time
	AcceptedAt *time.Time
	ClosedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Populated for display only.
	ProductName    string
	BuyerName      string
	SellerName     string
	BuyerPhotoURL  string
	SellerPhotoURL string
	// Owner fields are derived from the related product for authorization and
	// display; they are not columns on negotiations.
	OwnerType string
	OwnerID   int
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
