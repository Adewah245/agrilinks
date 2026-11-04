package integration_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
)

func authenticatorCode(secret string) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(time.Now().Unix()/30))
	h := hmac.New(sha1.New, key)
	h.Write(b[:])
	sum := h.Sum(nil)
	off := sum[len(sum)-1] & 15
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[off:off+4])&0x7fffffff)%1000000)
}

func TestAdminMFAEnrollmentSessionBindingReplayAndRecovery(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	ctx := context.Background()
	u := fixtureUser(t, c, 1, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, u.ID); err != nil {
		t.Fatal(err)
	}
	u, _ = c.UserRepo.GetByID(u.ID)
	token, err := c.Sessions.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "agroshield_session", Value: token}
	handler := routes.NewHandler(c)
	response := request(handler, "GET", "/admin/finance", nil, cookie)
	requireStatus(t, response, 303)
	if response.Header().Get("Location") != "/admin/security" {
		t.Fatal("administrator bypassed MFA")
	}
	requireStatus(t, request(handler, "POST", "/admin/action", nil, cookie), 403)
	if _, _, err = c.Security.BeginMFA(ctx, u, token, "wrong password", false); err == nil {
		t.Fatal("enrolled without password")
	}
	secret, _, err := c.Security.BeginMFA(ctx, u, token, "test password with spaces", false)
	if err != nil {
		t.Fatal(err)
	}
	m, err := c.Security.Repo.MFA(ctx, u.ID)
	if err != nil || strings.Contains(string(m.Secret), secret) {
		t.Fatal("authenticator secret stored in plaintext")
	}
	second, err := c.Sessions.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Security.VerifyMFA(ctx, u, second, authenticatorCode(secret)); err == nil {
		t.Fatal("setup verified from another session")
	}
	code := authenticatorCode(secret)
	var successes atomic.Int32
	var codes []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recovery, err := c.Security.VerifyMFA(ctx, u, token, code)
			if err == nil {
				successes.Add(1)
				mu.Lock()
				codes = recovery
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || len(codes) != 10 {
		t.Fatalf("MFA concurrency/recovery: successes=%d codes=%d", successes.Load(), len(codes))
	}
	requireStatus(t, request(handler, "GET", "/admin/finance", nil, cookie), 200)
	if fresh, _ := c.Security.FreshMFA(ctx, u.ID, second); fresh {
		t.Fatal("MFA shared across sessions")
	}
	if _, err = c.Security.VerifyMFA(ctx, u, token, code); err == nil {
		t.Fatal("TOTP replay accepted")
	}
	if err = c.Security.RecoverMFA(ctx, u, second, "test password with spaces", codes[0]); err != nil {
		t.Fatal(err)
	}
	// Verify consumption independently of the attempt limiter.
	if err = c.Security.Repo.RecoverMFA(ctx, u.ID, services.SecurityHash(second), services.SecurityHash(codes[0])); err == nil {
		t.Fatal("recovery code reused")
	}
	if _, err = db.Exec(`UPDATE user_sessions SET mfa_verified_until=CURRENT_TIMESTAMP-interval '1 second' WHERE token_hash=$1`, services.SecurityHash(token)); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", "/admin/finance", nil, cookie), 303)
	if _, _, err = c.Security.BeginMFA(ctx, u, token, "test password with spaces", true); err == nil {
		t.Fatal("expired MFA reset authenticator")
	}
}

