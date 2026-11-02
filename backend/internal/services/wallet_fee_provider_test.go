package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderFeesUseVerifiedAPIShape(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		want       int64
		fail       bool
	}{
		{"actual zero fee", `{"status":"success","data":[{"currency":"NGN","fee_type":"value","fee":0}]}`, 0, false},
		{"integer kobo", `{"status":"success","data":[{"currency":"NGN","fee_type":"value","fee":10.75}]}`, 1075, false},
		{"different currency", `{"status":"success","data":[{"currency":"USD","fee_type":"value","fee":10}]}`, 0, true},
		{"ambiguous fees", `{"status":"success","data":[{"currency":"NGN","fee_type":"value","fee":10},{"currency":"NGN","fee_type":"value","fee":20}]}`, 0, true},
		{"percentage rejected", `{"status":"success","data":[{"currency":"NGN","fee_type":"percentage","fee":1}]}`, 0, true},
		{"provider failure", `{"status":"error","message":"unavailable"}`, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/transfers/fee" || r.URL.Query().Get("currency") != "NGN" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("incorrect provider fee request")
				}
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			client := NewFlutterwaveClient("test-key", "", "")
			client.SetBaseURL(server.URL)
			quote, err := client.WithdrawalFee(context.Background(), "10000.00", "NGN")
			if tt.fail {
				if err == nil {
					t.Fatal("invalid provider fee accepted")
				}
				return
			}
			if err != nil || quote.FeeKobo != tt.want {
				t.Fatalf("fee=%+v error=%v", quote, err)
			}
		})
	}
}
func TestCollectionFeesSeparateCustomerAndMerchantCharges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("payment_type") != "bank_transfer" || r.URL.Query().Get("amount") != "10050.00" {
			t.Error("incorrect gross amount or payment method")
		}
		fmt.Fprint(w, `{"status":"success","data":{"fee":140,"charge_amount":10190,"merchant_fee":0,"flutterwave_fee":140,"stamp_duty_fee":0,"currency":"NGN"}}`)
	}))
	defer server.Close()
	client := NewFlutterwaveClient("test-key", "", "")
	client.SetBaseURL(server.URL)
	q, err := client.CollectionFee(context.Background(), "10050.00", "NGN", "banktransfer")
	if err != nil || q.FeeKobo != 14000 || q.ChargeKobo != 1019000 {
		t.Fatalf("fee quote=%+v %v", q, err)
	}
	if _, err = client.CollectionFee(context.Background(), "10050.00", "NGN", "unknown"); err == nil {
		t.Fatal("unsupported method accepted")
	}
}
