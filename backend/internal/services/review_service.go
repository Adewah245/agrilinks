package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"errors"
	"strings"
)

type ReviewService struct{ repo *repository.ReviewRepository }

func NewReviewService(r *repository.ReviewRepository) *ReviewService { return &ReviewService{r} }
func (s *ReviewService) ReviewProduct(userID, productID, orderID, rating int, comment string) (*models.ProductReview, error) {
	if userID <= 0 || productID <= 0 || orderID <= 0 || rating < 1 || rating > 5 {
		return nil, errors.New("invalid review")
	}
	v := &models.ProductReview{ProductID: productID, ReviewerID: userID, OrderID: orderID, Rating: rating, Comment: strings.TrimSpace(comment)}
	if e := s.repo.CreateProductReview(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *ReviewService) ProductReviews(id int) ([]models.ProductReview, error) {
	if id <= 0 {
		return nil, errors.New("invalid product")
	}
	return s.repo.ListProductReviews(id)
}
func (s *ReviewService) MyVerification(id int) (*models.Verification, error) {
	if id <= 0 {
		return nil, errors.New("invalid user")
	}
	return s.repo.MyVerification(id)
}
