package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"errors"
)

// DeliveryService is the buyer-facing, authorization-safe delivery boundary.
type DeliveryService struct {
	repo *repository.DeliveryRepository
}

func NewDeliveryService(repo *repository.DeliveryRepository) *DeliveryService {
	return &DeliveryService{repo: repo}
}
func (s *DeliveryService) MyDeliveries(buyerID int) ([]models.Delivery, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	return s.repo.ListByBuyer(buyerID)
}
func (s *DeliveryService) MyTracking(buyerID, deliveryID int) ([]models.DeliveryTracking, error) {
	if buyerID <= 0 || deliveryID <= 0 {
		return nil, errors.New("invalid delivery request")
	}
	return s.repo.TrackingForBuyer(buyerID, deliveryID)
}
