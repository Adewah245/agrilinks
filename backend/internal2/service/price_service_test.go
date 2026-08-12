package service

import (
	"context"
	"testing"

	"backend/internal2/domain"
	"backend/internal2/repository/memory"
)

func newTestService() *PriceService {
	return NewPriceService(memory.NewPriceRepo(), memory.NewSubmitterRepo())
}

func TestSubmitPrice_ColdStart_UnverifiedGoesPending(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Status != domain.StatusPending {
		t.Errorf("expected pending on cold start for unverified submitter, got %s", res.Report.Status)
	}
}

func TestSubmitPrice_ColdStart_TrustedVerifiedAutoAccepts(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	trusted := &domain.Submitter{ID: "agent-1", Role: domain.RoleFieldAgent, Verified: true, TrustScore: 0.9}
	_ = svc.submitters.Save(ctx, trusted)

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "agent-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Status != domain.StatusAccepted {
		t.Errorf("expected trusted verified submitter to auto-accept on cold start, got %s", res.Report.Status)
	}
}

func TestSubmitPrice_WithinRangeAccepted(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	seedAcceptedHistory(t, svc, "maize", "makurdi-central", []float64{18000, 18200, 17900, 18100, 18050})

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-2",
		Price: 18300, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Status != domain.StatusAccepted {
		t.Errorf("expected in-range price to be accepted, got %s: %s", res.Report.Status, res.Reason)
	}
}

func TestSubmitPrice_ModerateOutlierFlagged(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	seedAcceptedHistory(t, svc, "maize", "makurdi-central", []float64{18000, 18200, 17900, 18100, 18050})

	// ~60% above the local median (~18050) -> exceeds the ~52.5% threshold
	// a neutral-trust (0.5) submitter gets, but stays under the 90%
	// hard-reject threshold.
	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-3",
		Price: 29000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Status != domain.StatusFlagged {
		t.Errorf("expected moderate outlier to be flagged, got %s: %s", res.Report.Status, res.Reason)
	}
}

func TestSubmitPrice_ExtremeOutlierRejected(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	seedAcceptedHistory(t, svc, "maize", "makurdi-central", []float64{18000, 18200, 17900, 18100, 18050})

	// 10x the local median -> hard reject regardless of trust.
	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-4",
		Price: 180000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Report.Status != domain.StatusRejected {
		t.Errorf("expected extreme outlier to be rejected, got %s: %s", res.Report.Status, res.Reason)
	}
}

func TestSubmitPrice_InvalidInputRejectedEarly(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err == nil {
		t.Fatal("expected validation error for missing commodity_id")
	}

	_, err = svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: -5, Currency: "NGN", Unit: "bag_100kg",
	})
	if err == nil {
		t.Fatal("expected validation error for non-positive price")
	}
}

func TestTrustScore_UpdatesOnOutcome(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	seedAcceptedHistory(t, svc, "maize", "makurdi-central", []float64{18000, 18200, 17900, 18100, 18050})

	before, _ := svc.submitters.Get(ctx, "farmer-5")
	startTrust := before.TrustScore

	_, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-5",
		Price: 18100, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, _ := svc.submitters.Get(ctx, "farmer-5")
	if after.TrustScore <= startTrust {
		t.Errorf("expected trust score to increase after an accepted report, got %.3f -> %.3f", startTrust, after.TrustScore)
	}
	if after.TotalReports != 1 || after.AcceptedReports != 1 {
		t.Errorf("expected report counters to update, got total=%d accepted=%d", after.TotalReports, after.AcceptedReports)
	}
}

// seedAcceptedHistory directly submits and force-accepts a set of prices
// from a trusted seed submitter so later test submissions have local
// history to be judged against.
func seedAcceptedHistory(t *testing.T, svc *PriceService, commodityID, marketID string, prices []float64) {
	t.Helper()
	ctx := context.Background()
	seed := &domain.Submitter{ID: "seed-agent", Role: domain.RoleFieldAgent, Verified: true, TrustScore: 0.9}
	_ = svc.submitters.Save(ctx, seed)

	// First 5 submissions from a trusted seed will auto-accept via the
	// cold-start path, seeding local history for subsequent tests.
	for _, p := range prices {
		if _, err := svc.SubmitPrice(ctx, SubmitPriceInput{
			CommodityID: commodityID, MarketID: marketID, SubmitterID: "seed-agent",
			Price: p, Currency: "NGN", Unit: "bag_100kg",
		}); err != nil {
			t.Fatalf("seeding history failed: %v", err)
		}
	}
}
