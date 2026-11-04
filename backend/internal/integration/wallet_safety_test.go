package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestTransactionPINEnrollmentAndConcurrentLockout(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	user := fixtureUser(t, c, 1, "buyer")
	pins := services.NewTransactionPINService(c.UserRepo)
	if err := pins.Verify(ctx, user.ID, "582491"); !errors.Is(err, repository.ErrTransactionPINRequired) {
		t.Fatalf("missing PIN: %v", err)
	}
	if err := pins.Setup(ctx, user, "incorrect", "582491", "582491"); err == nil {
		t.Fatal("enrollment bypassed account password")
	}
	if err := pins.Setup(ctx, user, "test password with spaces", "582491", "582492"); err == nil {
		t.Fatal("mismatched PIN accepted")
	}
	if err := pins.Setup(ctx, user, "test password with spaces", "582491", "582491"); err != nil {
		t.Fatal(err)
	}
	if err := pins.Setup(ctx, user, "test password with spaces", "123456", "123456"); !errors.Is(err, repository.ErrTransactionPINExists) {
		t.Fatalf("PIN overwritten: %v", err)
	}
	var hash string
	if err := db.QueryRow(`SELECT pin_hash FROM user_transaction_pins WHERE user_id=$1`, user.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == "582491" || bcrypt.CompareHashAndPassword([]byte(hash), []byte("582491")) != nil {
		t.Fatal("PIN not securely hashed")
	}
	if err := pins.Verify(ctx, user.ID, "582491"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pins.Verify(ctx, user.ID, "000000")
			if !errors.Is(err, repository.ErrTransactionPINInvalid) && !errors.Is(err, repository.ErrTransactionPINLocked) {
				t.Errorf("failed attempt: %v", err)
			}
		}()
	}
	wg.Wait()
	var attempts int
	if err := db.QueryRow(`SELECT failed_attempts FROM user_transaction_pins WHERE user_id=$1`, user.ID).Scan(&attempts); err != nil || attempts != 5 {
		t.Fatalf("atomic attempt count %d: %v", attempts, err)
	}
	if err := pins.Verify(ctx, user.ID, "582491"); !errors.Is(err, repository.ErrTransactionPINLocked) {
		t.Fatalf("lock bypass: %v", err)
	}
	if _, err := db.Exec(`UPDATE user_transaction_pins SET blocked_until=CURRENT_TIMESTAMP-INTERVAL '1 minute' WHERE user_id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := pins.Verify(ctx, user.ID, "582491"); err != nil {
		t.Fatalf("expired lock: %v", err)
	}
	handler := routes.NewHandler(c)
	page := request(handler, "GET", "/wallet", nil, session(t, c, user))
	requireStatus(t, page, 200)
	if strings.Contains(page.Body.String(), hash) || strings.Contains(page.Body.String(), "582491") {
		t.Fatal("PIN leaked into wallet page")
	}
}

func TestWalletConcurrentRequestsReserveAndRefundOnce(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	user := fixtureUser(t, c, 1, "buyer")
	recipient := fixtureUser(t, c, 2, "buyer")
	repo := repository.NewWalletRepository(db, "NGN", 0)
	target, err := repo.Wallet(ctx, recipient.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Wallet(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	seedAccountingWallet(t, db, user.ID, 100000)
	parallel := func(run func() error) {
		t.Helper()
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := run(); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
	}
	parallel(func() error {
		return repo.RequestTransfer(ctx, user.ID, target.WalletNumber, "100.00", "repeated-transfer", "Seeds")
	})
	if err := repo.RequestTransfer(ctx, user.ID, target.WalletNumber, "101.00", "repeated-transfer", "Seeds"); !errors.Is(err, repository.ErrWalletRequestChanged) {
		t.Fatalf("changed transfer accepted: %v", err)
	}
	var transferID int64
	if err = db.QueryRow(`SELECT id FROM wallet_transfers WHERE reference='repeated-transfer'`).Scan(&transferID); err != nil {
		t.Fatal(err)
	}
	if err = repo.DecideTransfer(ctx, user.ID, false, transferID, "cancelled"); err != nil {
		t.Fatal(err)
	}
	parallel(func() error {
		return repo.RequestTransfer(ctx, user.ID, target.WalletNumber, "100.00", "repeated-transfer", "Seeds")
	})
	account := &models.UserBankAccount{BankName: "Test bank", BankCode: "001", AccountName: user.FullName, AccountNumber: "0123456789"}
	auth := fixtureEvidence(t, db, user.ID, "repeated-withdrawal")
	parallel(func() error {
		return repo.RequestWithdrawal(ctx, user.ID, "250.00", "repeated-withdrawal", account, nil, auth)
	})
	if err = repo.RequestWithdrawal(ctx, user.ID, "250.00", "another-key", account, nil, auth); err == nil {
		t.Fatal("second outstanding withdrawal accepted")
	}
	if err = repo.RequestWithdrawal(ctx, user.ID, "251.00", "repeated-withdrawal", account, nil, auth); !errors.Is(err, repository.ErrWalletRequestChanged) {
		t.Fatalf("changed amount accepted: %v", err)
	}
	x, err := repo.WithdrawalByReference(ctx, "repeated-withdrawal")
	if err != nil {
		t.Fatal(err)
	}

	admin := fixtureUser(t, c, 9, "buyer")
	if _, err := db.Exec(`UPDATE users SET is_admin=true WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	adminCookie := session(t, c, admin)
	fixtureReviewEvidence(t, c, x.ID, admin.ID, adminCookie.Value)
	var claims atomic.Int32
	parallel(func() error {
		if repo.ClaimWithdrawal(ctx, x.ID, admin.ID, services.SecurityHash(adminCookie.Value)) == nil {
			claims.Add(1)
		}
		return nil
	})
	if claims.Load() != 1 {
		t.Fatalf("provider submission claims: %d", claims.Load())
	}
	if err = repo.FinishWithdrawal(ctx, x.ID, user.ID, false, "cancelled", false); err == nil {
		t.Fatal("uncertain payout refunded")
	}
	if err = repo.SaveTransferID(ctx, x.ID, "9001"); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveTransferID(ctx, x.ID, "9002"); err == nil {
		t.Fatal("provider identity changed")
	}
	parallel(func() error { return repo.FinishWithdrawal(ctx, x.ID, 0, false, "failed", true) })
	var balance string
	var refunds, reservations int
	if err = db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, user.ID).Scan(&balance); err != nil || balance != "1000.00" {
		t.Fatalf("balance %s: %v", balance, err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FILTER(WHERE type='withdrawal_refund'),COUNT(*) FILTER(WHERE type='withdrawal_reserved') FROM wallet_transactions WHERE reference LIKE 'repeated-withdrawal:%'`).Scan(&refunds, &reservations); err != nil || refunds != 1 || reservations != 1 {
		t.Fatalf("refunds=%d reservations=%d: %v", refunds, reservations, err)
	}
}

func TestConcurrentCheckoutAndCrossDomainReceipt(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transactions/fee" {
			fmt.Fprintf(w, `{"status":"success","data":{"fee":0,"charge_amount":%s,"currency":"NGN"}}`, r.URL.Query().Get("amount"))
			return
		}
		calls.Add(1)
		fmt.Fprint(w, `{"status":"success","data":{"link":"https://checkout.example.test/session"}}`)
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.FlutterwaveSecretKey = "test-key"
	cfg.FlutterwaveBaseURL = provider.URL
	c := application(t, db, cfg)
	buyer := fixtureUser(t, c, 1, "buyer")
	seller := fixtureUser(t, c, 2, "farmer")
	product := fixtureProduct(t, db, seller.ID, "Seed", 10, 100)
	if err := c.Order.PlaceOrder(buyer.ID, product, 1, "receipt-test-order"); err != nil {
		t.Fatal(err)
	}
	var orderID int
	if err := db.QueryRow(`SELECT id FROM orders WHERE checkout_key='receipt-test-order'`).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = c.Payment.Initiate(ctx, buyer.ID, orderID) }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("order initialized %d times", calls.Load())
	}
	payRepo := repository.NewPaymentRepository(db)
	var ref string
	if err := db.QueryRow(`SELECT payment_reference FROM transactions WHERE order_id=$1`, orderID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	walletRepo := repository.NewWalletRepository(db, "NGN", 0)
	q, err := c.Wallet.Quote(ctx, buyer, "deposit", "100", "banktransfer")
	if err != nil {
		t.Fatal(err)
	}
	depositRef := fmt.Sprintf("WALLET-%d-%s", buyer.ID, q.ID)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = c.Wallet.StartDeposit(ctx, buyer, "100", q.ID) }()
	}
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("funding initialized more than once: %d calls", calls.Load())
	}
	if _, err := c.Wallet.StartDeposit(ctx, buyer, "100", strings.Repeat("b", 32)); err == nil {
		t.Fatal("page reload started another pending funding request")
	}
	if _, err := c.Wallet.StartDeposit(ctx, buyer, "101", q.ID); !errors.Is(err, repository.ErrWalletRequestChanged) {
		t.Fatalf("changed funding amount accepted: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatal("rejected funding request reached provider")
	}

	if _, err := walletRepo.FinalizeDeposit(ctx, depositRef, "8001", "150.00", "NGN", "successful", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := payRepo.FinalizePayment(ref, "successful", "8001", nil); err == nil {
		t.Fatal("provider receipt paid an order after funding a wallet")
	}
	if _, err := payRepo.FinalizePayment(ref, "successful", "8002", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := walletRepo.CreateDeposit(ctx, buyer.ID, "separate-deposit", "100.00"); err != nil {
		t.Fatal(err)
	}
	if _, err := walletRepo.FinalizeDeposit(ctx, "separate-deposit", "8002", "100.00", "NGN", "successful", nil); err == nil {
		t.Fatal("order receipt reused for wallet funding")
	}
	handler := routes.NewHandler(c)
	response := request(handler, "POST", "/wallet", url.Values{"action": {"pay_order"}, "order_id": {fmt.Sprint(orderID)}}, session(t, c, buyer))
	requireStatus(t, response, 400)
}
