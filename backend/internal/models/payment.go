package models

import (
	"database/sql"
	"time"
)

// Payment records one payment attempt for an order. Amounts are kept as
// strings at this boundary so provider-supplied decimal values are not rounded
// by floating-point conversion.
type Payment struct {
	ID            int64
	OrderID       int64
	Reference     string
	TransactionID sql.NullString
	Amount        string
	Currency      string
	Status        string
	Provider      string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// Display-only fields populated through joins.
	BuyerID     int
	ProductName string
	SellerName  string
}

const (
	PaymentPending    = "pending"
	PaymentSuccessful = "successful"
	PaymentFailed     = "failed"
)
