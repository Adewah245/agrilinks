package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFlutterwavePermanentAccountResponses(t *testing.T) {
	for _, test := range []struct {
		name, body string
		code       int
		want       error
	}{
		{"created", `{"status":"success","data":{"tx_ref":"VA-test","response_code":"02","account_number":"0123456789","bank_name":"Test Bank"}}`, 201, nil},
		{"wrong reference", `{"status":"success","data":{"tx_ref":"another-user"}}`, 200, ErrReferenceMismatch},
		{"unaccepted response", `{"status":"success","data":{"response_code":"99","account_number":"0123456789"}}`, 200, ErrPaymentProvider},
		{"credentials", `{"status":"error","message":"secret identity 00000000000"}`, 403, ErrProviderAccessDenied},
		{"rejected identity", `{"status":"error","message":"invalid BVN 00000000000"}`, 400, ErrProviderRefused},
		{"duplicate is uncertain", `{"status":"error","message":"reference already exists"}`, 400, ErrPaymentProvider},
		{"unreadable", `{"status":`, 200, ErrPaymentProvider},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/virtual-account-numbers" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("unexpected account request")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["bvn"] != "00000000000" || body["nin"] != nil || body["is_permanent"] != true || body["tx_ref"] != "VA-test" {
					t.Error("incorrect identity or permanent-account payload")
				}
				w.WriteHeader(test.code)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			client := NewFlutterwaveClient("test-key", "", "")
			client.SetBaseURL(server.URL)
			account, err := client.CreatePermanentVirtualAccount(context.Background(), CreateVirtualAccountRequest{Email: "user@example.test", TxRef: "VA-test", IdentityKind: "bvn", IdentityNumber: "00000000000", Narration: "Test Holder"})
			if test.want == nil {
				if err != nil || account.AccountName != "Test Holder" || account.Currency != "NGN" {
					t.Fatalf("account defaults were not retained: %v", err)
				}
			} else if !errors.Is(err, test.want) {
				t.Fatalf("got %v; want %v", err, test.want)
			}
			if err != nil && strings.Contains(err.Error(), "00000000000") {
				t.Fatal("provider identity data leaked into error")
			}
		})
	}
}

func TestFlutterwaveMissingAccountLookup(t *testing.T) {
	for _, code := range []int{200, 400, 404} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"status":"error","message":"No virtual account found"}`))
		}))
		client := NewFlutterwaveClient("test-key", "", "")
		client.SetBaseURL(server.URL)
		_, err := client.GetPermanentVirtualAccount(context.Background(), "order-ref")
		server.Close()
		if !errors.Is(err, ErrProviderNotFound) {
			t.Fatalf("HTTP %d: %v", code, err)
		}
	}
}
