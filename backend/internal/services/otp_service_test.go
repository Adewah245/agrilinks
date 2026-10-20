package services

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type fakeOTPStore struct {
	created       *models.OTPCode
	active        *models.OTPCode
	attempts      int
	consumed      int
	deleted       int
	createError   error
	deliveryError error
}

func (s *fakeOTPStore) Create(otp *models.OTPCode) error {
	if s.createError != nil {
		return s.createError
	}

	otp.ID = 1
	otp.CreatedAt = time.Now()
	s.created = otp
	s.active = otp
	return nil
}

func (s *fakeOTPStore) LatestActive(userID int, purpose string) (*models.OTPCode, error) {
	return s.active, nil
}

func (s *fakeOTPStore) IncrementAttempts(id int64) error {
	s.attempts++
	return nil
}

func (s *fakeOTPStore) Consume(id int64) error {
	s.consumed++
	return nil
}

func (s *fakeOTPStore) Delete(id int64) error {
	s.deleted++
	return nil
}

type fakeOTPDelivery struct {
	message OTPMessage
	err     error
}

func (d *fakeOTPDelivery) SendOTP(ctx context.Context, message OTPMessage) error {
	d.message = message
	return d.err
}

func newTestOTPService(store *fakeOTPStore, delivery *fakeOTPDelivery) *OTPService {
	return &OTPService{
		repo:        store,
		delivery:    delivery,
		ttl:         10 * time.Minute,
		maxAttempts: DefaultOTPMaxAttempts,
		hashCost:    bcrypt.MinCost,
		now:         func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
}

func TestOTPServiceIssueAndVerify(t *testing.T) {
	store := &fakeOTPStore{}
	delivery := &fakeOTPDelivery{}
	service := newTestOTPService(store, delivery)

	if err := service.Issue(
		context.Background(),
		7,
		"08012345678",
		"buyer@example.com",
		"transfer",
	); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if store.created == nil {
		t.Fatal("expected OTP to be stored")
	}
	if store.created.UserID != 7 || store.created.Purpose != "transfer" {
		t.Fatalf("unexpected stored OTP: %#v", store.created)
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(delivery.message.Code) {
		t.Fatalf("delivered code = %q, want six digits", delivery.message.Code)
	}
	if store.created.CodeHash == delivery.message.Code {
		t.Fatal("plaintext code must not be stored")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(store.created.CodeHash),
		[]byte(delivery.message.Code),
	); err != nil {
		t.Fatalf("stored hash does not match delivered code: %v", err)
	}

	if err := service.Verify(7, "transfer", delivery.message.Code); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if store.consumed != 1 {
		t.Fatalf("consumed = %d, want 1", store.consumed)
	}
}

func TestOTPServiceRejectsWrongCodeAndCountsAttempt(t *testing.T) {
	store := &fakeOTPStore{}
	delivery := &fakeOTPDelivery{}
	service := newTestOTPService(store, delivery)

	if err := service.Issue(
		context.Background(),
		7,
		"08012345678",
		"",
		"withdrawal",
	); err != nil {
		t.Fatal(err)
	}

	if err := service.Verify(7, "withdrawal", "000000"); err == nil {
		t.Fatal("expected invalid OTP error")
	}
	if store.attempts != 1 {
		t.Fatalf("attempts = %d, want 1", store.attempts)
	}
	if store.consumed != 0 {
		t.Fatal("invalid OTP must not be consumed")
	}
}

func TestOTPServiceRejectsExpiredCode(t *testing.T) {
	store := &fakeOTPStore{}
	delivery := &fakeOTPDelivery{}
	service := newTestOTPService(store, delivery)

	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store.active = &models.OTPCode{
		ID:        1,
		UserID:    7,
		Purpose:   "password_reset",
		CodeHash:  string(hash),
		ExpiresAt: service.now().Add(-time.Second),
	}

	if err := service.Verify(7, "password_reset", "123456"); err == nil {
		t.Fatal("expected expired OTP error")
	}
	if store.consumed != 0 {
		t.Fatal("expired OTP must not be consumed")
	}
}

func TestOTPServiceDeletesCodeWhenDeliveryFails(t *testing.T) {
	store := &fakeOTPStore{}
	delivery := &fakeOTPDelivery{err: errors.New("provider unavailable")}
	service := newTestOTPService(store, delivery)

	err := service.Issue(context.Background(), 7, "08012345678", "", "transfer")
	if err == nil {
		t.Fatal("expected delivery error")
	}
	if store.deleted != 1 {
		t.Fatalf("deleted = %d, want 1", store.deleted)
	}
}
