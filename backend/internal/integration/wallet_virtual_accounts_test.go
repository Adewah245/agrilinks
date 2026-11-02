package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
)

func TestWalletVirtualAccountWorkflow(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	var mu sync.Mutex
	mode := "active"
	creates := map[string]int{}
	accounts := map[string]map[string]any{}
	transactions := map[string]map[string]any{}
	latest := ""
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer test-va-key" {
			t.Error("missing provider authentication")
		}
		respond := func(data any) { _ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data}) }
		switch {
		case r.Method == "POST" && r.URL.Path == "/virtual-account-numbers":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				return
			}
			ref, _ := payload["tx_ref"].(string)
			creates[ref]++
			if payload["bvn"] != "00000000000" || payload["is_permanent"] != true || payload["firstname"] != "Test" {
				t.Error("identity payload did not use authenticated profile")
			}
			if mode == "uncertain" {
				w.WriteHeader(503)
				return
			}
			if mode == "rejected" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"status":"error","message":"invalid identity 00000000000"}`))
				return
			}
			id, _ := strconv.Atoi(strings.TrimPrefix(ref, "AGRO-VA-"))
			data := map[string]any{"tx_ref": ref, "order_ref": "ORDER-" + ref, "account_number": fmt.Sprintf("%010d", id), "bank_name": "Test Funding Bank", "currency": "NGN", "account_status": "active"}
			accounts["ORDER-"+ref] = data
			if mode == "delayed" {
				respond(map[string]any{"tx_ref": ref, "order_ref": "ORDER-" + ref, "account_status": "pending"})
				return
			}
			if mode == "inactive" {
				data["account_status"] = "inactive"
			}
			respond(data)
		case strings.HasPrefix(r.URL.Path, "/virtual-account-numbers/"):
			data, ok := accounts[strings.TrimPrefix(r.URL.Path, "/virtual-account-numbers/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			respond(data)
		case r.URL.Path == "/transactions/verify_by_reference":
			data, ok := transactions[latest]
			if !ok || data["tx_ref"] != r.URL.Query().Get("tx_ref") {
				http.NotFound(w, r)
				return
			}
			respond(data)
		case strings.HasPrefix(r.URL.Path, "/transactions/"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/transactions/"), "/verify")
			data, ok := transactions[id]
			if !ok {
				http.NotFound(w, r)
				return
			}
			respond(data)
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.FlutterwaveSecretKey = "test-va-key"
	cfg.FlutterwaveSecretHash = "test-va-hash"
	cfg.FlutterwaveBaseURL = provider.URL
	c := application(t, db, cfg)
	user := fixtureUser(t, c, 1, "buyer")
	other := fixtureUser(t, c, 2, "farmer")
	admin := fixtureUser(t, c, 3, "buyer")
	if err := c.Admin.Bootstrap(ctx, admin.Phone); err != nil {
		t.Fatal(err)
	}
	admin, err := c.UserRepo.GetByID(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	handler := routes.NewHandler(c)
	cookie := session(t, c, user)
	setup := services.VirtualAccountSetup{IdentityKind: "bvn", IdentityNumber: "00000000000", Consent: true}
	ref := fmt.Sprintf("AGRO-VA-%d", user.ID)
	form := url.Values{"action": {"setup_funding"}, "identity_kind": {"bvn"}, "identity_number": {"00000000000"}, "consent": {"yes"}, "firstname": {"Untrusted"}}
	execSQL := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	age := func(id int) {
		execSQL(`UPDATE wallet_virtual_accounts SET updated_at=CURRENT_TIMESTAMP-interval '3 minutes',last_checked_at=CURRENT_TIMESTAMP-interval '3 minutes' WHERE user_id=$1`, id)
	}
	setMode := func(value string) { mu.Lock(); mode = value; mu.Unlock() }
	balance := func(want string) {
		t.Helper()
		page, err := c.Wallet.Page(ctx, user)
		if err != nil || page.Wallet.Balance != want {
			t.Fatalf("balance should be %s: page=%v error=%v", want, page, err)
		}
	}
	transaction := func(id, status, amount string) {
		mu.Lock()
		_, already := transactions[id]
		mu.Unlock()
		if !already {
			q, err := c.Wallet.Quote(ctx, user, "funding_preview", amount, "")
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Wallet.AcceptFundingIntent(ctx, user.ID, q.ID); err != nil {
				t.Fatal(err)
			}
		}
		principal, err := money.Positive(amount)
		if err != nil {
			t.Fatal(err)
		}
		amount = money.Format(principal + money.FundingFee)
		mu.Lock()
		defer mu.Unlock()
		number, _ := strconv.Atoi(id)
		transactions[id] = map[string]any{"id": number, "tx_ref": ref, "status": status, "amount": json.Number(amount), "currency": "NGN"}
	}
	webhook := func(id string) error {
		return c.Payment.ProcessWebhook(ctx, []byte(fmt.Sprintf(`{"event":"charge.completed","data":{"id":%s,"tx_ref":%q}}`, id, ref)), "test-va-hash")
	}

	t.Run("setup authentication consent and privacy", func(t *testing.T) {
		requireStatus(t, request(handler, "POST", "/wallet", form, nil), 303)
		form.Del("consent")
		response := request(handler, "POST", "/wallet", form, cookie)
		requireStatus(t, response, 400)
		if strings.Contains(response.Body.String(), "00000000000") {
			t.Fatal("identity echoed into error page")
		}
		form.Set("consent", "yes")
		r := httptest.NewRequest("POST", "http://example.test/wallet", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "https://untrusted.example")
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		requireStatus(t, w, 403)
		requireStatus(t, request(handler, "POST", "/wallet", form, cookie), 303)
		page, err := c.Wallet.Page(ctx, user)
		if err != nil || !page.VirtualAccount.Ready() {
			t.Fatalf("account was not activated: %v", err)
		}
		var stored string
		if err := db.QueryRow(`SELECT row_to_json(a)::text FROM wallet_virtual_accounts a WHERE user_id=$1`, user.ID).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stored, "00000000000") {
			t.Fatal("raw identity persisted")
		}
		status := request(handler, "GET", fmt.Sprintf("/wallet/status?user_id=%d", other.ID), nil, cookie)
		requireStatus(t, status, 200)
		if !strings.Contains(status.Body.String(), page.Revision) || status.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("status did not use authenticated owner's private revision")
		}
	})

	t.Run("concurrent setup creates once", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := c.Wallet.SetupVirtualAccount(ctx, other, setup); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
		mu.Lock()
		defer mu.Unlock()
		if creates[fmt.Sprintf("AGRO-VA-%d", other.ID)] != 1 {
			t.Fatal("concurrent calls created multiple accounts")
		}
	})

	t.Run("partial response recovers saved order reference", func(t *testing.T) {
		u := fixtureUser(t, c, 4, "buyer")
		setMode("delayed")
		a, err := c.Wallet.SetupVirtualAccount(ctx, u, setup)
		if err != nil || a.Ready() || a.OrderRef == "" {
			t.Fatalf("partial response lost its reference: %v", err)
		}
		age(u.ID)
		setMode("active")
		a, err = c.Wallet.RefreshVirtualAccount(ctx, u.ID, u.FullName)
		if err != nil || !a.Ready() {
			t.Fatalf("account did not recover: %v", err)
		}
	})

	t.Run("uncertain creation is never duplicated", func(t *testing.T) {
		u := fixtureUser(t, c, 5, "buyer")
		setMode("uncertain")
		a, err := c.Wallet.SetupVirtualAccount(ctx, u, setup)
		if err != nil || a.Status != models.VirtualAccountPending {
			t.Fatalf("uncertain request must stay pending: %v", err)
		}
		age(u.ID)
		setMode("active")
		if _, err := c.Wallet.SetupVirtualAccount(ctx, u, setup); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		defer mu.Unlock()
		if creates[fmt.Sprintf("AGRO-VA-%d", u.ID)] != 1 {
			t.Fatal("ambiguous creation retried after lookup by noncanonical reference")
		}
	})

	t.Run("refused setup can retry after cooldown", func(t *testing.T) {
		u := fixtureUser(t, c, 6, "buyer")
		setMode("rejected")
		if _, err := c.Wallet.SetupVirtualAccount(ctx, u, setup); err != services.ErrVirtualAccountRejected {
			t.Fatalf("got %v", err)
		}
		setMode("active")
		a, err := c.Wallet.SetupVirtualAccount(ctx, u, setup)
		if err != nil || a.Status != models.VirtualAccountFailed {
			t.Fatal("refused requests must be throttled")
		}
		age(u.ID)
		a, err = c.Wallet.SetupVirtualAccount(ctx, u, setup)
		if err != nil || !a.Ready() {
			t.Fatalf("retry failed: %v", err)
		}
	})

	t.Run("inactive provider account remains hidden", func(t *testing.T) {
		u := fixtureUser(t, c, 7, "buyer")
		setMode("inactive")
		defer setMode("active")
		a, err := c.Wallet.SetupVirtualAccount(ctx, u, setup)
		if err != nil || a.Ready() {
			t.Fatalf("inactive provider account was activated: %v", err)
		}
	})

	t.Run("pending event can later settle and repeats credit once", func(t *testing.T) {
		transaction("12001", "pending", "10.00")
		if err := webhook("12001"); err != nil {
			t.Fatal(err)
		}
		balance("0.00")
		transaction("12001", "successful", "10.00")
		for i := 0; i < 3; i++ {
			if err := webhook("12001"); err != nil {
				t.Fatal(err)
			}
		}
		balance("10.00")
		transaction("12002", "successful", "50.00")
		if err := webhook("12002"); err != nil {
			t.Fatal(err)
		}
		balance("60.00")
		transaction("12003", "successful", "20.00")
		mu.Lock()
		transactions["12003"]["currency"] = "USD"
		mu.Unlock()
		if err := webhook("12003"); err == nil {
			t.Fatal("wrong currency credited")
		}
		balance("60.00")
	})

	t.Run("background recovery and older transfer recovery", func(t *testing.T) {
		transaction("12004", "pending", "7.00")
		if err := webhook("12004"); err != nil {
			t.Fatal(err)
		}
		transaction("12004", "successful", "7.00")
		transaction("12005", "successful", "5.00")
		mu.Lock()
		latest = "12005"
		mu.Unlock()
		age(user.ID)
		execSQL(`UPDATE wallet_deposits SET updated_at=CURRENT_TIMESTAMP-interval '3 minutes' WHERE reference='VA-12004'`)
		if err := c.Wallet.ReconcileFunding(ctx); err != nil {
			t.Fatal(err)
		}
		balance("72.00")
		transaction("12006", "successful", "42.00")
		if err := c.Wallet.RecoverFundingTransfer(ctx, user, "12006"); err == nil {
			t.Fatal("member recovered a transfer with a finance-only action")
		}
		adminCookie := session(t, c, admin)
		values := adminForm(t, handler, adminCookie, "/admin/finance", url.Values{"action": {"wallet_funding_check"}, "id": {"12006"}, "reason": {"Verify older missing transfer"}})
		for i := 0; i < 2; i++ {
			requireStatus(t, request(handler, "POST", "/admin/action", values, adminCookie), 303)
		}
		balance("114.00")
		transaction("12006", "reversed", "42.00")
		if err := c.Wallet.RecoverFundingTransfer(ctx, admin, "12006"); err != nil {
			t.Fatal(err)
		}
		var review bool
		if err := db.QueryRow(`SELECT review_required FROM wallet_deposits WHERE reference='VA-12006'`).Scan(&review); err != nil || !review {
			t.Fatalf("reversal was not flagged: %v", err)
		}
		var audits int
		if err := db.QueryRow(`SELECT count(*) FROM admin_audit_logs WHERE action='wallet_funding_check' AND succeeded=true`).Scan(&audits); err != nil || audits != 2 {
			t.Fatalf("recovery audit missing: %d %v", audits, err)
		}
	})
}
