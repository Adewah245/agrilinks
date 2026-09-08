package models

import "time"

// CartItem represents one product currently in a buyer's shopping cart.
//
// The cart item stores the agreed purchase quantity and price.
// When the item comes from an accepted negotiation, NegotiationID
// identifies that negotiation and PricePerUnit preserves the agreed price.
type CartItem struct {
	ID int `json:"id"`

	// Relationships.
	BuyerID       int  `json:"buyer_id"`
	ProductID     int  `json:"product_id"`
	NegotiationID *int `json:"negotiation_id,omitempty"`

	// Purchase information.
	Quantity     float64 `json:"quantity"`
	PricePerUnit float64 `json:"price_per_unit"`
	TotalPrice   float64 `json:"total_price"`

	// Timestamps.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// ============================================================
	// PRODUCT INFORMATION
	// These fields are populated through database joins.
	// They are not stored directly in cart_items.
	// ============================================================

	ProductName string `json:"product_name"`
	Unit        string `json:"unit"`
	ImageURL    string `json:"image_url"`

	// ============================================================
	// SELLER INFORMATION
	// These fields are populated from the product owner.
	// They are not stored directly in cart_items.
	// ============================================================

	SellerID       int    `json:"seller_id"`
	SellerName     string `json:"seller_name"`
	SellerEmail    string `json:"seller_email"`
	SellerPhone    string `json:"seller_phone"`
	SellerLocation string `json:"seller_location"`
	SellerPhotoURL string `json:"seller_photo_url"`

	// ============================================================
	// BUYER INFORMATION
	// These fields are populated from the buyer account.
	// They are not stored directly in cart_items.
	// ============================================================

	BuyerName     string `json:"buyer_name"`
	BuyerEmail    string `json:"buyer_email"`
	BuyerPhone    string `json:"buyer_phone"`
	BuyerLocation string `json:"buyer_location"`
	BuyerPhotoURL string `json:"buyer_photo_url"`
}
