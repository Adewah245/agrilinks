package models

import "time"

// DeliveryPartner represents a logistics provider registered on Agro-Shield.
type DeliveryPartner struct {
	ID                                                  int
	OrganizationID                                      *int
	Name, Phone, Email, VehicleType, RegistrationNumber string
	StateID, LGAID                                      *int
	Verified                                            bool
	Status                                              string
	CreatedAt, UpdatedAt                                time.Time

	// MemberRole is populated when a partner is loaded for a user account.
	MemberRole string
}

// Delivery is the fulfilment record for one order.
type Delivery struct {
	ID, OrderID       int
	BuyerID           int
	DeliveryPartnerID *int

	PickupAddress          string
	PickupStateID          *int
	PickupLGAID            *int
	PickupCommunityID      *int
	DestinationAddress     string
	DestinationStateID     *int
	DestinationLGAID       *int
	DestinationCommunityID *int

	DeliveryFee float64
	DistanceKM  *float64

	Status       string
	OrderStatus  string
	TrackingCode string

	PickedUpAt, DeliveredAt *time.Time
	CreatedAt, UpdatedAt    time.Time

	// Derived display fields populated by delivery queries.
	PartnerName string
	BuyerName   string
}

// DeliveryOrderOption is a paid order that a buyer can still request delivery
// for. ProductNames is display text only; delivery remains one record per order.
type DeliveryOrderOption struct {
	ID, BuyerID  int
	Status       string
	ProductNames string
	SellerName   string
	CreatedAt    time.Time
}

// DeliveryOrderContext is the information needed to snapshot addresses when a
// buyer requests delivery for an order.
type DeliveryOrderContext struct {
	OrderID, BuyerID int
	Status           string

	BuyerName, BuyerLocation, BuyerLGA, BuyerState, BuyerCountry string

	PickupAddress     string
	PickupStateID     *int
	PickupLGAID       *int
	PickupCommunityID *int
	SellerName        string
}

// DeliveryPartnerInput is the user-editable part of partner registration.
type DeliveryPartnerInput struct {
	Name, Phone, Email, VehicleType, RegistrationNumber string
	StateID, LGAID                                      *int
}

// DeliveryTracking is an immutable location update for a delivery.
type DeliveryTracking struct {
	ID, DeliveryID      int
	Latitude, Longitude float64
	LocationName, Note  string
	CreatedAt           time.Time
}
