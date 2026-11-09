package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

type walletCodeDelivery struct {
	messages []services.OTPMessage
	fail     bool
}

func (d *walletCodeDelivery) SendOTP(_ context.Context, m services.OTPMessage) error {
	if d.fail {
		return fmt.Errorf("test delivery failure")
	}
	d.messages = append(d.messages, m)
	return nil
}
func confirmationCapture(t *testing.T, handler http.Handler, cookie *http.Cookie, token string, overrides map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{"token": token, "latitude": "7.7", "longitude": "8.5", "accuracy": "12", "captured_at": time.Now().Format(time.RFC3339Nano)}
	for name, value := range overrides {
		fields[name] = value
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.CreateFormFile("selfie", "selfie.jpg")
	if err != nil {
		t.Fatal(err)
	}
	file.Write(verificationJPEG(t))
	writer.Close()
	req := httptest.NewRequest("POST", "http://example.test/wallet/confirmation/complete", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.test")
	req.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}
func confirmationToken(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	requireStatus(t, response, http.StatusOK)
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || len(result.Token) != 64 {
		t.Fatalf("invalid confirmation: %s %v", response.Body.String(), err)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("confirmation must not be cached")
	}
	return result.Token
}
func TestWalletConfirmationPINOTPAndImmediateTransfer(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	sender := fixtureUser(t, c, 1, "buyer")
	payee := fixtureUser(t, c, 2, "farmer")
	stranger := fixtureUser(t, c, 3, "buyer")
	hash, _ := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
	if err := c.UserRepo.CreateTransactionPIN(t.Context(), sender.ID, string(hash)); err != nil {
		t.Fatal(err)
	}
	seedAccountingWallet(t, db, sender.ID, 100000)
	delivery := &walletCodeDelivery{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), delivery)
	otp.Configure(10*time.Minute, 5, 0)
	c.Wallet.SetOTP(otp)
	handler := routes.NewHandler(c)
	cookie := session(t, c, sender)
	otherSession := session(t, c, sender)
	strangerCookie := session(t, c, stranger)
	form := url.Values{"kind": {"transfer"}, "recipient": {payee.Phone}, "amount": {"25.50"}, "note": {"Seeds"}, "request_key": {strings.Repeat("a", 32)}, "transaction_pin": {"000000"}}
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie), http.StatusBadRequest)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM wallet_confirmations`).Scan(&count)
	if count != 0 || len(delivery.messages) != 0 {
		t.Fatal("wrong PIN created or delivered OTP")
	}
	form.Set("transaction_pin", "582491")
	token := confirmationToken(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie))
	code := delivery.messages[0].Code
	var otpHash, sessionHash string
	db.QueryRow(`SELECT otp_hash,session_hash FROM wallet_confirmations WHERE token_hash=$1`, services.SecurityHash(token)).Scan(&otpHash, &sessionHash)
	if otpHash == code || bcrypt.CompareHashAndPassword([]byte(otpHash), []byte(code)) != nil || sessionHash == cookie.Value {
		t.Fatal("code or session stored insecurely")
	}
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusBadRequest)
	badCode := "000000"
	if code == badCode {
		badCode = "111111"
	}
	verify := url.Values{"token": {token}, "code": {badCode}}
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", verify, cookie), http.StatusBadRequest)
	verify.Set("code", code)
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", verify, otherSession), http.StatusBadRequest)
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", verify, strangerCookie), http.StatusBadRequest)
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", verify, cookie), http.StatusOK)
	requireStatus(t, confirmationCapture(t, handler, otherSession, token, nil), http.StatusBadRequest)
	requireStatus(t, confirmationCapture(t, handler, cookie, token, map[string]string{"latitude": "NaN"}), http.StatusBadRequest)
	requireStatus(t, confirmationCapture(t, handler, cookie, token, map[string]string{"captured_at": time.Now().Add(-5 * time.Minute).Format(time.RFC3339Nano)}), http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := confirmationCapture(t, handler, cookie, token, map[string]string{"recipient": strconv.Itoa(stranger.ID), "amount": "900.00"})
			if response.Code != http.StatusOK {
				t.Errorf("repeat completion: %s", response.Body.String())
			}
		}()
	}
	wg.Wait()
	for _, tc := range []struct {
		id   int
		want string
	}{{sender.ID, "974.50"}, {payee.ID, "25.50"}, {stranger.ID, "0.00"}} {
		var balance string
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, tc.id).Scan(&balance); err != nil || balance != tc.want {
			t.Fatalf("balance %d %s expected %s (%v)", tc.id, balance, tc.want, err)
		}
	}
	var status string
	var admin int
	if err := db.QueryRow(`SELECT status,COALESCE(approved_by,0) FROM wallet_transfers WHERE sender_id=$1`, sender.ID).Scan(&status, &admin); err != nil || status != "completed" || admin != 0 {
		t.Fatalf("transfer still needs an admin: %s %v", status, err)
	}
	var photos, entries int
	db.QueryRow(`SELECT COUNT(*) FROM private_verification_photos WHERE purpose='transfer'`).Scan(&photos)
	db.QueryRow(`SELECT COUNT(*) FROM wallet_transactions WHERE reference LIKE 'TRANSFER-%'`).Scan(&entries)
	if photos != 1 || entries != 2 {
		t.Fatalf("retries duplicated evidence or money: %d %d", photos, entries)
	}
	var encrypted []byte
	var photoID int64
	db.QueryRow(`SELECT p.id,p.encrypted_image FROM private_verification_photos p JOIN wallet_confirmations c ON c.photo_id=p.id WHERE c.token_hash=$1`, services.SecurityHash(token)).Scan(&photoID, &encrypted)
	if _, err := c.Security.Open(encrypted, "verification-photo"); err != nil {
		t.Fatal("confirmation photo is not encrypted", err)
	}
	requireStatus(t, request(handler, "GET", "/private/verification/"+strconv.FormatInt(photoID, 10), nil, cookie), http.StatusOK)
	requireStatus(t, request(handler, "GET", "/private/verification/"+strconv.FormatInt(photoID, 10), nil, strangerCookie), http.StatusNotFound)
	requireStatus(t, request(handler, "POST", "/wallet", url.Values{"action": {"transfer"}, "recipient": {payee.Phone}, "amount": {"1.00"}}, cookie), http.StatusBadRequest)
	requireStatus(t, request(handler, "POST", "/wallet", url.Values{"action": {"send_code"}, "purpose": {"transfer"}}, cookie), http.StatusBadRequest)
	if len(delivery.messages) != 1 {
		t.Fatal("legacy endpoint bypassed PIN-first delivery")
	}
	// Completed receipt remains replayable after expiry, without moving money.
	db.Exec(`UPDATE wallet_confirmations SET expires_at=CURRENT_TIMESTAMP-interval '1 hour' WHERE token_hash=$1`, services.SecurityHash(token))
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusOK)
}

func TestWalletConfirmationExpiryAttemptsAndDeliveryFailure(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	user := fixtureUser(t, c, 1, "buyer")
	payee := fixtureUser(t, c, 2, "farmer")
	hash, _ := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
	c.UserRepo.CreateTransactionPIN(t.Context(), user.ID, string(hash))
	seedAccountingWallet(t, db, user.ID, 100000)
	delivery := &walletCodeDelivery{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), delivery)
	otp.Configure(10*time.Minute, 5, 0)
	c.Wallet.SetOTP(otp)
	handler := routes.NewHandler(c)
	cookie := session(t, c, user)
	form := url.Values{"kind": {"transfer"}, "recipient": {strconv.Itoa(payee.ID)}, "amount": {"10.00"}, "request_key": {strings.Repeat("b", 32)}, "transaction_pin": {"582491"}}
	token := confirmationToken(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie))
	form.Set("amount", "11.00")
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie), http.StatusBadRequest)
	form.Set("amount", "10.00")
	if len(delivery.messages) != 1 {
		t.Fatal("changed amount received another code")
	}
	badCode := "000000"
	if delivery.messages[0].Code == badCode {
		badCode = "111111"
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {token}, "code": {badCode}}, cookie)
			if response.Code != http.StatusBadRequest {
				t.Errorf("wrong code accepted: %s", response.Body.String())
			}
		}()
	}
	wg.Wait()
	var attempts int
	db.QueryRow(`SELECT attempts FROM wallet_confirmations WHERE token_hash=$1`, services.SecurityHash(token)).Scan(&attempts)
	if attempts != 5 {
		t.Fatalf("concurrent guesses bypassed attempt limit: %d", attempts)
	}
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {token}, "code": {delivery.messages[0].Code}}, cookie), http.StatusBadRequest)
	form.Set("request_key", strings.Repeat("c", 32))
	next := confirmationToken(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie))
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {token}, "code": {delivery.messages[0].Code}}, cookie), http.StatusBadRequest)
	db.Exec(`UPDATE wallet_confirmations SET expires_at=CURRENT_TIMESTAMP-interval '1 second' WHERE token_hash=$1`, services.SecurityHash(next))
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {next}, "code": {delivery.messages[1].Code}}, cookie), http.StatusBadRequest)
	delivery.fail = true
	form.Set("request_key", strings.Repeat("d", 32))
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie), http.StatusBadRequest)
	var usable int
	db.QueryRow(`SELECT COUNT(*) FROM wallet_confirmations WHERE invalidated_at IS NULL AND expires_at>CURRENT_TIMESTAMP`).Scan(&usable)
	if usable != 0 {
		t.Fatal("failed delivery left a usable confirmation")
	}
}

func TestWalletConfirmationWithdrawalWaitsForAdmin(t *testing.T) {
	db, _ := testutil.Database(t, true)
	payouts := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/banks/NG":
			fmt.Fprint(w, `{"status":"success","data":[{"name":"Test Bank","code":"001"}]}`)
		case "/accounts/resolve":
			fmt.Fprint(w, `{"status":"success","data":{"account_name":"Test User 1","account_number":"0123456789"}}`)
		case "/transfers/fee":
			fmt.Fprint(w, `{"status":"success","data":[{"fee_type":"value","currency":"NGN","fee":10}]}`)
		case "/transfers":
			payouts++
			http.Error(w, "should wait for admin", 500)
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.FlutterwaveSecretKey = "test-key"
	cfg.FlutterwaveBaseURL = provider.URL
	c := application(t, db, cfg)
	user := fixtureUser(t, c, 1, "buyer")
	hash, _ := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
	c.UserRepo.CreateTransactionPIN(t.Context(), user.ID, string(hash))
	seedAccountingWallet(t, db, user.ID, 100000)
	if err := repository.NewPaymentRepository(db, 0).UpsertBankAccount(&models.UserBankAccount{UserID: user.ID, BankName: "Test Bank", BankCode: "001", AccountName: user.FullName, AccountNumber: "0123456789"}); err != nil {
		t.Fatal(err)
	}
	delivery := &walletCodeDelivery{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), delivery)
	otp.Configure(10*time.Minute, 5, 0)
	c.Wallet.SetOTP(otp)
	handler := routes.NewHandler(c)
	cookie := session(t, c, user)
	quote, err := c.Wallet.Quote(t.Context(), user, "withdrawal", "100.00", "")
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"kind": {"withdrawal"}, "quote": {quote.ID}, "amount": {quote.Amount()}, "request_key": {quote.ID}, "transaction_pin": {"582491"}}
	token := confirmationToken(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie))
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusBadRequest)
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {token}, "code": {delivery.messages[0].Code}}, cookie), http.StatusOK)
	db.Exec(`UPDATE user_bank_accounts SET account_number='9999999999' WHERE user_id=$1`, user.ID)
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusBadRequest)
	db.Exec(`UPDATE user_bank_accounts SET account_number='0123456789' WHERE user_id=$1`, user.ID)
	response := confirmationCapture(t, handler, cookie, token, nil)
	requireStatus(t, response, http.StatusOK)
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusOK)
	var balance, status string
	var evidence int
	db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, user.ID).Scan(&balance)
	db.QueryRow(`SELECT status FROM wallet_withdrawals WHERE user_id=$1`, user.ID).Scan(&status)
	db.QueryRow(`SELECT COUNT(*) FROM withdrawal_evidence e JOIN wallet_withdrawals w ON w.id=e.withdrawal_id WHERE w.user_id=$1 AND e.quote_id=$2 AND e.photo_id IS NOT NULL`, user.ID, quote.ID).Scan(&evidence)
	if balance != money.Format(100000-quote.WalletDebitKobo) || status != "pending" || evidence != 1 || payouts != 0 {
		t.Fatalf("withdrawal did not reserve for review: balance=%s status=%s evidence=%d payouts=%d", balance, status, evidence, payouts)
	}
}

func TestWalletConfirmationRechecksRecipientAndBalance(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	user := fixtureUser(t, c, 1, "buyer")
	payee := fixtureUser(t, c, 2, "farmer")
	hash, _ := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
	if err := c.UserRepo.CreateTransactionPIN(t.Context(), user.ID, string(hash)); err != nil {
		t.Fatal(err)
	}
	seedAccountingWallet(t, db, user.ID, 100000)
	delivery := &walletCodeDelivery{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), delivery)
	otp.Configure(10*time.Minute, 5, 0)
	c.Wallet.SetOTP(otp)
	handler := routes.NewHandler(c)
	cookie := session(t, c, user)
	form := url.Values{"kind": {"transfer"}, "recipient": {strconv.Itoa(payee.ID)}, "amount": {"1000.00"}, "request_key": {strings.Repeat("e", 32)}, "transaction_pin": {"582491"}}
	token := confirmationToken(t, request(handler, "POST", "/wallet/confirmation/start", form, cookie))
	requireStatus(t, request(handler, "POST", "/wallet/confirmation/verify", url.Values{"token": {token}, "code": {delivery.messages[0].Code}}, cookie), http.StatusOK)
	if _, err := db.Exec(`UPDATE users SET account_status='suspended' WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusBadRequest)
	if _, err := db.Exec(`UPDATE users SET account_status='active' WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewWalletRepository(db, "NGN", 0)
	wallet, err := repo.Wallet(t.Context(), payee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.RequestTransfer(t.Context(), user.ID, wallet.WalletNumber, "100.00", "legacy-reservation", ""); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, confirmationCapture(t, handler, cookie, token, nil), http.StatusBadRequest)
	var photos, receipts int
	if err = db.QueryRow(`SELECT COUNT(*) FROM private_verification_photos WHERE purpose='transfer'`).Scan(&photos); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT COUNT(*) FROM wallet_confirmations WHERE final_reference IS NOT NULL`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if photos != 0 || receipts != 0 {
		t.Fatal("failed transfer retained evidence or a successful receipt")
	}
	var balance string
	if err = db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, user.ID).Scan(&balance); err != nil || balance != "900.00" {
		t.Fatalf("failed confirmation changed the balance: %s %v", balance, err)
	}
}
