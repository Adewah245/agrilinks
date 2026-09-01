package models

import "time"

// CartItem represents a product that a buyer has agreed
// to purchase and is currently waiting for checkout.
//
// The cart item is created from an accepted negotiation.
// The negotiated price is preserved so that a later marketplace
// price change does not change the agreed deal.
type CartItem struct {
	ID int

	// IDs used to connect the cart item with other records.
	BuyerID       int
	CropID        int
	NegotiationID int

	// Purchase information.
	Quantity     float64
	PricePerUnit float64
	TotalPrice   float64

	CreatedAt time.Time

	// ========================================================
	// PRODUCT INFORMATION
	// Populated through database joins.
	// Not stored directly in cart_items.
	// ========================================================

	CropName string
	Unit     string
	ImageURL string

	// ========================================================
	// SELLER INFORMATION
	// Populated from the farmer who owns the crop.
	// Not stored directly in cart_items.
	// ========================================================

	SellerID       int
	SellerName     string
	SellerEmail    string
	SellerPhone    string
	SellerLocation string
	SellerPhotoURL string

	// ========================================================
	// BUYER INFORMATION
	// Populated from the farmer/buyer account.
	// Not stored directly in cart_items.
	// ========================================================

	BuyerName     string
	BuyerEmail    string
	BuyerPhone    string
	BuyerLocation string
	BuyerPhotoURL string
}