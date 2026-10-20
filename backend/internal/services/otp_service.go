package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultOTPTTL         = 10 * time.Minute
	DefaultOTPMaxAttempts = 5
	otpCodeDigits         = 6
)

// OTPMessage is the provider-neutral payload handed to a delivery channel.
type OTPMessage struct {
	UserID    int
	Phone     string
	Email     string
	Purpose   string
	Code      string
	ExpiresAt time.Time
}

// OTPDelivery sends an OTP through a concrete provider such as Termii or
// Brevo. Providers can be replaced without changing OTP generation or storage.
type OTPDelivery interface {
	SendOTP(ctx context.Context, message OTPMessage) error
}

type otpStore interface {
	Create(*models.OTPCode) error
	LatestActive(userID int, purpose string) (*models.OTPCode, error)
	IncrementAttempts(id int64) error
	Consume(id int64) error
	Delete(id int64) error
}

// OTPService generates, stores, delivers, and verifies one-time passcodes.
//
// Plaintext codes exist only long enough to be sent through the selected
// delivery channel. Only bcrypt hashes are persisted.
type OTPService struct {
	repo        otpStore
	delivery    OTPDelivery
	ttl         time.Duration
	maxAttempts int
	hashCost    int
	now         func() time.Time
}

func NewOTPService(repo *repository.OTPRepository, delivery OTPDelivery) *OTPService {
	return &OTPService{
		repo:        repo,
		delivery:    delivery,
		ttl:         DefaultOTPTTL,
		maxAttempts: DefaultOTPMaxAttempts,
		hashCost:    bcrypt.DefaultCost,
		now:         time.Now,
	}
}

// Issue creates and delivers a new OTP for a user and purpose.
func (s *OTPService) Issue(
	ctx context.Context,
	userID int,
	phone string,
	email string,
	purpose string,
) error {
	if s == nil || s.repo == nil {
		return errors.New("OTP service is not initialized")
	}
	if s.delivery == nil {
		return errors.New("OTP delivery channel is not configured")
	}
	if userID <= 0 {
		return errors.New("invalid user")
	}

	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(email)
	purpose = strings.TrimSpace(purpose)

	if phone == "" && email == "" {
		return errors.New("OTP recipient is required")
	}
	if purpose == "" {
		return errors.New("OTP purpose is required")
	}

	code, err := generateOTPCode(otpCodeDigits)
	if err != nil {
		return fmt.Errorf("generate OTP: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), s.hashCost)
	if err != nil {
		return fmt.Errorf("hash OTP: %w", err)
	}

	expiresAt := s.now().Add(s.ttl)
	otp := &models.OTPCode{
		UserID:    userID,
		Phone:     phone,
		Purpose:   purpose,
		CodeHash:  string(hash),
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Create(otp); err != nil {
		return err
	}

	err = s.delivery.SendOTP(ctx, OTPMessage{
		UserID:    userID,
		Phone:     phone,
		Email:     email,
		Purpose:   purpose,
		Code:      code,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		_ = s.repo.Delete(otp.ID)
		return fmt.Errorf("deliver OTP: %w", err)
	}

	return nil
}

// Verify checks the newest active OTP and consumes it after a successful match.
func (s *OTPService) Verify(
	userID int,
	purpose string,
	code string,
) error {
	if s == nil || s.repo == nil {
		return errors.New("OTP service is not initialized")
	}
	if userID <= 0 {
		return errors.New("invalid user")
	}

	purpose = strings.TrimSpace(purpose)
	code = strings.TrimSpace(code)

	if purpose == "" {
		return errors.New("OTP purpose is required")
	}
	if code == "" {
		return errors.New("OTP code is required")
	}

	otp, err := s.repo.LatestActive(userID, purpose)
	if err != nil {
		return err
	}
	if otp == nil {
		return errors.New("OTP code not found or already used")
	}
	if !s.now().Before(otp.ExpiresAt) {
		return errors.New("OTP code has expired")
	}
	if otp.Attempts >= s.maxAttempts {
		return errors.New("too many OTP attempts; request a new code")
	}

	if bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)) != nil {
		if err := s.repo.IncrementAttempts(otp.ID); err != nil {
			return err
		}

		return errors.New("invalid OTP code")
	}

	return s.repo.Consume(otp.ID)
}

func generateOTPCode(digits int) (string, error) {
	if digits <= 0 {
		return "", errors.New("OTP length must be greater than zero")
	}

	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", digits, value.Int64()), nil
}
