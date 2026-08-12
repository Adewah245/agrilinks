package domain

import (
	"context"
	"time"
)

// PriceRepository persists price reports and reads recent history needed
// for outlier detection and aggregation. Implementations: in-memory (now),
// Postgres (later) — the service layer never changes.
type PriceRepository interface {
	SaveReport(ctx context.Context, r *PriceReport) error
	GetReport(ctx context.Context, reportID string) (*PriceReport, error)
	UpdateReportStatus(ctx context.Context, reportID string, status ReportStatus, reason string) error

	// RecentAcceptedPrices returns accepted prices for a commodity/market
	// within the given lookback window, used for outlier comparison and
	// trend/aggregate computation.
	RecentAcceptedPrices(ctx context.Context, commodityID, marketID string, since time.Time) ([]PriceReport, error)

	CurrentAggregate(ctx context.Context, commodityID, marketID string) (*PriceAggregate, error)
	SaveAggregate(ctx context.Context, agg *PriceAggregate) error
}

// SubmitterRepository manages submitters and their trust scores.
type SubmitterRepository interface {
	Get(ctx context.Context, submitterID string) (*Submitter, error)
	Save(ctx context.Context, s *Submitter) error
}
