package models

import "time"

type ProductReview struct {
	ID, ProductID, ReviewerID, OrderID, Rating int
	ReviewerName, Comment                      string
	CreatedAt, UpdatedAt                       time.Time
}
type UserReview struct {
	ID                                      int
	ReviewedType                            string
	ReviewedID, ReviewerID, OrderID, Rating int
	Comment                                 string
	CreatedAt                               time.Time
}
type Verification struct {
	ID, UserID       int
	NIN              string
	NINVerified      bool
	Reference, Level string
	VerifiedAt       *time.Time
	CreatedAt        time.Time
}
type VerificationBadge struct {
	ID          int
	OwnerType   string
	OwnerID     int
	BadgeType   string
	IssuedBy    *int
	Description string
	IssuedAt    time.Time
	ExpiresAt   *time.Time
	Active      bool
}

// TrustSummary is the public trust information shown for a product or seller.
// It deliberately contains aggregates and badges only, never verification data.
type TrustSummary struct {
	AverageRating float64
	ReviewCount   int
	Badges        []VerificationBadge
}
