package models

import "time"

// Transaction records one payment attempt for an order. Amounts are stored in
// PostgreSQL NUMERIC columns; float64 is used here only at the existing UI/API
// boundary, where all monetary values are formatted to two decimals.
type Transaction struct {
	ID               int
	OrderID          int
	BuyerID          int
	SellerType       string
	SellerID         int
	Amount           float64
	Currency         string
	PaymentMethod    string
	PaymentReference string
	PaymentProvider  string
	Status           string
	PaidAt           *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ProductName      string
	SellerName       string
}

type Wallet struct {
	ID        int
	OwnerType string
	OwnerID   int
	Balance   float64
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WalletTransaction struct {
	ID              int
	WalletID        int
	TransactionType string
	Amount          float64
	Description     string
	ReferenceType   string
	ReferenceID     *int
	CreatedAt       time.Time
}