func TestAdminMFAReplacementKeepsProtectionUntilConfirmed(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	ctx := context.Background()
	u := fixtureUser(t, c, 1, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, u.ID); err != nil {
		t.Fatal(err)
	}
	u, _ = c.UserRepo.GetByID(u.ID)
	session, err := c.Sessions.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	secret, _, err := c.Security.BeginMFA(ctx, u, session, "test password with spaces", false)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := c.Security.VerifyMFA(ctx, u, session, authenticatorCode(secret))
	if err != nil {
		t.Fatal(err)
	}
	other, err := c.Sessions.Create(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	replacement, _, err := c.Security.BeginMFA(ctx, u, session, "test password with spaces", true)
	if err != nil {
		t.Fatal(err)
	}
	m, err := c.Security.Repo.MFA(ctx, u.ID)
	if err != nil || !m.Enabled || len(m.ReplacementSecret) == 0 {
		t.Fatal("replacement disabled the active factor", err)
	}
	if _, _, err = c.Security.BeginMFA(ctx, u, other, "test password with spaces", false); err == nil {
		t.Fatal("password-only session replaced the active factor")
	}
	m.UseReplacement = true
	if err = c.Security.Repo.ConfirmMFA(ctx, u.ID, services.SecurityHash(other), m, time.Now().Unix()/30, nil); err == nil {
		t.Fatal("replacement crossed sessions")
	}
	if err = c.Security.RecoverMFA(ctx, u, other, "test password with spaces", codes[0]); err != nil {
		t.Fatal("active recovery stopped working before replacement", err)
	}
	newCodes, err := c.Security.VerifyMFA(ctx, u, session, authenticatorCode(replacement))
	if err != nil || len(newCodes) != 10 {
		t.Fatal("replacement failed", err)
	}
	if err = c.Security.Repo.RecoverMFA(ctx, u.ID, services.SecurityHash(session), services.SecurityHash(codes[1])); err == nil {
		t.Fatal("old recovery code survived replacement")
	}
	if id, err := c.Sessions.UserID(ctx, other); err != nil || id != 0 {
		t.Fatal("other session survived replacement", err)
	}
	if fresh, err := c.Security.FreshMFA(ctx, u.ID, session); err != nil || !fresh {
		t.Fatal("replacement session not verified", err)
	}
	if _, _, err = c.Security.BeginMFA(ctx, u, session, "test password with spaces", true); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE admin_mfa SET setup_expires_at=CURRENT_TIMESTAMP-interval '1 second' WHERE user_id=$1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if fresh, err := c.Security.FreshMFA(ctx, u.ID, session); err != nil || !fresh {
		t.Fatal("abandoned replacement disabled MFA", err)
	}
	if _, _, err = c.Security.BeginMFA(ctx, u, session, "test password with spaces", false); err == nil {
		t.Fatal("expired replacement allowed password-only enrollment")
	}
}

func TestPrivateEvidenceBindingValidationAndAccess(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	ctx := context.Background()
	owner := fixtureUser(t, c, 1, "buyer")
	outsider := fixtureUser(t, c, 2, "buyer")
	admin := fixtureUser(t, c, 3, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	ownerCookie := session(t, c, owner)
	outsideCookie := session(t, c, outsider)
	adminCookie := session(t, c, admin)
	quote := strings.Repeat("a", 32)
	// An authentic fee quote is required to obtain a capture challenge.
	if _, err := c.Security.StartEvidence(ctx, owner.ID, ownerCookie.Value, quote); err == nil {
		t.Fatal("challenge accepted without a fee quote")
	}
	_, err := db.Exec(`INSERT INTO wallet_fee_quotes(id,user_id,kind,currency,amount_kobo,service_kobo,provider_kobo,customer_provider_kobo,wallet_debit_kobo,wallet_credit_kobo,bank_credit_kobo,checkout_total_kobo,provider_evidence,policy_id,expires_at) VALUES($1,$2,'withdrawal','NGN',10000,0,0,0,10000,0,10000,0,'{}',(SELECT id FROM wallet_fee_policies ORDER BY id DESC LIMIT 1),CURRENT_TIMESTAMP+interval '10 minutes')`, quote, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.Security.StartEvidence(ctx, owner.ID, ownerCookie.Value, quote)
	if err != nil {
		t.Fatal(err)
	}
	e := models.WithdrawalEvidence{Latitude: 7.7, Longitude: 8.5, AccuracyM: 10, CapturedAt: time.Now()}
	if err = c.Security.SaveEvidence(ctx, outsider.ID, outsideCookie.Value, token, verificationJPEG(t), e); err == nil {
		t.Fatal("another user supplied evidence")
	}
	otherSession, _ := c.Sessions.Create(ctx, owner.ID)
	if err = c.Security.SaveEvidence(ctx, owner.ID, otherSession, token, verificationJPEG(t), e); err == nil {
		t.Fatal("challenge crossed sessions")
	}
	bad := e
	bad.AccuracyM = 1001
	if err = c.Security.SaveEvidence(ctx, owner.ID, ownerCookie.Value, token, verificationJPEG(t), bad); err == nil {
		t.Fatal("imprecise location accepted")
	}
	bad = e
	bad.CapturedAt = time.Now().Add(-10 * time.Minute)
	if err = c.Security.SaveEvidence(ctx, owner.ID, ownerCookie.Value, token, verificationJPEG(t), bad); err == nil {
		t.Fatal("stale GPS accepted")
	}
	if err = c.Security.SaveEvidence(ctx, owner.ID, ownerCookie.Value, token, verificationJPEG(t), e); err != nil {
		t.Fatal(err)
	}
	if err = c.Security.SaveEvidence(ctx, owner.ID, ownerCookie.Value, token, verificationJPEG(t), e); err == nil {
		t.Fatal("capture token replay accepted")
	}
	var photoID int64
	var ciphertext []byte
	if err = db.QueryRow(`SELECT p.id,p.encrypted_image FROM private_verification_photos p JOIN withdrawal_evidence e ON e.photo_id=p.id WHERE e.token_hash=$1`, services.SecurityHash(token)).Scan(&photoID, &ciphertext); err != nil {
		t.Fatal(err)
	}
	if len(ciphertext) > 2 && ciphertext[0] == 0xff && ciphertext[1] == 0xd8 {
		t.Fatal("image stored in plaintext")
	}
	handler := routes.NewHandler(c)
	path := fmt.Sprintf("/private/verification/%d", photoID)
	requireStatus(t, request(handler, "GET", path, nil, nil), 303)
	requireStatus(t, request(handler, "GET", path, nil, outsideCookie), 404)
	requireStatus(t, request(handler, "GET", path, nil, ownerCookie), 200)
	response := request(handler, "GET", path, nil, adminCookie)
	requireStatus(t, response, 200)
	if !strings.Contains(response.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("private photo cached")
	}
	if _, err = db.Exec(`UPDATE users SET staff_role='customer_care' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", path, nil, adminCookie), 404)
	if _, err = db.Exec(`UPDATE withdrawal_evidence SET expires_at=CURRENT_TIMESTAMP-interval '2 hours' WHERE token_hash=$1`, services.SecurityHash(token)); err != nil {
		t.Fatal(err)
	}
	if err = c.Security.Repo.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", path, nil, ownerCookie), 404)
}

func TestSecurityRateLimitsPersistAndDetectRepeatedFailures(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	ctx := context.Background()
	u := fixtureUser(t, c, 1, "buyer")
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.Security.Limit(ctx, u.ID, "financial_test", 5, time.Minute) == nil {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("atomic rate limit allowed %d", allowed.Load())
	}
	restarted := application(t, db, testConfig(t))
	if err := restarted.Security.Limit(ctx, u.ID, "financial_test", 5, time.Minute); !errors.Is(err, services.ErrSecurityLimit) {
		t.Fatal("restart reset account rate limit")
	}
	for i := 0; i < 5; i++ {
		c.Security.Event(ctx, u.ID, "wallet_action_failed", "warning")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM security_events WHERE user_id=$1 AND kind='repeated_wallet_failures' AND severity='high'`, u.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("suspicious activity not reported: %d %v", count, err)
	}
}

func TestLegacyVerificationPhotographsAreMovedOutOfPublicStorage(t *testing.T) {
	db, _ := testutil.Database(t, true)
	cfg := testConfig(t)
	cfg.FrontendDir = t.TempDir()
	c := application(t, db, cfg)
	u := fixtureUser(t, c, 1, "buyer")
	dir := filepath.Join(cfg.FrontendDir, "uploads", "users")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "old-selfie.jpg")
	// Historic uploads accepted up to 10 MB; appended bytes must be stripped.
	legacy := append(verificationJPEG(t), make([]byte, 3<<20)...)
	if err := os.WriteFile(file, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO legacy_private_photo_paths(public_path,user_id) VALUES('/static/uploads/users/old-selfie.jpg',$1)`, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Security.MigrateLegacyPhotos(context.Background(), cfg.FrontendDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("legacy verification file remains publicly stored")
	}
	if id, err := c.Security.Repo.RegistrationPhoto(context.Background(), u.ID); err != nil || id == 0 {
		t.Fatal("legacy verification photo lost")
	}
	if err := c.Security.MigrateLegacyPhotos(context.Background(), cfg.FrontendDir); err != nil {
		t.Fatal("migration is not restart-safe")
	}
}
