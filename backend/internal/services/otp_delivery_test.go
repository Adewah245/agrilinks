package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestTermiiOTPDeliverySendsExpectedRequest(t *testing.T) {
	var request struct {
		APIKey  string `json:"api_key"`
		To      string `json:"to"`
		From    string `json:"from"`
		SMS     string `json:"sms"`
		Type    string `json:"type"`
		Channel string `json:"channel"`
	}

	delivery, err := NewTermiiOTPDelivery("api-key", "AgroShield", "https://termii.test")
	if err != nil {
		t.Fatal(err)
	}
	delivery.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/sms/send" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"code":"ok"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	err = delivery.SendOTP(context.Background(), OTPMessage{
		Phone:     "08012345678",
		Purpose:   "transfer",
		Code:      "123456",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("SendOTP() error = %v", err)
	}

	if request.APIKey != "api-key" || request.To != "08012345678" || request.From != "AgroShield" {
		t.Fatalf("unexpected Termii request: %#v", request)
	}
	if !strings.Contains(request.SMS, "123456") {
		t.Fatalf("SMS body does not contain OTP: %q", request.SMS)
	}
	if request.Type != "plain" || request.Channel != "generic" {
		t.Fatalf("unexpected Termii message settings: %#v", request)
	}
}

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
