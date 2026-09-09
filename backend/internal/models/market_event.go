package models

import (
	"encoding/json"
	"time"
)

// MarketEvent records marketplace activity used to generate
// transparent demand and market-behaviour signals.
//
// A market event may optionally belong to:
//   - a product
//   - an authenticated user
//   - an anonymous/session-based visitor
//
// The event itself is intentionally generic so the marketplace
// can record activities such as:
//
//   - product_view
//   - cart_add
//   - negotiation_started
//   - negotiation_accepted
//   - order_created
//   - checkout_abandoned
//
// ProductID and UserID are pointers because the database allows
// them to be NULL.
type MarketEvent struct {
	ID        int             `json:"id"`
	EventType string          `json:"event_type"`
	ProductID *int            `json:"product_id,omitempty"`
	UserID    *int            `json:"user_id,omitempty"`
	SessionID *string         `json:"session_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

// ProduceDemandSummary is the dashboard projection built from marketplace
// events, products, negotiations, and external price data. It is not a table;
// the repository populates it from the schema-backed aggregate queries.
type ProduceDemandSummary struct {
	ProduceType               string  `json:"produce_type"`
	Views                     int     `json:"views"`
	CartAdds                  int     `json:"cart_adds"`
	Negotiations              int     `json:"negotiations"`
	Orders                    int     `json:"orders"`
	UnitsOrdered              float64 `json:"units_ordered"`
	UnitsAdded                float64 `json:"units_added"`
	CurrentListings           int     `json:"current_listings"`
	Searches                  int     `json:"searches"`
	TrendPercent              float64 `json:"trend_percent"`
	Activity                  string  `json:"activity"`
	DemandScore               float64 `json:"demand_score"`
	ComparisonLocation        string  `json:"comparison_location"`
	ComparisonScope           string  `json:"comparison_scope"`
	LocalListings             int     `json:"local_listings"`
	LocalPriceMin             float64 `json:"local_price_min"`
	LocalPriceMedian          float64 `json:"local_price_median"`
	LocalPriceMax             float64 `json:"local_price_max"`
	StateListings             int     `json:"state_listings"`
	StatePriceMedian          float64 `json:"state_price_median"`
	NearbyListings            int     `json:"nearby_listings"`
	NearbyPriceMedian         float64 `json:"nearby_price_median"`
	NationalListings          int     `json:"national_listings"`
	NationalPriceMedian       float64 `json:"national_price_median"`
	ExternalListings          int     `json:"external_listings"`
	ExternalPriceMedian       float64 `json:"external_price_median"`
	FarmerPrice               float64 `json:"farmer_price"`
	PriceAvailable            bool    `json:"price_available"`
	ExternalPriceAvailable    bool    `json:"external_price_available"`
	PriceDifferencePercent    float64 `json:"price_difference_percent"`
	PricePosition             string  `json:"price_position"`
	Recommendation            string  `json:"recommendation"`
	NegotiationsActive        int     `json:"negotiations_active"`
	NegotiationsAccepted      int     `json:"negotiations_accepted"`
	NegotiationsRejected      int     `json:"negotiations_rejected"`
	NegotiatedUnits           float64 `json:"negotiated_units"`
	AverageOfferPrice         float64 `json:"average_offer_price"`
	SellThroughRate           float64 `json:"sell_through_rate"`
	TimeToFirstOrderHours     float64 `json:"time_to_first_order_hours"`
	TimeToSellOutHours        float64 `json:"time_to_sell_out_hours"`
	LifecycleDataAvailable    bool    `json:"lifecycle_data_available"`
	NegotiationConversionRate float64 `json:"negotiation_conversion_rate"`
}

// NewMarketEvent creates a market event with an empty JSON object
// when no additional metadata is supplied.
func NewMarketEvent(eventType string) *MarketEvent {
	return &MarketEvent{
		EventType: eventType,
		Metadata:  json.RawMessage(`{}`),
	}
}

// SetMetadata replaces the event metadata with any JSON-compatible
// value.
func (e *MarketEvent) SetMetadata(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	e.Metadata = json.RawMessage(data)
	return nil
}

// HasMetadata reports whether the event contains usable metadata.
func (e *MarketEvent) HasMetadata() bool {
	return len(e.Metadata) > 0 && string(e.Metadata) != "null"
}

// IsProductEvent reports whether this event is associated with
// a marketplace product.
func (e *MarketEvent) IsProductEvent() bool {
	return e.ProductID != nil
}

// IsUserEvent reports whether this event is associated with
// an authenticated user.
func (e *MarketEvent) IsUserEvent() bool {
	return e.UserID != nil
}
