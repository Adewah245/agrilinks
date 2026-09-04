package models

import "time"

// Order represents a purchase of a product.
type Order struct {
	ID       int
	BuyerID  int
	ProductID int

	Quantity   float64
	TotalPrice float64
	Status     string
	CreatedAt  time.Time

	// Populated only for display, not stored directly on the order row.
	ProductName string
	SellerName  string
	BuyerName   string
}
