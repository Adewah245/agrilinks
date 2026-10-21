package services

import (
	"errors"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

type fakeReviewStore struct {
	createErr    error
	productAvg   float64
	productCount int
	sellerAvg    float64
	sellerCount  int
	badges       []models.VerificationBadge
	issuedBadge  *models.VerificationBadge
	revokedID    int
}

func (s *fakeReviewStore) CreateProductReview(review *models.ProductReview) error {
	if s.createErr != nil {
		return s.createErr
	}
	review.ID = 3
	return nil
}

func (s *fakeReviewStore) ListProductReviews(int) ([]models.ProductReview, error) {
	return nil, nil
}

func (s *fakeReviewStore) ProductRating(int) (float64, int, error) {
	return s.productAvg, s.productCount, nil
}

func (s *fakeReviewStore) SellerRating(string, int) (float64, int, error) {
	return s.sellerAvg, s.sellerCount, nil
}

func (s *fakeReviewStore) ActiveBadges(string, int) ([]models.VerificationBadge, error) {
	return s.badges, nil
}

func (s *fakeReviewStore) IssueBadge(badge *models.VerificationBadge, actorID int) error {
	badge.ID = 8
	badge.IssuedAt = time.Now()
	badge.IssuedBy = &actorID
	s.issuedBadge = badge
	return nil
}

func (s *fakeReviewStore) RevokeBadge(badgeID int) error {
	s.revokedID = badgeID
	return nil
}

func (s *fakeReviewStore) MyVerification(int) (*models.Verification, error) {
	return nil, nil
}

func TestReviewServiceSurfacesDeliveryRequirement(t *testing.T) {
	store := &fakeReviewStore{createErr: repository.ErrReviewRequiresDeliveredOrder}
	service := NewReviewService(store)

	_, err := service.ReviewProduct(7, 3, 11, 5, "Good produce")
	if !errors.Is(err, ErrReviewRequiresDeliveredOrder) {
		t.Fatalf("error = %v, want %v", err, ErrReviewRequiresDeliveredOrder)
	}
}

func TestSellerTrustCombinesRatingAndBadges(t *testing.T) {
	store := &fakeReviewStore{
		sellerAvg:   4.25,
		sellerCount: 8,
		badges:      []models.VerificationBadge{{BadgeType: "verified_farmer"}},
	}
	service := NewReviewService(store)

	trust, err := service.SellerTrust("user", 12)
	if err != nil {
		t.Fatal(err)
	}
	if trust.AverageRating != 4.25 || trust.ReviewCount != 8 || len(trust.Badges) != 1 {
		t.Fatalf("unexpected trust summary: %#v", trust)
	}
}

func TestOnlyAdministratorCanManageBadges(t *testing.T) {
	store := &fakeReviewStore{}
	service := NewReviewService(store)

	if _, err := service.IssueBadge(&models.User{ID: 4, Role: "farmer"}, "user", 7, "trusted_seller", "", nil); !errors.Is(err, ErrTrustForbidden) {
		t.Fatalf("non-admin issue error = %v", err)
	}
	badge, err := service.IssueBadge(&models.User{ID: 1, IsAdmin: true}, "user", 7, "trusted_seller", "Reliable seller", nil)
	if err != nil {
		t.Fatal(err)
	}
	if badge.ID != 8 || store.issuedBadge == nil {
		t.Fatal("admin badge was not stored")
	}

	if err := service.RevokeBadge(&models.User{ID: 4, Role: "farmer"}, 8); !errors.Is(err, ErrTrustForbidden) {
		t.Fatalf("non-admin revoke error = %v", err)
	}
	if err := service.RevokeBadge(&models.User{ID: 1, IsAdmin: true}, 8); err != nil {
		t.Fatal(err)
	}
	if store.revokedID != 8 {
		t.Fatalf("revoked ID = %d, want 8", store.revokedID)
	}
}
