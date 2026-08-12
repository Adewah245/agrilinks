// Package memory provides in-memory implementations of the domain
// repositories, useful for local development and tests before a real
// Postgres-backed implementation is wired in.
package memory

import (
	"context"
	"sync"
	"time"

	"backend/internal2/domain"
)

type PriceRepo struct {
	mu          sync.RWMutex
	reports     map[string]*domain.PriceReport
	aggregates  map[string]*domain.PriceAggregate // key: commodityID|marketID
}

func NewPriceRepo() *PriceRepo {
	return &PriceRepo{
		reports:    make(map[string]*domain.PriceReport),
		aggregates: make(map[string]*domain.PriceAggregate),
	}
}

func aggKey(commodityID, marketID string) string {
	return commodityID + "|" + marketID
}

func (r *PriceRepo) SaveReport(ctx context.Context, rep *domain.PriceReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[rep.ID] = rep
	return nil
}

func (r *PriceRepo) GetReport(ctx context.Context, reportID string) (*domain.PriceReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rep, ok := r.reports[reportID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	// Return a copy so callers can't mutate internal state directly.
	repCopy := *rep
	return &repCopy, nil
}

func (r *PriceRepo) UpdateReportStatus(ctx context.Context, reportID string, status domain.ReportStatus, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rep, ok := r.reports[reportID]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now()
	rep.Status = status
	rep.FlagReason = reason
	rep.ReviewedAt = &now
	return nil
}

func (r *PriceRepo) RecentAcceptedPrices(ctx context.Context, commodityID, marketID string, since time.Time) ([]domain.PriceReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.PriceReport
	for _, rep := range r.reports {
		if rep.CommodityID != commodityID || rep.MarketID != marketID {
			continue
		}
		if rep.Status != domain.StatusAccepted {
			continue
		}
		if rep.SubmittedAt.Before(since) {
			continue
		}
		out = append(out, *rep)
	}
	return out, nil
}

func (r *PriceRepo) CurrentAggregate(ctx context.Context, commodityID, marketID string) (*domain.PriceAggregate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agg, ok := r.aggregates[aggKey(commodityID, marketID)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return agg, nil
}

func (r *PriceRepo) SaveAggregate(ctx context.Context, agg *domain.PriceAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aggregates[aggKey(agg.CommodityID, agg.MarketID)] = agg
	return nil
}
