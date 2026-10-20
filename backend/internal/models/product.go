package models

import "time"

// Product represents an agricultural product listing.
type Product struct {
	ID int

	// Ownership.
	OwnerType string
	OwnerID   int

	// Product information.
	Name        string
	Description string
	Category    string
	Quantity    float64
	Unit        string

	// Pricing.
	Price float64

	// Location.
	Location         string
	StateID          *int
	LGAID            *int
	CommunityID      *int
	Latitude         float64
	Longitude        float64
	LocationAccuracy float64

	// Listing lifecycle.
	Status                string
	InitialListedQuantity float64
	FirstListedAt         *time.Time
	FirstOrderAt          *time.Time
	SoldOutAt             *time.Time

	// Images.
	Images []ProductImage

	// Marketplace information.
	SellerName     string
	SellerPhotoURL string

	// Timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsAvailable reports whether the product can currently
// be purchased from the marketplace.
func (p *Product) IsAvailable() bool {
	return p.Status == "active" && p.Quantity > 0
}

// IsSoldOut reports whether the product has no remaining quantity
// or has been explicitly marked as sold out.
func (p *Product) IsSoldOut() bool {
	return p.Status == "sold_out" || p.Quantity <= 0
}

// IsInactive reports whether the product is not currently listed.
func (p *Product) IsInactive() bool {
	return p.Status == "inactive"
}

// PrimaryImageURL returns the image that represents this product in
// marketplace and inventory lists. The primary image is preferred, with the
// first image used as a fallback for older records that have no primary flag.
func (p Product) PrimaryImageURL() string {
	for _, image := range p.Images {
		if image.IsPrimary && image.ImageURL != "" {
			return image.ImageURL
		}
	}

	for _, image := range p.Images {
		if image.ImageURL != "" {
			return image.ImageURL
		}
	}

	return ""
}

// ProductImage represents one picture belonging to a product.
type ProductImage struct {
	ID        int
	ProductID int
	ImageURL  string
	IsPrimary bool
	SortOrder int
	CreatedAt time.Time
}
