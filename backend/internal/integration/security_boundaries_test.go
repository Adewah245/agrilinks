package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testui"
	"backend/internal/testutil"
)

func TestAdminMFAProtectsPrivilegedMemberRoutes(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	admin := fixtureUser(t, c, 1, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	token, err := c.Sessions.Create(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "agroshield_session", Value: token}
	handler := routes.NewHandler(c)
	for _, path := range []string{"/trust", "/deliveries", "/delivery-partner", "/orders/protection?order_id=1"} {
		t.Run("GET "+path, func(t *testing.T) {
			response := request(handler, "GET", path, nil, cookie)
			requireStatus(t, response, http.StatusSeeOther)
			if response.Header().Get("Location") != "/admin/security" {
				t.Fatal("privileged page did not require MFA")
			}
		})
	}
	for _, path := range []string{"/trust", "/delivery-partner/review", "/deliveries/assign", "/deliveries/status", "/deliveries/tracking", "/orders/protection"} {
		t.Run("POST "+path, func(t *testing.T) {
			form := url.Values{"order_id": {"1"}, "action": {"refund"}, "reason": {"Security regression check"}, "csrf": {services.SecurityHash("agro-shield:admin-actions:" + token)}}
			requireStatus(t, request(handler, "POST", path, form, cookie), http.StatusForbidden)
		})
	}

	// Ordinary members still use their delivery pages without staff enrollment.
	member := fixtureUser(t, c, 2, "buyer")
	memberCookie := session(t, c, member)
	for _, path := range []string{"/deliveries", "/delivery-partner"} {
		requireStatus(t, request(handler, "GET", path, nil, memberCookie), http.StatusOK)
	}

	authorizeFixtureAdmin(t, c, admin.ID, token)
	requireStatus(t, request(handler, "GET", "/trust", nil, cookie), http.StatusOK)
	if _, err := db.Exec(`UPDATE user_sessions SET mfa_verified_until=CURRENT_TIMESTAMP-interval '1 second' WHERE token_hash=$1`, services.SecurityHash(token)); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "POST", "/trust", nil, cookie), http.StatusForbidden)
}

