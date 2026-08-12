package service

import (
	"context"
	"fmt"
	"time"
	 "github.com/google/uuid"
	"backend/internal2/domain"
)

const (
	// lookbackWindow is how far back we look for comparable accepted
	// prices when checking a new submission for outliers.
	lookbackWindow = 14 * 24 * time.Hour

	// minSamplesForStats is the minimum number of recent accepted prices
	// needed before we trust median/MAD enough to auto-accept or flag on
	// it. Below this, we lean on submitter trust instead (cold start).
	minSamplesForStats = 5

	// baseOutlierThreshold is the fractional deviation from the local
	// median that's considered "normal" for a trusted submitter.
	baseOutlierThreshold = 0.35 // 35%

	// hardRejectThreshold is deviation so extreme it's rejected outright
	// regardless of submitter trust (almost certainly a typo, e.g. an
	// extra zero, or a bad-faith entry).
	hardRejectThreshold = 0.90 // 90%

	// coldStartAutoAcceptTrust is the trust score above which a submitter's
	// report is auto-accepted even when there isn't enough local price
	// history yet to run outlier detection.
	coldStartAutoAcceptTrust = 0.75

	trustDeltaOnAccept = 0.02
	trustDeltaOnFlag   = -0.05
	trustDeltaOnReject = -0.15
)

type PriceService struct {
	prices     domain.PriceRepository
	submitters domain.SubmitterRepository
	now        func() time.Time // overridable for tests
}

func NewPriceService(prices domain.PriceRepository, submitters domain.SubmitterRepository) *PriceService {
	return &PriceService{
		prices:     prices,
		submitters: submitters,
		now:        time.Now,
	}
}

// SubmitPriceInput is what a farmer/field agent sends in when reporting a
// price they observed at a market.
type SubmitPriceInput struct {
	CommodityID string
	MarketID    string
	SubmitterID string
	Price       float64
	Currency    string
	Unit        string
}

// SubmitResult tells the caller what happened and why, so the API layer can
// return a helpful message ("submitted", "held for review", etc).
type SubmitResult struct {
	Report *domain.PriceReport
	Reason string
}

func (s *PriceService) SubmitPrice(ctx context.Context, in SubmitPriceInput) (*SubmitResult, error) {
	if err := validate(in); err != nil {
		return nil, err
	}

	submitter, err := s.submitters.Get(ctx, in.SubmitterID)
	if err != nil {
		return nil, fmt.Errorf("loading submitter: %w", err)
	}

	report := &domain.PriceReport{
		ID:          uuid.NewString(),
		CommodityID: in.CommodityID,
		MarketID:    in.MarketID,
		SubmitterID: in.SubmitterID,
		Price:       in.Price,
		Currency:    in.Currency,
		Unit:        in.Unit,
		Status:      domain.StatusPending,
		SubmittedAt: s.now(),
	}

	status, reason, err := s.evaluate(ctx, in, submitter)
	if err != nil {
		return nil, err
	}
	report.Status = status
	report.FlagReason = reason

	if err := s.prices.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("saving report: %w", err)
	}

	s.updateTrust(ctx, submitter, status)

	if status == domain.StatusAccepted {
		if err := s.recomputeAggregate(ctx, in.CommodityID, in.MarketID); err != nil {
			// Don't fail the submission over a stale aggregate — log and
			// let the next accepted report (or a scheduled job) fix it.
			reason = reason + " (aggregate refresh deferred)"
		}
	}

	return &SubmitResult{Report: report, Reason: reason}, nil
}

// evaluate decides accept / flag / reject for a submission. It's the one
// place outlier + trust logic lives, so tuning it later doesn't ripple
// through the rest of the service.
func (s *PriceService) evaluate(ctx context.Context, in SubmitPriceInput, submitter *domain.Submitter) (domain.ReportStatus, string, error) {
	since := s.now().Add(-lookbackWindow)
	recent, err := s.prices.RecentAcceptedPrices(ctx, in.CommodityID, in.MarketID, since)
	if err != nil {
		return domain.StatusPending, "", fmt.Errorf("loading recent prices: %w", err)
	}

	// Cold start: not enough local history to judge against. Lean on
	// submitter trust instead of leaving every early report stuck pending.
	if len(recent) < minSamplesForStats {
		if submitter.Verified && submitter.TrustScore >= coldStartAutoAcceptTrust {
			return domain.StatusAccepted, "auto-accepted: trusted submitter, insufficient local history for comparison", nil
		}
		return domain.StatusPending, "held for review: insufficient local price history to auto-validate", nil
	}

	values := make([]float64, len(recent))
	for i, r := range recent {
		values[i] = r.Price
	}
	med := median(values)
	deviation := percentDeviation(in.Price, med)

	// Extreme deviation: reject outright regardless of trust. A trusted
	// submitter is far more likely to have made a typo than to have found
	// a 90%+ price swing no one else is reporting.
	if deviation >= hardRejectThreshold {
		return domain.StatusRejected, fmt.Sprintf("rejected: %.0f%% deviation from local median (₦%.2f)", deviation*100, med), nil
	}

	// Trusted submitters get a wider allowed band before review is
	// triggered, since they've historically been reliable.
	threshold := baseOutlierThreshold * (1 + submitter.TrustScore)
	if deviation >= threshold {
		return domain.StatusFlagged, fmt.Sprintf("flagged for review: %.0f%% deviation from local median (₦%.2f)", deviation*100, med), nil
	}

	return domain.StatusAccepted, "accepted: within normal range of recent local prices", nil
}

