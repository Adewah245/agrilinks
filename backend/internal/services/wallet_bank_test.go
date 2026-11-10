package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"backend/internal/models"
)

func TestBankAccountVerificationUsesSelectedBankAndProviderName(t *testing.T) {
	var lookups atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-bank-key" {
			t.Error("bank lookup did not use the server's provider credential")
		}
		switch r.URL.Path {
		case "/banks/NG":
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Fintech Bank","code":"100004"}]}`)
		case "/accounts/resolve":
			lookups.Add(1)
			var input map[string]string
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || input["account_number"] != "0123456789" || input["account_bank"] != "100004" || len(input) != 2 {
				t.Errorf("wrong account lookup: %s %+v", r.Method, input)
			}
			fmt.Fprint(w, `{"status":"success","data":{"account_number":"0123456789","account_name":"  FARMER, ADA  "}}`)
		default:
			t.Errorf("unexpected provider request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	client := NewFlutterwaveClient("test-bank-key", "", "")
	client.SetBaseURL(provider.URL)
	service := NewWalletService(nil, nil, client, nil, "NGN")
	user := &models.User{ID: 1, FullName: "Ada Farmer"}
	for range 2 {
		account, err := service.ResolveBankAccount(t.Context(), user, " 100004 ", " 0123456789 ")
		if err != nil {
			t.Fatal(err)
		}
		if account.BankCode != "100004" || account.BankName != "Fintech Bank" || account.AccountNumber != "0123456789" || account.AccountName != "FARMER, ADA" || !account.MatchesProfile {
			t.Fatalf("incorrect verification: %+v", account)
		}
	}
	if lookups.Load() != 2 {
		t.Fatal("account names must be verified afresh, even when the bank directory is cached")
	}
}

func TestBankAccountVerificationRejectsInvalidDetailsBeforeProviderLookup(t *testing.T) {
	var lookups atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/banks/NG" {
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Test Bank","code":"001"}]}`)
			return
		}
		lookups.Add(1)
		http.Error(w, "unexpected account lookup", http.StatusInternalServerError)
	}))
	defer provider.Close()
	client := NewFlutterwaveClient("test-key", "", "")
	client.SetBaseURL(provider.URL)
	service := NewWalletService(nil, nil, client, nil, "NGN")
	user := &models.User{ID: 1, FullName: "Ada Farmer"}
	for _, test := range []struct {
		name, bank, number string
		want               error
	}{
		{"missing number", "001", "", ErrBankAccountNumber},
		{"short number", "001", "123456789", ErrBankAccountNumber},
		{"long number", "001", "01234567890", ErrBankAccountNumber},
		{"letters", "001", "012345678a", ErrBankAccountNumber},
		{"missing bank", "", "0123456789", ErrBankSelection},
		{"unknown bank", "unlisted-bank", "0123456789", ErrBankSelection},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.ResolveBankAccount(t.Context(), user, test.bank, test.number)
			if result != nil || !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %+v %v", test.want, result, err)
			}
		})
	}
	if account, err := service.ResolveBankAccount(t.Context(), nil, "001", "0123456789"); err == nil || account != nil {
		t.Fatal("anonymous lookup accepted")
	}
	if lookups.Load() != 0 {
		t.Fatal("invalid details reached the account resolver")
	}
	service = NewWalletService(nil, nil, nil, nil, "NGN")
	if account, err := service.ResolveBankAccount(t.Context(), user, "001", "0123456789"); account != nil || !errors.Is(err, ErrBankVerificationUnavailable) {
		t.Fatalf("unconfigured provider: %+v %v", account, err)
	}
}

func TestBankAccountVerificationFailsClosedOnProviderErrors(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		directory  bool
		want       error
	}{
		{"invalid account", `{"status":"error","message":"Invalid account"}`, 400, false, ErrBankAccountUnconfirmed},
		{"account missing", `{"status":"error","message":"Account not found"}`, 404, false, ErrBankAccountUnconfirmed},
		{"unconfirmed account", `{"status":"error"}`, 422, false, ErrBankAccountUnconfirmed},
		{"missing account in success response", `{"status":"error","message":"No account found"}`, 200, false, ErrBankAccountUnconfirmed},
		{"different number", `{"status":"success","data":{"account_number":"9999999999","account_name":"Ada Farmer"}}`, 200, false, ErrBankAccountUnconfirmed},
		{"empty name", `{"status":"success","data":{"account_number":"0123456789","account_name":"  "}}`, 200, false, ErrBankAccountUnconfirmed},
		{"missing data", `{"status":"success","data":null}`, 200, false, ErrBankAccountUnconfirmed},
		{"bad credential", `{"status":"error"}`, 401, false, ErrBankVerificationUnavailable},
		{"access denied", `{"status":"error"}`, 403, false, ErrBankVerificationUnavailable},
		{"provider throttled", `{"status":"error"}`, 429, false, ErrBankVerificationUnavailable},
		{"provider outage", `{"status":"error"}`, 503, false, ErrBankVerificationUnavailable},
		{"invalid response", `not json`, 200, false, ErrBankVerificationUnavailable},
		{"directory outage", `{"status":"error"}`, 503, true, ErrBankVerificationUnavailable},
		{"empty directory", `{"status":"success","data":[]}`, 200, true, ErrBankVerificationUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/banks/NG" && !test.directory {
					fmt.Fprint(w, `{"status":"success","data":[{"name":"Test Bank","code":"001"}]}`)
					return
				}
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer provider.Close()
			client := NewFlutterwaveClient("test-key", "", "")
			client.SetBaseURL(provider.URL)
			service := NewWalletService(nil, nil, client, nil, "NGN")
			account, err := service.ResolveBankAccount(t.Context(), &models.User{ID: 1, FullName: "Ada Farmer"}, "001", "0123456789")
			if account != nil || !errors.Is(err, test.want) {
				t.Fatalf("expected %v without account details, got %+v %v", test.want, account, err)
			}
		})
	}
}

func TestBankAccountVerificationReportsDifferentOwnerWithoutReplacingTheName(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/banks/NG" {
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Test Bank","code":"001"}]}`)
			return
		}
		fmt.Fprint(w, `{"status":"success","data":{"account_number":"0123456789","account_name":"Another Owner"}}`)
	}))
	defer provider.Close()
	client := NewFlutterwaveClient("test-key", "", "")
	client.SetBaseURL(provider.URL)
	service := NewWalletService(nil, nil, client, nil, "NGN")
	account, err := service.ResolveBankAccount(t.Context(), &models.User{ID: 1, FullName: "Ada Farmer"}, "001", "0123456789")
	if err != nil || account == nil || account.MatchesProfile || account.AccountName != "Another Owner" {
		t.Fatalf("mismatched owner was hidden or treated as a match: %+v %v", account, err)
	}
}