func TestWithdrawalApprovalRequiresBothPrivatePhotosAndCurrentRole(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	owner := fixtureUser(t, c, 1, "buyer")
	admin := fixtureUser(t, c, 2, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	adminCookie := session(t, c, admin)
	repo := repository.NewWalletRepository(db, "NGN", 0, c.UserRepo)
	if _, err := repo.Wallet(ctx, owner.ID); err != nil {
		t.Fatal(err)
	}
	seedAccountingWallet(t, db, owner.ID, 100000)
	privateImage, err := c.Security.PrivateImage(verificationJPEG(t))
	if err != nil {
		t.Fatal(err)
	}
	var registrationID int64
	if err = db.QueryRow(`INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'registration',$2) RETURNING id`, owner.ID, privateImage).Scan(&registrationID); err != nil {
		t.Fatal(err)
	}
	const reference = "review-both-photos"
	auth := fixtureEvidence(t, db, owner.ID, reference)
	if _, err = db.Exec(`UPDATE private_verification_photos SET encrypted_image=$2 WHERE id=(SELECT photo_id FROM withdrawal_evidence WHERE reference=$1)`, reference, privateImage); err != nil {
		t.Fatal(err)
	}
	account := &models.UserBankAccount{BankName: "Test bank", BankCode: "001", AccountName: owner.FullName, AccountNumber: "0123456789"}
	if err = repo.RequestWithdrawal(ctx, owner.ID, "250.00", reference, account, nil, auth); err != nil {
		t.Fatal(err)
	}
	withdrawal, err := repo.WithdrawalByReference(ctx, reference)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := c.Security.Repo.Evidence(ctx, withdrawal.ID)
	if err != nil {
		t.Fatal(err)
	}
	handler := routes.NewHandler(c)
	reviewPath := fmt.Sprintf("/admin/withdrawals/%d/evidence", withdrawal.ID)
	photoPath := func(id int64) string { return "/private/verification/" + strconv.FormatInt(id, 10) }
	const changedPublicPhoto = "/static/uploads/profiles/changed-profile.jpg"
	if err = c.Auth.UpdatePhoto(owner.ID, changedPublicPhoto); err != nil {
		t.Fatal(err)
	}
	review := request(handler, "GET", reviewPath, nil, adminCookie)
	requireStatus(t, review, http.StatusOK)
	reviewHTML := testui.HTML(t, review.Body.String())
	if !strings.Contains(reviewHTML, photoPath(registrationID)) || !strings.Contains(reviewHTML, "Original registration selfie") || strings.Contains(reviewHTML, changedPublicPhoto) {
		t.Fatal("withdrawal comparison must use the preserved registration original after a public profile edit")
	}
	requireStatus(t, request(handler, "GET", photoPath(evidence.PhotoID), nil, adminCookie), http.StatusOK)
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("approval skipped the private account verification photograph")
	}

	// A different session reading the account photograph cannot authorize this review.
	otherSession := session(t, c, admin)
	requireStatus(t, request(handler, "GET", photoPath(registrationID), nil, otherSession), http.StatusOK)
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("photo access crossed review sessions")
	}
	requireStatus(t, request(handler, "GET", photoPath(registrationID), nil, adminCookie), http.StatusOK)

	// A new review must load its photographs again, even within the MFA window.
	requireStatus(t, request(handler, "GET", reviewPath, nil, adminCookie), http.StatusOK)
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("new review reused earlier photo access")
	}
	requireStatus(t, request(handler, "GET", photoPath(evidence.PhotoID), nil, adminCookie), http.StatusOK)
	requireStatus(t, request(handler, "GET", photoPath(registrationID), nil, adminCookie), http.StatusOK)
	var replacementID int64
	if err = db.QueryRow(`INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'registration',$2) RETURNING id`, owner.ID, privateImage).Scan(&replacementID); err != nil {
		t.Fatal(err)
	}
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("approval used an outdated account photograph")
	}
	requireStatus(t, request(handler, "GET", reviewPath, nil, adminCookie), http.StatusOK)
	requireStatus(t, request(handler, "GET", photoPath(evidence.PhotoID), nil, adminCookie), http.StatusOK)
	requireStatus(t, request(handler, "GET", photoPath(registrationID), nil, adminCookie), http.StatusOK)
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("opening the old photograph satisfied a new review")
	}
	requireStatus(t, request(handler, "GET", photoPath(replacementID), nil, adminCookie), http.StatusOK)
	c.UserRepo.SetConfiguredAdministrators([]int{admin.ID})
	if _, err = db.Exec(`UPDATE users SET staff_role='customer_care' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("revoked finance permission still claimed a payout")
	}
	// Legacy configuration grants still work, but cannot override an explicit
	// lower staff role or disabled account.
	if _, err = db.Exec(`UPDATE users SET staff_role='' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err != nil {
		t.Fatal(err)
	}
	if err = repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(adminCookie.Value)); err == nil {
		t.Fatal("review claimed a payout twice")
	}
}

