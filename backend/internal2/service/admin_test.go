package service

import (
	"context"
	"testing"

	"backend/internal2/domain"
)

func TestVerifySubmitter_SetsVerifiedAndTrust(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	submitter, err := svc.VerifySubmitter(ctx, "agent-1", 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !submitter.Verified {
		t.Error("expected submitter to be marked verified")
	}
	if submitter.TrustScore != 0.9 {
		t.Errorf("expected trust score 0.9, got %.2f", submitter.TrustScore)
	}
}

func TestVerifySubmitter_ThenColdStartAutoAccepts(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	if _, err := svc.VerifySubmitter(ctx, "agent-1", 0.9); err != nil {
		t.Fatalf("unexpected error verifying: %v", err)
	}

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "agent-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}
	if res.Report.Status != domain.StatusAccepted {
		t.Errorf("expected verified trusted agent to auto-accept on cold start, got %s", res.Report.Status)
	}
}

func TestReviewReport_ApprovePendingReport(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}
	if res.Report.Status != domain.StatusPending {
		t.Fatalf("expected pending on cold start, got %s", res.Report.Status)
	}

	updated, err := svc.ReviewReport(ctx, res.Report.ID, domain.StatusAccepted, "confirmed manually")
	if err != nil {
		t.Fatalf("unexpected error reviewing: %v", err)
	}
	if updated.Status != domain.StatusAccepted {
		t.Errorf("expected report to be accepted after review, got %s", updated.Status)
	}

	agg, err := svc.GetCurrentPrice(ctx, "maize", "makurdi-central")
	if err != nil {
		t.Fatalf("expected an aggregate to exist after admin approval: %v", err)
	}
	if agg.SampleSize != 1 {
		t.Errorf("expected aggregate sample size 1, got %d", agg.SampleSize)
	}
}

func TestReviewReport_RejectAlreadyReviewedFails(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	res, err := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}

	if _, err := svc.ReviewReport(ctx, res.Report.ID, domain.StatusAccepted, "ok"); err != nil {
		t.Fatalf("unexpected error on first review: %v", err)
	}

	if _, err := svc.ReviewReport(ctx, res.Report.ID, domain.StatusRejected, "changed my mind"); err == nil {
		t.Error("expected error reviewing an already-resolved report")
	}
}

func TestReviewReport_InvalidDecisionRejected(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	res, _ := svc.SubmitPrice(ctx, SubmitPriceInput{
		CommodityID: "maize", MarketID: "makurdi-central", SubmitterID: "farmer-1",
		Price: 18000, Currency: "NGN", Unit: "bag_100kg",
	})

	if _, err := svc.ReviewReport(ctx, res.Report.ID, domain.StatusFlagged, "not a valid final decision"); err == nil {
		t.Error("expected error for a non-terminal decision value")
	}
}
