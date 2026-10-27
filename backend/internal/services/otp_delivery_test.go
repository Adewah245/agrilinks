package services

import (
	"context"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

func TestBrevoOTPDeliverySendsExpectedMessage(t *testing.T) {
	delivery, err := NewBrevoOTPDelivery(
		"smtp-relay.brevo.com",
		"login",
		"key",
		"noreply@example.com",
		"Agro-Shield",
	)
	if err != nil {
		t.Fatal(err)
	}

	var sentAddress string
	var sentFrom string
	var sentTo []string
	var sentMessage []byte
	delivery.sendMail = func(
		addr string,
		auth smtp.Auth,
		from string,
		to []string,
		message []byte,
	) error {
		sentAddress = addr
		sentFrom = from
		sentTo = to
		sentMessage = message
		return nil
	}

	err = delivery.SendOTP(context.Background(), OTPMessage{
		Email:     "buyer@example.com",
		Purpose:   "password_reset",
		Code:      "654321",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("SendOTP() error = %v", err)
	}

	if sentAddress != "smtp-relay.brevo.com:587" {
		t.Fatalf("SMTP address = %q", sentAddress)
	}
	if sentFrom != "noreply@example.com" {
		t.Fatalf("SMTP sender = %q", sentFrom)
	}
	if len(sentTo) != 1 || sentTo[0] != "buyer@example.com" {
		t.Fatalf("SMTP recipients = %#v", sentTo)
	}
	if !strings.Contains(string(sentMessage), "654321") {
		t.Fatalf("email message does not contain OTP: %q", sentMessage)
	}
}
