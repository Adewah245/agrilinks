package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestWalletBankVerificationAndSaveWithPostgres(t *testing.T) {
	db, _ := testutil.Database(t, true)
	var lookups atomic.Int32
	var unavailable atomic.Bool
	var bankName atomic.Value
	bankName.Store("  Test User 1  ")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/banks/NG" {
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Test Bank","code":"001"},{"name":"Other Bank","code":"002"}]}`)
			return
		}
		if r.URL.Path != "/accounts/resolve" || r.Method != http.MethodPost {
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		lookups.Add(1)
		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status":"error","message":"private provider diagnostic"}`)
			return
		}
		var input struct {
			Bank   string `json:"account_bank"`
			Number string `json:"account_number"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.Number != "0123456789" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"status":"error","message":"Invalid account number"}`)
			return
		}
		name := bankName.Load().(string)
		if input.Bank == "002" {
			name = "Test User 2"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]string{
			"account_number": input.Number, "account_name": name,
		}})
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.FlutterwaveSecretKey = "test-bank-key"
	cfg.FlutterwaveSecretHash = "test-bank-hash"
	cfg.FlutterwaveBaseURL = provider.URL
	cfg.PaymentRedirectURL = "http://example.test/payments/callback"
	c := application(t, db, cfg)
	user := fixtureUser(t, c, 1, "buyer")
	pinHash, err := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UserRepo.CreateTransactionPIN(t.Context(), user.ID, string(pinHash)); err != nil {
		t.Fatal(err)
	}
	handler := routes.NewHandler(c)
	cookie := session(t, c, user)
	details := url.Values{"bank_code": {"001"}, "account_number": {"0123456789"}, "account_name": {"Forged Name"}}
	lookup := func(form url.Values, status int) *httptest.ResponseRecorder {
		t.Helper()
		response := request(handler, "POST", "/wallet/resolve-account", form, cookie)
		requireStatus(t, response, status)
		if response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
			t.Fatal("account lookups must return uncached JSON")
		}
		return response
	}
	assertSaved := func(name string) {
		t.Helper()
		profile, err := c.Payment.PaymentProfile(user.ID)
		if err != nil || profile == nil {
			t.Fatalf("load bank profile: %+v %v", profile, err)
		}
		if profile.AccountName != name || (name != "" && (profile.BankCode != "001" || profile.BankName != "Test Bank" || profile.AccountNumber != "0123456789")) {
			t.Fatalf("saved bank details changed unexpectedly: %+v", profile)
		}
	}

	requireStatus(t, request(handler, "POST", "/wallet/resolve-account", details, nil), http.StatusSeeOther)
	requireStatus(t, request(handler, "GET", "/wallet/resolve-account", nil, cookie), http.StatusMethodNotAllowed)
	crossOrigin := httptest.NewRequest("POST", "http://example.test/wallet/resolve-account", strings.NewReader(details.Encode()))
	crossOrigin.Header.Set("Origin", "https://other.example")
	crossOrigin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	crossOrigin.AddCookie(cookie)
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, crossOrigin)
	requireStatus(t, blocked, http.StatusForbidden)
	if lookups.Load() != 0 {
		t.Fatal("unauthenticated or cross-origin request reached the provider")
	}
	lookup(url.Values{"bank_code": {"001"}, "account_number": {"123"}}, http.StatusBadRequest)
	response := lookup(details, http.StatusOK)
	var account services.BankAccountVerification
	if err := json.Unmarshal(response.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	if account.AccountName != user.FullName || !account.MatchesProfile || account.AccountNumber != "0123456789" || account.BankCode != "001" {
		t.Fatalf("incorrect verified name: %+v", account)
	}
	assertSaved("")

	form := url.Values{"action": {"save_bank"}, "bank_code": {"001"}, "account_number": {"0123456789"}, "account_name": {account.AccountName}, "transaction_pin": {"invalid"}}
	before := lookups.Load()
	requireStatus(t, request(handler, "POST", "/wallet", form, cookie), http.StatusBadRequest)
	form.Set("transaction_pin", "582491")
	form.Del("account_name")
	requireStatus(t, request(handler, "POST", "/wallet", form, cookie), http.StatusBadRequest)
	if lookups.Load() != before {
		t.Fatal("missing PIN or name confirmation reached the provider")
	}
	assertSaved("")
	form.Set("account_name", "Forged Name")
	requireStatus(t, request(handler, "POST", "/wallet", form, cookie), http.StatusBadRequest)
	assertSaved("")
	form.Set("account_name", account.AccountName)
	response = request(handler, "POST", "/wallet", form, cookie)
	requireStatus(t, response, http.StatusSeeOther)
	if response.Header().Get("Location") != "/wallet?notice=bank_saved" {
		t.Fatalf("missing bank confirmation: %s", response.Header().Get("Location"))
	}
	assertSaved(user.FullName)

	// The same number at a different bank can belong to somebody else.
	details.Set("bank_code", "002")
	response = lookup(details, http.StatusOK)
	if err := json.Unmarshal(response.Body.Bytes(), &account); err != nil || account.MatchesProfile || account.AccountName != "Test User 2" {
		t.Fatalf("different bank owner was accepted: %+v %v", account, err)
	}
	form.Set("bank_code", "002")
	form.Set("account_name", account.AccountName)
	requireStatus(t, request(handler, "POST", "/wallet", form, cookie), http.StatusBadRequest)
	assertSaved(user.FullName)

	// A successful preview does not allow a changed name or a later outage to save.
	form.Set("bank_code", "001")
	form.Set("account_name", user.FullName)
	bankName.Store("User 1 Test")
	response = request(handler, "POST", "/wallet", form, cookie)
	requireStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(response.Body.String(), "different account name") {
		t.Fatal("a changed name must require a new review")
	}
	assertSaved(user.FullName)
	unavailable.Store(true)
	details.Set("bank_code", "001")
	response = lookup(details, http.StatusServiceUnavailable)
	if strings.Contains(response.Body.String(), "private provider diagnostic") {
		t.Fatal("provider diagnostic exposed")
	}
	requireStatus(t, request(handler, "POST", "/wallet", form, cookie), http.StatusBadRequest)
	assertSaved(user.FullName)
	unavailable.Store(false)
	details.Set("account_number", "1111111111")
	lookup(details, http.StatusUnprocessableEntity)
	details.Set("account_number", "0123456789")
	for range 5 {
		lookup(details, http.StatusOK)
	}
	before = lookups.Load()
	response = lookup(details, http.StatusTooManyRequests)
	if response.Header().Get("Retry-After") != "60" || lookups.Load() != before {
		t.Fatal("rate-limited lookup reached the provider or omitted retry guidance")
	}
}
