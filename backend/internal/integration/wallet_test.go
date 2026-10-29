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

	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
)

func TestWalletMoneyFlowsWithPostgresAndProvider(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	var mu sync.Mutex
	depositReference, transferReference := "", ""
	transferAmount := "10.00"
	transferStatus := "successful"
	transferRequests := 0
	ambiguousTransfer := false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer test-wallet-key" {
			t.Error("missing wallet provider credential")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/payments":
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			json.Unmarshal(payload["tx_ref"], &depositReference)
			var amount string
			json.Unmarshal(payload["amount"], &amount)
			if amount != "1000.00" {
				t.Errorf("wrong funding amount %s", amount)
			}
			fmt.Fprint(w, `{"status":"success","data":{"link":"https://checkout.example.test/wallet"}}`)
		case r.URL.Path == "/transactions/8001/verify":
			json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"id": 8001, "tx_ref": depositReference, "status": "successful", "amount": 1000, "currency": "NGN"}})
		case r.URL.Path == "/banks/NG":
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Test bank","code":"001"}]}`)
		case r.URL.Path == "/accounts/resolve":
			fmt.Fprint(w, `{"status":"success","data":{"account_name":"Test User 1","account_number":"0123456789"}}`)
		case r.Method == "POST" && r.URL.Path == "/transfers":
			transferRequests++
			var payload struct {
				Reference string      `json:"reference"`
				Amount    json.Number `json:"amount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			transferReference, transferAmount = payload.Reference, payload.Amount.String()
			if ambiguousTransfer {
				w.WriteHeader(500)
				fmt.Fprint(w, `{"status":"error"}`)
				return
			}
			fmt.Fprint(w, `{"status":"success","data":{"id":9001}}`)
		case strings.HasPrefix(r.URL.Path, "/transfers/"):
			id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/transfers/"))
			json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"id": id, "reference": transferReference, "amount": json.Number(transferAmount), "currency": "NGN", "status": transferStatus}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.CommissionPercent = "2.5"
	cfg.FlutterwaveSecretKey = "test-wallet-key"
	cfg.FlutterwaveSecretHash = "test-wallet-hash"
	cfg.FlutterwaveBaseURL = provider.URL
	cfg.PaymentRedirectURL = "http://example.test/payments/callback"
	c := application(t, db, cfg)
	sender := fixtureUser(t, c, 1, "buyer")
	recipient := fixtureUser(t, c, 2, "buyer")
	seller := fixtureUser(t, c, 3, "farmer")
	admin := fixtureUser(t, c, 4, "farmer")
	if _, err := db.Exec(`UPDATE users SET is_admin=TRUE WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	admin.IsAdmin = true
	capture := &captureOTP{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), capture)
	otp.Configure(cfg.OTPTTL, cfg.OTPMaxAttempts, 0)
	c.Wallet.SetOTP(otp)
	handler := routes.NewHandler(c)
	cookie := session(t, c, sender)
	deposit, err := c.Wallet.StartDeposit(ctx, sender, "1000", "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Payment.ConfirmCallback(ctx, "8001", deposit.Reference); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = c.Payment.ConfirmCallback(ctx, "8001", deposit.Reference); err != nil {
			t.Fatal(err)
		}
	}
	balance := func(id int, want string) {
		t.Helper()
		var got string
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, id).Scan(&got); err != nil || got != want {
			t.Fatalf("wallet %d has %s; want %s (%v)", id, got, want, err)
		}
	}
	balance(sender.ID, "1000.00")
	recipientPage, err := c.Wallet.Page(ctx, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.RequestTransfer(ctx, sender, recipientPage.Wallet.WalletNumber, "100", "", "wrong"); err == nil {
		t.Fatal("wallet transfer accepted without a code")
	}
	if err = c.Wallet.SendCode(ctx, sender, "transfer"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.RequestTransfer(ctx, sender, recipientPage.Wallet.WalletNumber, "100", "Seed purchase", capture.message.Code); err != nil {
		t.Fatal(err)
	}
	balance(sender.ID, "900.00")
	balance(recipient.ID, "0.00")
	page, err := c.Wallet.Page(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	transferID := page.Transfers[0].ID
	if err = c.Wallet.DecideTransfer(ctx, recipient, transferID, "completed"); err == nil {
		t.Fatal("recipient approved own transfer")
	}
	if err = c.Wallet.DecideTransfer(ctx, admin, transferID, "completed"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.DecideTransfer(ctx, admin, transferID, "completed"); err == nil {
		t.Fatal("transfer completed twice")
	}
	balance(recipient.ID, "100.00")
	if err = c.Wallet.SendCode(ctx, sender, "transfer"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.RequestTransfer(ctx, sender, recipientPage.Wallet.WalletNumber, "50", "", capture.message.Code); err != nil {
		t.Fatal(err)
	}
	page, err = c.Wallet.Page(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	transferID = page.Transfers[0].ID
	if err = c.Wallet.DecideTransfer(ctx, recipient, transferID, "cancelled"); err == nil {
		t.Fatal("outsider cancelled a transfer")
	}
	if err = c.Wallet.DecideTransfer(ctx, sender, transferID, "cancelled"); err != nil {
		t.Fatal(err)
	}
	balance(sender.ID, "900.00")
	if err = c.Wallet.SaveBank(ctx, sender.ID, "001", "0123456789"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.SendCode(ctx, sender, "withdrawal"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.RequestWithdrawal(ctx, sender, "10", capture.message.Code); err != nil {
		t.Fatal(err)
	}
	page, err = c.Wallet.Page(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	withdrawalID := page.Withdrawals[0].ID
	if err = c.Wallet.DecideWithdrawal(ctx, recipient, withdrawalID, "approve"); err == nil {
		t.Fatal("outsider approved withdrawal")
	}
	if err = c.Wallet.DecideWithdrawal(ctx, admin, withdrawalID, "approve"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.DecideWithdrawal(ctx, admin, withdrawalID, "approve"); err == nil {
		t.Fatal("paid withdrawal sent twice")
	}
	balance(sender.ID, "890.00")
	mu.Lock()
	requests := transferRequests
	ambiguousTransfer = true
	transferStatus = "failed"
	mu.Unlock()
	if requests != 1 {
		t.Fatalf("provider received %d transfers", requests)
	}
	if err = c.Wallet.SendCode(ctx, sender, "withdrawal"); err != nil {
		t.Fatal(err)
	}
	if err = c.Wallet.RequestWithdrawal(ctx, sender, "20", capture.message.Code); err != nil {
		t.Fatal(err)
	}
	page, err = c.Wallet.Page(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	withdrawalID = page.Withdrawals[0].ID
	if err = c.Wallet.DecideWithdrawal(ctx, admin, withdrawalID, "approve"); err == nil {
		t.Fatal("provider failure claimed success")
	}
	if err = c.Wallet.DecideWithdrawal(ctx, sender, withdrawalID, "cancelled"); err == nil {
		t.Fatal("ambiguous payout was refunded without verification")
	}
	balance(sender.ID, "870.00")
	mu.Lock()
	ref := transferReference
	mu.Unlock()
	body := []byte(fmt.Sprintf(`{"event":"transfer.completed","data":{"id":9002,"reference":%q}}`, ref))
	if err = c.Payment.ProcessWebhook(ctx, body, cfg.FlutterwaveSecretHash); err != nil {
		t.Fatal(err)
	}
	if err = c.Payment.ProcessWebhook(ctx, body, cfg.FlutterwaveSecretHash); err != nil {
		t.Fatal(err)
	}
	balance(sender.ID, "890.00")
	p := fixtureProduct(t, db, seller.ID, "Wallet purchase", 10, 100)
	if err = c.Order.PlaceOrder(sender.ID, p, 2, "wallet-order-test"); err != nil {
		t.Fatal(err)
	}
	var orderID int
	if err = db.QueryRow(`SELECT id FROM orders WHERE checkout_key='wallet-order-test'`).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "POST", "/wallet", url.Values{"action": {"pay_order"}, "order_id": {strconv.Itoa(orderID)}}, cookie), http.StatusSeeOther)
	balance(sender.ID, "690.00")
	balance(seller.ID, "195.00")
	var fee string
	if err = db.QueryRow(`SELECT amount::text FROM platform_revenue WHERE order_id=$1`, orderID).Scan(&fee); err != nil || fee != "5.00" {
		t.Fatalf("configured fee was not applied: %s %v", fee, err)
	}
	if err = c.Wallet.PayOrder(ctx, sender.ID, orderID); err == nil {
		t.Fatal("wallet paid same order twice")
	}
	for _, path := range []string{"/wallet", "/wallet/banks"} {
		w := request(handler, "GET", path, nil, cookie)
		requireStatus(t, w, http.StatusOK)
		if w.Body.Len() == 0 {
			t.Fatal("empty wallet response")
		}
	}
	requireStatus(t, request(handler, "GET", "/wallet/admin", nil, cookie), http.StatusForbidden)
	requireStatus(t, request(handler, "GET", "/wallet/admin", nil, session(t, c, admin)), http.StatusSeeOther)
}
