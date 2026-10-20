package services

import (
	"strings"
	"testing"
)

func TestPaymentProfileRejectsInvalidUser(t *testing.T) {
	service := NewPaymentService(nil)

	profile, err := service.PaymentProfile(0)
	if err == nil || !strings.Contains(err.Error(), "invalid user") {
		t.Fatalf("expected invalid user error, got %v", err)
	}
	if profile != nil {
		t.Fatalf("expected no profile, got %#v", profile)
	}
}

func TestUpdateBankAccountValidation(t *testing.T) {
	service := NewPaymentService(nil)

	tests := []struct {
		name          string
		userID        int
		bankName      string
		bankCode      string
		accountName   string
		accountNumber string
		want          string
	}{
		{
			name:          "invalid user",
			userID:        0,
			bankName:      "Example Bank",
			bankCode:      "001",
			accountName:   "James Adah",
			accountNumber: "0123456789",
			want:          "invalid user",
		},
		{
			name:          "missing bank name",
			userID:        7,
			bankName:      "   ",
			bankCode:      "001",
			accountName:   "James Adah",
			accountNumber: "0123456789",
			want:          "required",
		},
		{
			name:          "missing bank code",
			userID:        7,
			bankName:      "Example Bank",
			bankCode:      "",
			accountName:   "James Adah",
			accountNumber: "0123456789",
			want:          "required",
		},
		{
			name:          "missing account name",
			userID:        7,
			bankName:      "Example Bank",
			bankCode:      "001",
			accountName:   " ",
			accountNumber: "0123456789",
			want:          "required",
		},
		{
			name:          "missing account number",
			userID:        7,
			bankName:      "Example Bank",
			bankCode:      "001",
			accountName:   "James Adah",
			accountNumber: "",
			want:          "required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.UpdateBankAccount(
				test.userID,
				test.bankName,
				test.bankCode,
				test.accountName,
				test.accountNumber,
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}