func TestEscrowRefundRequiresFreshAdminMFA(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	buyer := fixtureUser(t, c, 1, "buyer")
	seller := fixtureUser(t, c, 2, "farmer")
	admin := fixtureUser(t, c, 3, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	product := fixtureProduct(t, db, seller.ID, "MFA protected order", 10, 100)
	if err := c.Order.PlaceOrder(buyer.ID, product, 2, "mfa-refund-order"); err != nil {
		t.Fatal(err)
	}
	var orderID int
	if err := db.QueryRow(`SELECT id FROM orders WHERE checkout_key='mfa-refund-order'`).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	payments := repository.NewPaymentRepository(db, 250)
	if _, err := payments.CreatePendingForOrder(buyer.ID, orderID, "MFA-REFUND", "NGN", "flutterwave"); err != nil {
		t.Fatal(err)
	}
	zero := int64(0)
	if _, err := payments.FinalizePayment("MFA-REFUND", "successful", "71001", nil, repository.DepositCosts{ProviderKobo: &zero}); err != nil {
		t.Fatal(err)
	}
	token, err := c.Sessions.Create(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "agroshield_session", Value: token}
	handler := routes.NewHandler(c)
	form := url.Values{"order_id": {strconv.Itoa(orderID)}, "action": {"refund"}, "reason": {"Verified non-delivery; refund buyer"}, "csrf": {services.SecurityHash("agro-shield:admin-actions:" + token)}}
	assertMoney := func(state, balance string) {
		t.Helper()
		var gotState, gotBalance string
		if err := db.QueryRow(`SELECT e.state,w.balance::text FROM order_escrows e JOIN wallets w ON w.owner_type='user' AND w.owner_id=e.buyer_id AND w.currency=e.currency WHERE e.order_id=$1`, orderID).Scan(&gotState, &gotBalance); err != nil {
			t.Fatal(err)
		}
		if gotState != state || gotBalance != balance {
			t.Fatalf("escrow=%s wallet=%s, want escrow=%s wallet=%s", gotState, gotBalance, state, balance)
		}
	}
	requireStatus(t, request(handler, "POST", "/orders/protection", form, cookie), http.StatusForbidden)
	assertMoney("held", "0.00")
	memberPage := request(handler, "GET", "/orders/protection?order_id="+strconv.Itoa(orderID), nil, session(t, c, buyer))
	requireStatus(t, memberPage, http.StatusOK)
	authorizeFixtureAdmin(t, c, admin.ID, token)
	for i := 0; i < 2; i++ {
		requireStatus(t, request(handler, "POST", "/orders/protection", form, cookie), http.StatusSeeOther)
	}
	assertMoney("refunded", "200.00")
	if _, err := db.Exec(`UPDATE user_sessions SET mfa_verified_until=CURRENT_TIMESTAMP-interval '1 second' WHERE token_hash=$1`, services.SecurityHash(token)); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "POST", "/orders/protection", form, cookie), http.StatusForbidden)
	assertMoney("refunded", "200.00")
}

