package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

const otpDevTestConfirmation = `{"account_id":"account-1","message_id":"message-1","phone":"2348012345678","create_date":"2026-10-24T10:00:00Z","expire_date":"2026-10-24T12:00:00Z"}`

func otpDevTestDelivery(t *testing.T, transport roundTripFunc) *OTPDevDelivery {
	t.Helper()
	delivery, err := NewOTPDevDelivery("test-api-key", "AgroShield", "6d16aa9d-bf19-4141-8169-48b46d972fc6", "")
	if err != nil {
		t.Fatal(err)
	}
	delivery.client.Transport = transport
	return delivery
}

func otpDevResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestOTPDevDeliveryUsesDocumentedRequestAndStoredCode(t *testing.T) {
	for _, purpose := range []string{"password_reset", "withdrawal", "transfer"} {
		t.Run(purpose, func(t *testing.T) {
			var sentCode string
			delivery := otpDevTestDelivery(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.String() != "https://api.otp.dev/v1/verifications" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("X-OTP-Key") != "test-api-key" || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatal("missing OTP.dev request headers")
				}
				var payload struct {
					Data map[string]string `json:"data"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				data := payload.Data
				if data["channel"] != "sms" || data["phone"] != "2348012345678" || data["sender"] != "AgroShield" || data["template"] != "6d16aa9d-bf19-4141-8169-48b46d972fc6" {
					t.Fatalf("unexpected delivery payload: %#v", data)
				}
				if _, ok := data["code_length"]; ok {
					t.Fatal("code_length must not accompany our supplied code")
				}
				sentCode = data["code"]
				return otpDevResponse(http.StatusOK, otpDevTestConfirmation), nil
			})
			store := &fakeOTPStore{}
			service := newTestOTPService(store, nil)
			service.delivery = delivery
			if err := service.Issue(context.Background(), 7, "08012345678", "", purpose); err != nil {
				t.Fatal(err)
			}
			if len(sentCode) != 6 || store.created == nil {
				t.Fatal("expected a stored and delivered six-digit OTP")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(store.created.CodeHash), []byte(sentCode)); err != nil {
				t.Fatal("OTP.dev received a different code from the stored hash")
			}
			if err := service.Verify(7, purpose, sentCode); err != nil {
				t.Fatalf("delivered code did not verify: %v", err)
			}
			if store.consumed != 1 {
				t.Fatal("verified code was not consumed")
			}
		})
	}
}

func TestOTPDevDeliveryRejectsFailureAndInvalidConfirmations(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, `{"message":"test-api-key 123456 2348012345678"}`},
		{"rate limited", 429, `{"errors":[{"code":"limit"}]}`},
		{"provider unavailable", 503, "unavailable"},
		{"invalid JSON", 200, "not JSON"},
		{"empty confirmation", 200, `{}`},
		{"wrong phone", 200, `{"message_id":"message-1","phone":"2348099999999"}`},
		{"no message ID", 200, `{"phone":"2348012345678"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delivery := otpDevTestDelivery(t, func(*http.Request) (*http.Response, error) {
				return otpDevResponse(test.status, test.body), nil
			})
			err := delivery.SendOTP(context.Background(), OTPMessage{Phone: "08012345678", Code: "123456"})
			if err == nil {
				t.Fatal("expected delivery to fail")
			}
			for _, secret := range []string{"test-api-key", "123456", "2348012345678"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("provider error exposed sensitive data")
				}
			}
		})
	}
}

func TestOTPDevDeliveryFailureRemovesUnsentCode(t *testing.T) {
	store := &fakeOTPStore{}
	service := newTestOTPService(store, nil)
	service.delivery = otpDevTestDelivery(t, func(*http.Request) (*http.Response, error) {
		return otpDevResponse(400, `{"errors":[{"code":"1512","message":"Invalid template id"}]}`), nil
	})
	if err := service.Issue(context.Background(), 7, "08012345678", "", "password_reset"); err == nil {
		t.Fatal("failed delivery reported success")
	}
	if store.deleted != 1 {
		t.Fatal("failed delivery left an active code")
	}
}

func TestOTPDevDeliveryDoesNotFollowRedirects(t *testing.T) {
	calls := 0
	delivery := otpDevTestDelivery(t, func(*http.Request) (*http.Response, error) {
		calls++
		response := otpDevResponse(http.StatusTemporaryRedirect, "")
		response.Header.Set("Location", "https://other.example.test/collect")
		return response, nil
	})
	if err := delivery.SendOTP(context.Background(), OTPMessage{Phone: "08012345678", Code: "123456"}); err == nil {
		t.Fatal("redirect reported success")
	}
	if calls != 1 {
		t.Fatalf("followed redirect with credentials: %d requests", calls)
	}
}

func TestOTPDevDeliveryHonorsCancellation(t *testing.T) {
	delivery := otpDevTestDelivery(t, func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := delivery.SendOTP(ctx, OTPMessage{Phone: "08012345678", Code: "123456"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

func TestOTPDevDeliveryValidatesPhoneAndCodeBeforeSending(t *testing.T) {
	delivery := otpDevTestDelivery(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid input reached the provider")
		return nil, nil
	})
	for _, message := range []OTPMessage{
		{Phone: "", Code: "123456"},
		{Phone: "123", Code: "123456"},
		{Phone: "08012345678", Code: ""},
		{Phone: "08012345678", Code: "12A456"},
		{Phone: "08012345678", Code: "12345"},
	} {
		if err := delivery.SendOTP(context.Background(), message); err == nil {
			t.Fatal("invalid input was accepted")
		}
	}
}
