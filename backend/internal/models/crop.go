package models

import "time"

type Crop struct {
	ID            int
	FarmerID      int
	Name          string
	Quantity      float64
	Unit          string
	Location      string
	PricePerUnit  float64
	ListedForSale bool
	ImageURL      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	// Populated only when listing marketplace results, not stored on the row itself.
	SellerName string
}