func TestPrivatePhotoAccessAuditingRetentionAndLimits(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	owner := fixtureUser(t, c, 1, "buyer")
	admin := fixtureUser(t, c, 2, "buyer")
	outsider := fixtureUser(t, c, 3, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := c.Security.PrivateImage(verificationJPEG(t))
	if err != nil {
		t.Fatal(err)
	}
	var photoID int64
	if err := db.QueryRow(`INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'withdrawal',$2) RETURNING id`, owner.ID, ciphertext).Scan(&photoID); err != nil {
		t.Fatal(err)
	}
	path := "/private/verification/" + strconv.FormatInt(photoID, 10)
	handler := routes.NewHandler(c)
	ownerCookie, outsiderCookie := session(t, c, owner), session(t, c, outsider)
	adminToken, err := c.Sessions.Create(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	adminCookie := &http.Cookie{Name: "agroshield_session", Value: adminToken}
	requireStatus(t, request(handler, "GET", path, nil, outsiderCookie), http.StatusNotFound)
	requireStatus(t, request(handler, "GET", path, nil, adminCookie), http.StatusNotFound)
	response := request(handler, "GET", path, nil, ownerCookie)
	requireStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Header().Get("Cache-Control"), "no-store") || response.Header().Get("Cross-Origin-Resource-Policy") != "same-origin" {
		t.Fatal("private photo response is cacheable or cross-origin")
	}
	authorizeFixtureAdmin(t, c, admin.ID, adminToken)
	requireStatus(t, request(handler, "GET", path, nil, adminCookie), http.StatusOK)
	if _, err := db.Exec(`CREATE FUNCTION reject_photo_audit_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated audit storage failure'; END; $$;
      CREATE TRIGGER reject_photo_audit_test BEFORE INSERT ON private_photo_access_logs FOR EACH ROW EXECUTE FUNCTION reject_photo_audit_test()`); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", path, nil, ownerCookie), http.StatusServiceUnavailable)
	if _, err := db.Exec(`DROP TRIGGER reject_photo_audit_test ON private_photo_access_logs`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE private_verification_photos SET encrypted_image=$2 WHERE id=$1`, photoID, []byte("tampered ciphertext")); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", path, nil, ownerCookie), http.StatusServiceUnavailable)
	var accesses int
	if err := db.QueryRow(`SELECT count(*) FROM private_photo_access_logs WHERE photo_id=$1 AND owner_id=$2 AND purpose='withdrawal' AND ((viewer_id=$2 AND session_hash=$3) OR (viewer_id=$4 AND session_hash=$5))`, photoID, owner.ID, services.SecurityHash(ownerCookie.Value), admin.ID, services.SecurityHash(adminToken)).Scan(&accesses); err != nil || accesses != 2 {
		t.Fatalf("authorized photo access audit=%d: %v", accesses, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM private_photo_access_logs`).Scan(&accesses); err != nil || accesses != 2 {
		t.Fatalf("failed photo read was recorded as successful: %d %v", accesses, err)
	}
	if err := c.Security.Repo.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM private_photo_access_logs WHERE photo_id IS NULL AND owner_id=$1`, owner.ID).Scan(&accesses); err != nil || accesses != 2 {
		t.Fatalf("retention lost the access audit: %d %v", accesses, err)
	}
	for i := 1; i < 60; i++ {
		requireStatus(t, request(handler, "GET", path, nil, outsiderCookie), http.StatusNotFound)
	}
	requireStatus(t, request(handler, "GET", path, nil, outsiderCookie), http.StatusTooManyRequests)
	restarted := routes.NewHandler(application(t, db, testConfig(t)))
	requireStatus(t, request(restarted, "GET", path, nil, outsiderCookie), http.StatusTooManyRequests)
}

func TestConcurrentFinancialFailuresProduceOneAlert(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	user := fixtureUser(t, c, 1, "buyer")
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := c.Security.Repo.Event(context.Background(), user.ID, "wallet_action_failed", "warning"); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	var failures, alerts int
	if err := db.QueryRow(`SELECT count(*) FILTER(WHERE kind='wallet_action_failed'),count(*) FILTER(WHERE kind='repeated_wallet_failures') FROM security_events WHERE user_id=$1`, user.ID).Scan(&failures, &alerts); err != nil || failures != 20 || alerts != 1 {
		t.Fatalf("failures=%d alerts=%d: %v", failures, alerts, err)
	}
}

func TestSecuritySensitiveRoutesUsePersistentLimits(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	admin := fixtureUser(t, c, 1, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	cookie := session(t, c, admin)
	handler := routes.NewHandler(c)
	for i := 0; i < 20; i++ {
		if response := request(handler, "POST", "/trust", url.Values{"action": {"invalid"}}, cookie); response.Code == http.StatusTooManyRequests {
			t.Fatal("sensitive action limited before its configured allowance")
		}
	}
	response := request(handler, "POST", "/trust", nil, cookie)
	requireStatus(t, response, http.StatusTooManyRequests)
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("rate limit omitted retry guidance")
	}
	// Requests on another instance share the same account limit.
	restarted := routes.NewHandler(application(t, db, testConfig(t)))
	requireStatus(t, request(restarted, "POST", "/delivery-partner/review", nil, cookie), http.StatusTooManyRequests)
	if strings.Contains(response.Body.String(), cookie.Value) {
		t.Fatal("rate limit exposed the session token")
	}
}