func (s *PriceService) updateTrust(ctx context.Context, submitter *domain.Submitter, status domain.ReportStatus) {
	submitter.TotalReports++
	var delta float64
	switch status {
	case domain.StatusAccepted:
		delta = trustDeltaOnAccept
		submitter.AcceptedReports++
	case domain.StatusFlagged:
		delta = trustDeltaOnFlag
	case domain.StatusRejected:
		delta = trustDeltaOnReject
	}
	submitter.TrustScore = clamp(submitter.TrustScore+delta, 0, 1)
	_ = s.submitters.Save(ctx, submitter) // best-effort; a save failure here shouldn't fail the submission
}

func (s *PriceService) recomputeAggregate(ctx context.Context, commodityID, marketID string) error {
	since := s.now().Add(-24 * time.Hour)
	recent, err := s.prices.RecentAcceptedPrices(ctx, commodityID, marketID, since)
	if err != nil || len(recent) == 0 {
		return err
	}

	min, max, sum := recent[0].Price, recent[0].Price, 0.0
	for _, r := range recent {
		if r.Price < min {
			min = r.Price
		}
		if r.Price > max {
			max = r.Price
		}
		sum += r.Price
	}

	agg := &domain.PriceAggregate{
		CommodityID: commodityID,
		MarketID:    marketID,
		Date:        s.now(),
		AvgPrice:    sum / float64(len(recent)),
		MinPrice:    min,
		MaxPrice:    max,
		SampleSize:  len(recent),
	}
	return s.prices.SaveAggregate(ctx, agg)
}

// GetCurrentPrice returns today's aggregate for a commodity/market pair.
func (s *PriceService) GetCurrentPrice(ctx context.Context, commodityID, marketID string) (*domain.PriceAggregate, error) {
	return s.prices.CurrentAggregate(ctx, commodityID, marketID)
}

// VerifySubmitter marks a submitter (typically a field agent or vetted
// farmer) as verified and sets their initial trust score. This is what lets
// them auto-accept during cold start (see coldStartAutoAcceptTrust) instead
// of every one of their early reports sitting pending.
func (s *PriceService) VerifySubmitter(ctx context.Context, submitterID string, trustScore float64) (*domain.Submitter, error) {
	submitter, err := s.submitters.Get(ctx, submitterID)
	if err != nil {
		return nil, fmt.Errorf("loading submitter: %w", err)
	}
	submitter.Verified = true
	submitter.TrustScore = clamp(trustScore, 0, 1)
	if err := s.submitters.Save(ctx, submitter); err != nil {
		return nil, fmt.Errorf("saving submitter: %w", err)
	}
	return submitter, nil
}

// ReviewReport lets an admin manually resolve a pending or flagged report.
// Only Accepted/Rejected are valid outcomes here — a human is making the
// final call, so there's no "flag again" or "re-pend" path.
func (s *PriceService) ReviewReport(ctx context.Context, reportID string, decision domain.ReportStatus, reason string) (*domain.PriceReport, error) {
	if decision != domain.StatusAccepted && decision != domain.StatusRejected {
		return nil, fmt.Errorf("%w: decision must be accepted or rejected", domain.ErrInvalidInput)
	}

	report, err := s.prices.GetReport(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("loading report: %w", err)
	}
	if report.Status != domain.StatusPending && report.Status != domain.StatusFlagged {
		return nil, fmt.Errorf("%w: report is already %s, not pending review", domain.ErrInvalidInput, report.Status)
	}

	if err := s.prices.UpdateReportStatus(ctx, reportID, decision, reason); err != nil {
		return nil, fmt.Errorf("updating report status: %w", err)
	}

	submitter, err := s.submitters.Get(ctx, report.SubmitterID)
	if err == nil {
		s.updateTrust(ctx, submitter, decision)
	}

	if decision == domain.StatusAccepted {
		if err := s.recomputeAggregate(ctx, report.CommodityID, report.MarketID); err != nil {
			// Same non-fatal handling as the submission path — the next
			// accepted report or scheduled job will catch it up.
			_ = err
		}
	}

	updated, err := s.prices.GetReport(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("loading updated report: %w", err)
	}
	return updated, nil
}

func validate(in SubmitPriceInput) error {
	if in.CommodityID == "" || in.MarketID == "" || in.SubmitterID == "" || in.Currency == "" || in.Unit == "" {
		return domain.ErrInvalidInput
	}
	if in.Price <= 0 {
		return domain.ErrInvalidPrice
	}
	return nil
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
