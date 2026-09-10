package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"backend/internal/models"
	"backend/internal/repository"
)

// PaymentService isolates B-Moni-facing workflow from HTTP handlers. Provider
// confirmation remains intentionally unavailable until signed B-Moni webhooks
// and credentials are configured.
type PaymentService struct{ repo *repository.PaymentRepository }

func NewPaymentService(repo *repository.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) StartBMoniPayment(buyerID, orderID int) (*models.Transaction, error) {
	if buyerID <= 0 || orderID <= 0 {
		return nil, errors.New("invalid payment request")
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("create payment reference: %w", err)
	}
	return s.repo.CreatePendingForOrder(buyerID, orderID, "AGS-"+hex.EncodeToString(b))
}
func (s *PaymentService) MyPayments(buyerID int) ([]models.Transaction, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	return s.repo.ListByBuyer(buyerID)
}
func (s *PaymentService) MyWallet(ownerID int) (*models.Wallet, error) {
	if ownerID <= 0 {
		return nil, errors.New("invalid wallet owner")
	}
	return s.repo.GetWallet("user", ownerID)
}
