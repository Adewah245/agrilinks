package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type registrationStoreStub struct {
	record    *models.RegistrationPhoneCode
	createErr error
	deleted   bool
}

func (s *registrationStoreStub) Create(_ context.Context, code *models.RegistrationPhoneCode, ttl, _ time.Duration) error {
	if s.createErr != nil {
		return s.createErr
	}
	code.ExpiresAt = time.Now().Add(ttl)
	s.record = code
	return nil
}
func (s *registrationStoreStub) Attempt(_ context.Context, tokenHash, phone string, maxAttempts int) (*models.RegistrationPhoneCode, error) {
	if s.record == nil || s.record.TokenHash != tokenHash || s.record.Phone != phone || s.record.Attempts >= maxAttempts || !time.Now().Before(s.record.ExpiresAt) {
		return nil, repository.ErrRegistrationOTPInvalid
	}
	s.record.Attempts++
	return s.record, nil
}
func (s *registrationStoreStub) Delete(_ context.Context, tokenHash string) error {
	if s.record != nil && s.record.TokenHash == tokenHash {
		s.record = nil
	}
	s.deleted = true
	return nil
}

func TestRegistrationCodeBindsPhoneAndBrowser(t *testing.T) {
	store := &registrationStoreStub{}
	delivery := &fakeOTPDelivery{}
	service := NewRegistrationOTPService(store, delivery, time.Minute, 5, time.Minute)
	service.hashCost = bcrypt.MinCost
	token, err := service.Issue(context.Background(), "08012345678")
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 || store.record.TokenHash == token || store.record.CodeHash == delivery.message.Code {
		t.Fatal("signup secrets must be stored as hashes")
	}
	if delivery.message.Phone != "2348012345678" || delivery.message.Email != "" || delivery.message.UserID != 0 || delivery.message.Purpose != "registration" {
		t.Fatal("signup code must go to the phone before an account exists")
	}
	for _, input := range []struct{ token, phone, code string }{
		{"", "08012345678", delivery.message.Code},
		{strings.Repeat("z", 64), "08012345678", delivery.message.Code},
		{token, "08099999999", delivery.message.Code},
		{token, "08012345678", ""},
	} {
		if _, err := service.Check(context.Background(), input.token, input.phone, input.code); !errors.Is(err, repository.ErrRegistrationOTPInvalid) {
			t.Fatalf("unbound challenge accepted: %v", err)
		}
	}
	proof, err := service.Check(context.Background(), token, "+234 801 234 5678", delivery.message.Code)
	if err != nil || proof.Phone != "2348012345678" {
		t.Fatalf("valid code failed: %v", err)
	}
}

func TestRegistrationCodeDeliveryFailureCleansUp(t *testing.T) {
	store := &registrationStoreStub{}
	delivery := &fakeOTPDelivery{err: errors.New("provider failure")}
	service := NewRegistrationOTPService(store, delivery, time.Minute, 5, time.Minute)
	service.hashCost = bcrypt.MinCost
	if token, err := service.Issue(context.Background(), "08012345678"); err == nil || token != "" {
		t.Fatal("failed delivery returned a usable browser token")
	}
	if !store.deleted || store.record != nil {
		t.Fatal("unsent challenge was retained")
	}
}

func TestRegistrationRequiresPhoneDelivery(t *testing.T) {
	service := NewRegistrationOTPService(&registrationStoreStub{}, nil, time.Minute, 5, time.Minute)
	if _, err := service.Issue(context.Background(), "08012345678"); !errors.Is(err, ErrRegistrationOTPUnavailable) {
		t.Fatalf("unconfigured phone provider accepted: %v", err)
	}
	if _, err := service.Check(context.Background(), strings.Repeat("a", 64), "08012345678", "123456"); !errors.Is(err, ErrRegistrationOTPUnavailable) {
		t.Fatalf("confirmation bypassed provider requirement: %v", err)
	}
}
