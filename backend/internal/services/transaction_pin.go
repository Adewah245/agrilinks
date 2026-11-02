package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"

	"backend/internal/models"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var transactionPINDigits = regexp.MustCompile(`^[0-9]{6}$`)

func hashTransactionPIN(pin, confirmation string) (string, error) {
	if !transactionPINDigits.MatchString(pin) {
		return "", errors.New("create a transaction PIN using exactly six digits")
	}
	if subtle.ConstantTimeCompare([]byte(pin), []byte(confirmation)) != 1 {
		return "", errors.New("the transaction PIN and confirmation do not match")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	return string(hash), err
}

type TransactionPINService struct{ users *repository.UserRepository }

func NewTransactionPINService(users *repository.UserRepository) *TransactionPINService {
	return &TransactionPINService{users: users}
}

func (s *TransactionPINService) HasPIN(ctx context.Context, userID int) (bool, error) {
	return s.users.HasTransactionPIN(ctx, userID)
}

func (s *TransactionPINService) Verify(ctx context.Context, userID int, pin string) error {
	if userID <= 0 {
		return errors.New("please sign in")
	}
	if !transactionPINDigits.MatchString(pin) {
		return repository.ErrTransactionPINInvalid
	}
	return s.users.VerifyTransactionPIN(ctx, userID, pin)
}

// Existing accounts require their current password to enrol; an authenticated
// browser session alone must not be enough to choose a member's first PIN.
func (s *TransactionPINService) Setup(ctx context.Context, user *models.User, password, pin, confirmation string) error {
	if user == nil || user.ID <= 0 {
		return errors.New("please sign in")
	}
	current, err := s.users.GetByID(user.ID)
	if err != nil {
		return err
	}
	if current == nil || bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(password)) != nil {
		return errors.New("enter your current account password to create a transaction PIN")
	}
	hash, err := hashTransactionPIN(pin, confirmation)
	if err != nil {
		return err
	}
	return s.users.CreateTransactionPIN(ctx, user.ID, hash)
}
