package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// PaymentService isolates Flutterwave-facing workflow from HTTP handlers.
// Provider confirmation remains intentionally unavailable until signed
// Flutterwave webhooks
// and credentials are configured.
type PaymentService struct{ repo *repository.PaymentRepository }

func NewPaymentService(repo *repository.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) StartFlutterwavePayment(buyerID, orderID int) (*models.Payment, error) {
	if buyerID <= 0 || orderID <= 0 {
		return nil, errors.New("invalid payment request")
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("create payment reference: %w", err)
	}
	return s.repo.CreatePendingForOrder(buyerID, orderID, "AGS-"+hex.EncodeToString(b))
}
func (s *PaymentService) MyPayments(buyerID int) ([]models.Payment, error) {
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

// PaymentProfile loads the private bank-account and wallet-number details
// used by the transfer flow.
func (s *PaymentService) PaymentProfile(userID int) (*models.User, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	return s.repo.GetPaymentProfile(userID)
}

// UpdateBankAccount validates and saves a user's payout account.
func (s *PaymentService) UpdateBankAccount(
	userID int,
	bankName string,
	bankCode string,
	accountName string,
	accountNumber string,
) error {
	if userID <= 0 {
		return errors.New("invalid user")
	}

	bankName = strings.TrimSpace(bankName)
	bankCode = strings.TrimSpace(bankCode)
	accountName = strings.TrimSpace(accountName)
	accountNumber = strings.TrimSpace(accountNumber)

	if bankName == "" || bankCode == "" || accountName == "" || accountNumber == "" {
		return errors.New("bank name, bank code, account name, and account number are required")
	}

	return s.repo.UpsertBankAccount(&models.UserBankAccount{
		UserID:        userID,
		BankName:      bankName,
		BankCode:      bankCode,
		AccountName:   accountName,
		AccountNumber: accountNumber,
	})
}
