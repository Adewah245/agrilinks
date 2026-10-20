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
	TrackingCode string

	PickedUpAt, DeliveredAt *time.Time
	CreatedAt, UpdatedAt    time.Time

	// Derived display fields populated by delivery queries.
	PartnerName string
	BuyerName   string
}

// DeliveryTracking is an immutable location update for a delivery.
type DeliveryTracking struct {
	ID, DeliveryID      int
	Latitude, Longitude float64
	LocationName, Note  string
	CreatedAt           time.Time
}
