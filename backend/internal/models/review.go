package models

import "time"

type ProductReview struct {
	ID, ProductID, ReviewerID, OrderID, Rating int
	Comment                                    string
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
