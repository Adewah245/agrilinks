package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type flutterwaveRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn flutterwaveRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFlutterwaveInitializePayment(t *testing.T) {
	client := NewFlutterwaveClient(
		"secret-key",
		"https://agro.example/payments/callback",
		"card,banktransfer",
	)

	client.httpClient = &http.Client{Transport: flutterwaveRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", req.Method)
		}
		if got := req.URL.String(); got != flutterwaveBaseURL+"/payments" {
			t.Fatalf("URL = %q", got)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("Authorization = %q", got)
		}

		var payload initializePaymentPayload
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.TxRef != "ORDER-9-reference" || payload.Amount != "50000.00" || payload.Currency != "NGN" {
			t.Fatalf("unexpected payment payload: %#v", payload)
		}
		if payload.RedirectURL != "https://agro.example/payments/callback" {
			t.Fatalf("redirect URL = %q", payload.RedirectURL)
		}
		if payload.PaymentOptions != "card,banktransfer" {
			t.Fatalf("payment options = %q", payload.PaymentOptions)
		}

		return jsonHTTPResponse(http.StatusOK, `{"status":"success","data":{"link":"https://checkout.flutterwave.test/session"}}`), nil
	})}

	link, err := client.InitializePayment(context.Background(), InitializePaymentRequest{
		Reference:     "ORDER-9-reference",
		Amount:        "50000.00",
		Currency:      "NGN",
		CustomerEmail: "buyer@example.test",
		CustomerName:  "Buyer",
		Description:   "Order 9",
	})
	if err != nil {
		t.Fatalf("InitializePayment() error = %v", err)
	}
	if link != "https://checkout.flutterwave.test/session" {
		t.Fatalf("checkout link = %q", link)
	}
}

func TestFlutterwaveVerifyTransaction(t *testing.T) {
	client := NewFlutterwaveClient("secret-key", "https://agro.example/payments/callback", "")
	client.httpClient = &http.Client{Transport: flutterwaveRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", req.Method)
		}
		if got := req.URL.Path; got != "/v3/transactions/123/verify" {
			t.Fatalf("path = %q", got)
		}
		return jsonHTTPResponse(http.StatusOK, `{"status":"success","data":{"id":123,"tx_ref":"ORDER-9-reference","status":"successful","amount":50000,"currency":"NGN"}}`), nil
	})}

	verified, err := client.VerifyTransaction(context.Background(), "123")
	if err != nil {
		t.Fatalf("VerifyTransaction() error = %v", err)
	}
	if verified.TransactionID != "123" || verified.Reference != "ORDER-9-reference" {
		t.Fatalf("unexpected verification: %#v", verified)
	}
	if verified.Amount.String() != "50000" || verified.Currency != "NGN" || verified.Status != "successful" {
		t.Fatalf("unexpected verification: %#v", verified)
	}
}

func TestFlutterwaveVerifyRejectsInvalidTransactionID(t *testing.T) {
	client := NewFlutterwaveClient("secret-key", "https://agro.example/payments/callback", "")
	called := false
	client.httpClient = &http.Client{Transport: flutterwaveRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return jsonHTTPResponse(http.StatusOK, `{}`), nil
	})}

	if _, err := client.VerifyTransaction(context.Background(), "../123"); err == nil {
		t.Fatal("expected invalid transaction ID to be rejected")
	}
	if called {
		t.Fatal("invalid transaction ID reached the provider")
	}
}

func TestFlutterwaveRejectsInvalidCheckoutURL(t *testing.T) {
	client := NewFlutterwaveClient("secret-key", "https://agro.example/payments/callback", "")
	client.httpClient = &http.Client{Transport: flutterwaveRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{"status":"success","data":{"link":"javascript:alert(1)"}}`), nil
	})}

	if _, err := client.InitializePayment(context.Background(), InitializePaymentRequest{
		Reference:     "ref",
		Amount:        "1.00",
		Currency:      "NGN",
		CustomerEmail: "buyer@example.test",
	}); err == nil {
		t.Fatal("expected invalid checkout URL to be rejected")
	}
}
