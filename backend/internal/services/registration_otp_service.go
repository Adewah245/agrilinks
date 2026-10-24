package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrRegistrationOTPUnavailable = errors.New("phone confirmation is unavailable right now; please try registering again later")

type registrationOTPStore interface {
	Create(context.Context, *models.RegistrationPhoneCode, time.Duration, time.Duration) error
	Attempt(context.Context, string, string, int) (*models.RegistrationPhoneCode, error)
	Delete(context.Context, string) error
}

type RegistrationOTPService struct {
	repo          registrationOTPStore
	delivery      OTPDelivery
	ttl, cooldown time.Duration
	maxAttempts   int
	hashCost      int
}

// delivery must be a phone channel. App wiring excludes email-only providers.
func NewRegistrationOTPService(repo registrationOTPStore, delivery OTPDelivery, ttl time.Duration, maxAttempts int, cooldown time.Duration) *RegistrationOTPService {
	return &RegistrationOTPService{repo: repo, delivery: delivery, ttl: ttl, maxAttempts: maxAttempts, cooldown: cooldown, hashCost: bcrypt.DefaultCost}
}

func (s *RegistrationOTPService) Available() bool {
	return s != nil && s.repo != nil && s.delivery != nil
}

func (s *RegistrationOTPService) Issue(ctx context.Context, phone string) (string, error) {
	if !s.Available() {
		return "", ErrRegistrationOTPUnavailable
	}
	phone = NormalizeNigerianPhone(phone)
	if phone == "" {
		return "", errors.New("enter a valid Nigerian phone number")
	}
	var tokenBytes [32]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes[:])
	code, err := generateOTPCode(otpCodeDigits)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), s.hashCost)
	if err != nil {
		return "", err
	}
	record := &models.RegistrationPhoneCode{TokenHash: sessionHash(token), Phone: phone, CodeHash: string(hash)}
	if err := s.repo.Create(ctx, record, s.ttl, s.cooldown); err != nil {
		return "", err
	}
	if err := s.delivery.SendOTP(ctx, OTPMessage{Phone: phone, Code: code, Purpose: "registration", ExpiresAt: record.ExpiresAt}); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.Delete(cleanup, record.TokenHash)
		return "", errors.New("we could not send your confirmation code; please try again")
	}
	return token, nil
}

func (s *RegistrationOTPService) Check(ctx context.Context, token, phone, code string) (*models.RegistrationPhoneCode, error) {
	if !s.Available() {
		return nil, ErrRegistrationOTPUnavailable
	}
	phone = NormalizeNigerianPhone(phone)
	code = strings.TrimSpace(code)
	if len(token) != 64 || phone == "" || len(code) != otpCodeDigits {
		return nil, repository.ErrRegistrationOTPInvalid
	}
	record, err := s.repo.Attempt(ctx, sessionHash(token), phone, s.maxAttempts)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(record.CodeHash), []byte(code)) != nil {
		return nil, repository.ErrRegistrationOTPInvalid
	}
	return record, nil
}
