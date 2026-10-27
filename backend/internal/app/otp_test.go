package app

import (
	"testing"

	"backend/internal/config"
	"backend/internal/services"
)

func TestOTPDevProviderSelection(t *testing.T) {
	for _, provider := range []string{"otpdev", ""} {
		t.Run(provider, func(t *testing.T) {
			delivery, err := newOTPDelivery(config.Config{
				SMSProvider: provider, OTPDevAPIKey: "test-key", OTPDevSender: "AgroShield",
				OTPDevTemplateID: "6d16aa9d-bf19-4141-8169-48b46d972fc6",
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := delivery.(*services.OTPDevDelivery); !ok {
				t.Fatalf("selected delivery = %T, want OTP.dev", delivery)
			}
		})
	}
}

func TestOTPProviderRejectsMissingConfiguration(t *testing.T) {
	if _, err := newOTPDelivery(config.Config{SMSProvider: "otpdev"}); err == nil {
		t.Fatal("OTP.dev selected without credentials")
	}
	if _, err := newOTPDelivery(config.Config{}); err == nil {
		t.Fatal("default OTP.dev provider selected without credentials")
	}
	if _, err := newOTPDelivery(config.Config{SMSProvider: "unknown"}); err == nil {
		t.Fatal("unknown provider silently disabled OTP delivery")
	}
	delivery, err := newOTPDelivery(config.Config{SMSProvider: "disabled"})
	if err != nil || delivery != nil {
		t.Fatalf("disabled delivery = %v, error = %v", delivery, err)
	}
}
