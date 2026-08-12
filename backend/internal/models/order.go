package models

import "time"

type Order struct {
	ID         int
	BuyerID    int
	CropID     int
	Quantity   float64
	TotalPrice float64
	Status     string
	CreatedAt  time.Time

	// Populated only for display, not stored on the row itself.
	CropName   string
	SellerName string
	BuyerName  string
}
