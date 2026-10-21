package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"errors"
	"strings"
	"time"
)

var (
	ErrReviewRequiresDeliveredOrder = repository.ErrReviewRequiresDeliveredOrder
	ErrReviewAlreadyExists          = repository.ErrReviewAlreadyExists
	ErrInvalidBadgeType             = repository.ErrInvalidBadgeType
	ErrBadgeOwnerNotFound           = repository.ErrBadgeOwnerNotFound
	ErrTrustForbidden               = errors.New("only administrators can manage trust badges")
)

type reviewStore interface {
	CreateProductReview(*models.ProductReview) error
	ListProductReviews(int) ([]models.ProductReview, error)
	ProductRating(int) (float64, int, error)
	SellerRating(string, int) (float64, int, error)
	ActiveBadges(string, int) ([]models.VerificationBadge, error)
	IssueBadge(*models.VerificationBadge, int) error
	RevokeBadge(int) error
	MyVerification(int) (*models.Verification, error)
}

type ReviewService struct{ repo reviewStore }

func NewReviewService(repo reviewStore) *ReviewService { return &ReviewService{repo: repo} }

// ReviewProduct records buyer feedback for a delivered purchase.
func (s *ReviewService) ReviewProduct(
	userID,
	productID,
	orderID,
	rating int,
	comment string,
) (*models.ProductReview, error) {
	if userID <= 0 || productID <= 0 || orderID <= 0 || rating < 1 || rating > 5 {
		return nil, errors.New("invalid review")
	}

	review := &models.ProductReview{
		ProductID:  productID,
		ReviewerID: userID,
		OrderID:    orderID,
		Rating:     rating,
		Comment:    strings.TrimSpace(comment),
	}
	if err := s.repo.CreateProductReview(review); err != nil {
		return nil, err
	}
	return review, nil
}

func (s *ReviewService) ProductReviews(
	productID int,
) ([]models.ProductReview, error) {
	if productID <= 0 {
		return nil, errors.New("invalid product")
	}
	return s.repo.ListProductReviews(productID)
}

// ProductTrust returns the public rating summary for one product.
func (s *ReviewService) ProductTrust(productID int) (*models.TrustSummary, error) {
	if productID <= 0 {
		return nil, errors.New("invalid product")
	}
	average, count, err := s.repo.ProductRating(productID)
	if err != nil {
		return nil, err
	}
	return &models.TrustSummary{AverageRating: average, ReviewCount: count}, nil
}

// SellerTrust combines product-review reputation with active verification
// badges. It never includes NIN, references, or other private verification data.
func (s *ReviewService) SellerTrust(
	ownerType string,
	ownerID int,
) (*models.TrustSummary, error) {
	if ownerID <= 0 {
		return nil, errors.New("invalid seller")
	}

	average, count, err := s.repo.SellerRating(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	badges, err := s.repo.ActiveBadges(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	return &models.TrustSummary{
		AverageRating: average,
		ReviewCount:   count,
		Badges:        badges,
	}, nil
}

// ActiveBadges returns public badges for a user or organization.
func (s *ReviewService) ActiveBadges(
	ownerType string,
	ownerID int,
) ([]models.VerificationBadge, error) {
	if ownerID <= 0 {
		return nil, errors.New("invalid badge owner")
	}
	return s.repo.ActiveBadges(ownerType, ownerID)
}

// IssueBadge lets an administrator grant a manual trust badge.
func (s *ReviewService) IssueBadge(
	actor *models.User,
	ownerType string,
	ownerID int,
	badgeType,
	description string,
	expiresAt *time.Time,
) (*models.VerificationBadge, error) {
	if actor == nil || !actor.IsAdministrator() {
		return nil, ErrTrustForbidden
	}
	if ownerID <= 0 {
		return nil, errors.New("invalid badge owner")
	}

	badge := &models.VerificationBadge{
		OwnerType:   strings.TrimSpace(ownerType),
		OwnerID:     ownerID,
		BadgeType:   strings.TrimSpace(badgeType),
		Description: strings.TrimSpace(description),
		ExpiresAt:   expiresAt,
		Active:      true,
	}
	if err := s.repo.IssueBadge(badge, actor.ID); err != nil {
		return nil, err
	}
	return badge, nil
}

// RevokeBadge lets an administrator remove a badge from public display.
func (s *ReviewService) RevokeBadge(actor *models.User, badgeID int) error {
	if actor == nil || !actor.IsAdministrator() {
		return ErrTrustForbidden
	}
	if badgeID <= 0 {
		return errors.New("invalid badge")
	}
	return s.repo.RevokeBadge(badgeID)
}

func (s *ReviewService) MyVerification(userID int) (*models.Verification, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}
	return s.repo.MyVerification(userID)
}
