package domain

import "time"

// SubmitterRole distinguishes who reported a price, since trust and
// verification rules differ by role.
type SubmitterRole string

const (
	RoleFarmer     SubmitterRole = "farmer"
	RoleFieldAgent SubmitterRole = "field_agent"
	RoleAdmin      SubmitterRole = "admin"
)

// ReportStatus tracks a submitted price through moderation.
type ReportStatus string

const (
	StatusPending  ReportStatus = "pending"  // awaiting outlier/trust check
	StatusAccepted ReportStatus = "accepted" // counted into aggregates
	StatusFlagged  ReportStatus = "flagged"  // suspicious, held for review
	StatusRejected ReportStatus = "rejected"
)

// Commodity is a sellable crop/product type, e.g. Maize, Cassava.
type Commodity struct {
	ID   string
	Name string
	Unit string // e.g. "bag_100kg", "kg", "ton"
}

// Market is a physical or regional market location.
type Market struct {
	ID     string
	Name   string
	Region string
	State  string
}

// Submitter is whoever reports a price (a farmer, field agent, or admin).
// TrustScore drives how much weight their reports carry and how quickly
// their submissions get auto-accepted vs held for review.
type Submitter struct {
	ID              string
	Role            SubmitterRole
	Verified        bool
	TrustScore      float64 // 0.0 - 1.0
	TotalReports    int
	AcceptedReports int
}

// PriceReport is a single crowd-sourced price submission.
type PriceReport struct {
	ID          string
	CommodityID string
	MarketID    string
	SubmitterID string
	Price       float64
	Currency    string
	Unit        string
	Status      ReportStatus
	FlagReason  string // populated when Status == Flagged/Rejected
	SubmittedAt time.Time
	ReviewedAt  *time.Time
}

// PriceAggregate is a precomputed daily summary for a commodity/market pair,
// built only from Accepted reports.
type PriceAggregate struct {
	CommodityID string
	MarketID    string
	Date        time.Time
	AvgPrice    float64
	MinPrice    float64
	MaxPrice    float64
	SampleSize  int
}
