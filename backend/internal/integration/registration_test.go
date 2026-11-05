package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testui"
	"backend/internal/testutil"
)

func TestRegistrationPhoneConfirmationWithPostgresAndOTPDev(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	var code string
	requests := 0
	failDelivery := false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "POST" || r.URL.Path != "/verifications" || r.Header.Get("X-OTP-Key") != "signup-test-key" {
			t.Error("incorrect OTP.dev request")
		}
		var body struct {
			Data struct{ Phone, Code, Channel string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		code = body.Data.Code
		if failDelivery {
			http.Error(w, "provider unavailable", 503)
			return
		}
		if body.Data.Channel != "sms" {
			t.Error("registration did not use SMS")
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"message_id": "signup-message", "phone": body.Data.Phone}})
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.FrontendDir = t.TempDir()
	cfg.SMSProvider, cfg.OTPDevAPIKey = "otpdev", "signup-test-key"
	cfg.OTPDevSender, cfg.OTPDevTemplateID, cfg.OTPDevBaseURL = "AgroShield", "test-template", provider.URL
	c := application(t, db, cfg)
	existing := fixtureUser(t, c, 1, "buyer")
	handler := routes.NewHandler(c)
	phone := "2348012345678"
	register := func(token, enteredCode, targetPhone, email string) error {
		if err := c.Auth.VerifyRegistrationPhone(ctx, token, targetPhone, enteredCode); err != nil {
			return err
		}
		return c.Auth.Register(ctx, token, "582491", "582491", "New", "Buyer", targetPhone, email,
			"signup password", "Makurdi", "Makurdi", "Benue", "Nigeria", "buyer", "individual", "", nil, verificationJPEG(t))
	}
	noNewAccount := func() {
		t.Helper()
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("unconfirmed signup created an account: %d %v", count, err)
		}
		requireRegistrationUploads(t, cfg.FrontendDir, 0)
	}
	allowResend := func() {
		t.Helper()
		if _, err := db.Exec(`UPDATE registration_phone_codes SET created_at=CURRENT_TIMESTAMP-INTERVAL '20 minutes' WHERE phone=$1`, phone); err != nil {
			t.Fatal(err)
		}
	}
	sendCode := func() *http.Cookie {
		t.Helper()
		response := request(handler, "POST", "/register/phone-code", url.Values{"phone": {"08012345678"}}, nil)
		requireStatus(t, response, http.StatusOK)
		cookies := response.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Path != "/register" || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatal("signup challenge cookie is missing protections")
		}
		if strings.Contains(response.Body.String(), code) {
			t.Fatal("code leaked into HTTP response")
		}
		return cookies[0]
	}

	if err := register("", "", phone, "new@example.test"); err == nil {
		t.Fatal("registration succeeded without confirmation")
	}
	noNewAccount()
	cookie := sendCode()
	noNewAccount()
	requireStatus(t, request(handler, "POST", "/register/phone-code", url.Values{"phone": {phone}}, nil), http.StatusTooManyRequests)
	if requests != 1 {
		t.Fatal("resend cooldown did not stop provider request")
	}
	if err := register(strings.Repeat("a", 64), code, phone, "new@example.test"); err == nil {
		t.Fatal("different browser accepted")
	}
	if err := register(cookie.Value, code, "2348099999999", "new@example.test"); err == nil {
		t.Fatal("different phone accepted")
	}
	wrongCode := "000000"
	if code == wrongCode {
		wrongCode = "111111"
	}
	for i := 0; i < cfg.OTPMaxAttempts; i++ {
		if err := register(cookie.Value, wrongCode, phone, "new@example.test"); err == nil {
			t.Fatal("wrong code accepted")
		}
	}
	if err := register(cookie.Value, code, phone, "new@example.test"); err == nil {
		t.Fatal("attempt limit bypassed")
	}
	noNewAccount()

	allowResend()
	oldCookie, oldCode := cookie, code
	cookie = sendCode()
	if err := register(oldCookie.Value, oldCode, phone, "new@example.test"); err == nil {
		t.Fatal("replaced challenge accepted")
	}
	allowResend()
	if _, err := db.Exec(`UPDATE registration_phone_codes SET expires_at=CURRENT_TIMESTAMP-INTERVAL '1 second' WHERE phone=$1`, phone); err != nil {
		t.Fatal(err)
	}
	if err := register(cookie.Value, code, phone, "new@example.test"); err == nil {
		t.Fatal("expired code accepted")
	}
	cookie = sendCode()
	if err := register(cookie.Value, code, phone, existing.Email); !errors.Is(err, repository.ErrEmailExists) {
		t.Fatalf("expected duplicate-email failure: %v", err)
	}
	noNewAccount()
	if err := register(cookie.Value, code, phone, "new@example.test"); err != nil {
		t.Fatalf("retry after failed insert: %v", err)
	}
	user, err := c.Auth.Login(phone, "signup password")
	if err != nil || user.PhoneVerifiedAt == nil {
		t.Fatalf("confirmed account did not become usable: %v", err)
	}
	sum := sha256.Sum256([]byte(cookie.Value))
	if _, err := repository.NewRegistrationOTPRepository(db).Attempt(ctx, hex.EncodeToString(sum[:]), phone, 5); !errors.Is(err, repository.ErrRegistrationOTPInvalid) {
		t.Fatal("used code accepted again")
	}
	if err := c.Auth.ResetPassword(ctx, phone, "replacement password", code); err == nil {
		t.Fatal("signup code used for password recovery")
	}
	if _, err := db.Exec(`UPDATE users SET phone='2348099999999' WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	changed, err := c.UserRepo.GetByID(user.ID)
	if err != nil || changed.PhoneVerifiedAt != nil {
		t.Fatal("confirmation remained attached to a changed phone")
	}

	allowResend()
	failDelivery = true
	response := request(handler, "POST", "/register/phone-code", url.Values{"phone": {phone}}, nil)
	if response.Code == 200 || len(response.Result().Cookies()) != 0 {
		t.Fatal("failed delivery appeared successful")
	}
	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM registration_phone_codes WHERE phone=$1`, phone).Scan(&pending); err != nil || pending != 0 {
		t.Fatal("failed delivery left a usable challenge")
	}
}

func TestRegistrationAttemptLimitIsAtomic(t *testing.T) {
	db, _ := testutil.Database(t, true)
	repo := repository.NewRegistrationOTPRepository(db)
	code := &models.RegistrationPhoneCode{Phone: "2348012345678", TokenHash: strings.Repeat("a", 64), CodeHash: "test-hash"}
	if err := repo.Create(context.Background(), code, time.Minute, time.Minute); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Attempt(context.Background(), code.TokenHash, code.Phone, 5)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if !errors.Is(err, repository.ErrRegistrationOTPInvalid) {
			t.Fatal(err)
		}
	}
	if accepted != 5 {
		t.Fatalf("reserved %d attempts, want 5", accepted)
	}
}

func TestRegistrationRequiresPINAfterVerifiedFifthAttempt(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	cfg := testConfig(t)
	cfg.FrontendDir = t.TempDir()
	c := application(t, db, cfg)
	delivery := &captureOTP{}
	c.Auth.SetRegistrationOTP(services.NewRegistrationOTPService(repository.NewRegistrationOTPRepository(db), delivery, cfg.OTPTTL, 5, 0))
	phone := "2348012345678"
	token, err := c.Auth.RequestRegistrationCode(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	register := func(pin, confirmation string) error {
		return c.Auth.Register(ctx, token, pin, confirmation, "New", "Buyer", phone, "new@example.test", "signup password", "Makurdi", "Makurdi", "Benue", "Nigeria", "buyer", "individual", "", nil, verificationJPEG(t))
	}
	if err = register("582491", "582491"); err == nil {
		t.Fatal("registration bypassed phone verification")
	}
	wrong := "000000"
	if delivery.message.Code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 4; i++ {
		if err = c.Auth.VerifyRegistrationPhone(ctx, token, phone, wrong); err == nil {
			t.Fatal("incorrect phone code accepted")
		}
	}
	if err = c.Auth.VerifyRegistrationPhone(ctx, token, phone, delivery.message.Code); err != nil {
		t.Fatal(err)
	}
	if err = register("582491", "582492"); err == nil {
		t.Fatal("mismatched PIN accepted")
	}
	if err = register("", " "); err == nil {
		t.Fatal("missing PIN accepted")
	}
	requireRegistrationUploads(t, cfg.FrontendDir, 0)
	if err = register("582491", "582491"); err != nil {
		t.Fatalf("fifth correct attempt could not finish registration: %v", err)
	}
	requireRegistrationUploads(t, cfg.FrontendDir, 1)
	user, err := c.Auth.FindByPhone(phone)
	if err != nil || user == nil {
		t.Fatal(err)
	}
	if err = c.UserRepo.VerifyTransactionPIN(ctx, user.ID, "582491"); err != nil {
		t.Fatalf("registration did not persist PIN: %v", err)
	}
	sum := sha256.Sum256([]byte(token))
	if _, err = repository.NewRegistrationOTPRepository(db).Verified(ctx, hex.EncodeToString(sum[:]), phone); !errors.Is(err, repository.ErrRegistrationOTPInvalid) {
		t.Fatalf("used proof accepted: %v", err)
	}
}

func TestRegistrationHTTPRequiresConfirmation(t *testing.T) {
	db, _ := testutil.Database(t, true)
	cfg := testConfig(t)
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "frontend"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg.FrontendDir = filepath.Join(workspace, "frontend")
	c := application(t, db, cfg)
	capture := &captureOTP{}
	c.Auth.SetRegistrationOTP(services.NewRegistrationOTPService(repository.NewRegistrationOTPRepository(db), capture, cfg.OTPTTL, cfg.OTPMaxAttempts, cfg.OTPResendInterval))
	handler := routes.NewHandler(c)
	send := request(handler, "POST", "/register/phone-code", url.Values{"phone": {"08012345678"}}, nil)
	requireStatus(t, send, 200)
	cookie := send.Result().Cookies()[0]
	var selfie bytes.Buffer
	portrait := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			portrait.SetRGBA(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 128, A: 255})
		}
	}
	if err := png.Encode(&selfie, portrait); err != nil {
		t.Fatal(err)
	}
	selfie.WriteString("<script>registration-upload-payload</script>")
	post := func(cookie *http.Cookie, code string, picture []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for key, value := range map[string]string{"first-name": "HTTP", "last-name": "Buyer", "phone": "08012345678", "email": "http@example.test", "password": "signup password", "confirm-password": "signup password", "country": "Nigeria", "state": "Benue", "lga": "Makurdi", "community": "Makurdi", "purpose": "buy", "account_type": "individual", "registration_code": code, "transaction_pin": "582491", "transaction_pin_confirmation": "582491"} {
			if err := form.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
		if picture != nil {
			file, err := form.CreateFormFile("profile_picture", "selfie.jpg")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(picture); err != nil {
				t.Fatal(err)
			}
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("POST", "http://example.test/register", &body)
		r.Header.Set("Content-Type", form.FormDataContentType())
		r.Header.Set("Origin", "http://example.test")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	noNewAccount := func() {
		t.Helper()
		var users, photos, pins, wallets int
		if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM users), (SELECT COUNT(*) FROM private_verification_photos), (SELECT COUNT(*) FROM user_transaction_pins), (SELECT COUNT(*) FROM wallets)`).Scan(&users, &photos, &pins, &wallets); err != nil {
			t.Fatal(err)
		}
		if users != 0 || photos != 0 || pins != 0 || wallets != 0 {
			t.Fatalf("failed registration left account data: users=%d photos=%d pins=%d wallets=%d", users, photos, pins, wallets)
		}
		requireRegistrationUploads(t, cfg.FrontendDir, 0)
	}
	requireStatus(t, post(nil, capture.message.Code, selfie.Bytes()), 400)
	requireStatus(t, post(cookie, "", selfie.Bytes()), 400)
	bad := post(cookie, "abcdef", selfie.Bytes())
	requireStatus(t, bad, 400)
	requireStatus(t, post(cookie, capture.message.Code, nil), 400)
	noNewAccount()
	verify := request(handler, "POST", "/register/phone-verify", url.Values{"phone": {"08012345678"}, "code": {capture.message.Code}}, cookie)
	requireStatus(t, verify, 200)
	for _, invalid := range []struct {
		name string
		data []byte
	}{
		{"missing selfie", nil},
		{"empty selfie", []byte{}},
		{"invalid selfie", []byte("not an image")},
		{"oversized selfie", bytes.Repeat([]byte("a"), (2<<20)+1)},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			requireStatus(t, post(cookie, capture.message.Code, invalid.data), 400)
			noNewAccount()
		})
	}
	// Mandatory private evidence also prevents signup when encryption is absent.
	c.Auth.SetSecurity(nil)
	requireStatus(t, post(cookie, capture.message.Code, selfie.Bytes()), http.StatusServiceUnavailable)
	noNewAccount()
	c.Auth.SetSecurity(c.Security)
	// A storage failure cannot create an account or consume the phone proof.
	uploadRoot := filepath.Join(cfg.FrontendDir, "uploads")
	if err := os.Mkdir(uploadRoot, 0755); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(uploadRoot, "profiles")
	if err := os.WriteFile(blocker, []byte("storage failure fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	failedStorage := post(cookie, capture.message.Code, selfie.Bytes())
	requireStatus(t, failedStorage, http.StatusServiceUnavailable)
	if strings.Contains(failedStorage.Body.String(), cfg.FrontendDir) {
		t.Fatal("storage error exposed a server filesystem path")
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	noNewAccount()
	// Fail after the user insert to exercise account, PIN, wallet, proof, and
	// public-file rollback when the private original cannot be persisted.
	if _, err := db.Exec(`ALTER TABLE private_verification_photos ADD CONSTRAINT registration_photo_test_failure CHECK (purpose <> 'registration')`); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, post(cookie, capture.message.Code, selfie.Bytes()), 400)
	noNewAccount()
	if _, err := db.Exec(`ALTER TABLE private_verification_photos DROP CONSTRAINT registration_photo_test_failure`); err != nil {
		t.Fatal(err)
	}
	success := post(cookie, capture.message.Code, selfie.Bytes())
	requireStatus(t, success, 303)
	if success.Header().Get("Location") != "/login" {
		t.Fatal("confirmed signup did not reach login")
	}
	u, err := c.Auth.FindByPhone("08012345678")
	if err != nil || u == nil || u.PhoneVerifiedAt == nil {
		t.Fatalf("phone was not confirmed: %v", err)
	}
	// The public picture and encrypted original derive from the same sanitized
	// capture. The source PNG and appended payload are never published.
	requireRegistrationUploads(t, cfg.FrontendDir, 1)
	if !strings.HasPrefix(u.PhotoURL, "/static/uploads/profiles/") || !strings.HasSuffix(u.PhotoURL, ".jpg") {
		t.Fatalf("registration did not set the public selfie: %q", u.PhotoURL)
	}
	normalized, err := services.NormalizeVerificationImage(selfie.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	publicPhoto := request(handler, "GET", u.PhotoURL, nil, nil)
	requireStatus(t, publicPhoto, http.StatusOK)
	if publicPhoto.Header().Get("Content-Type") != "image/jpeg" || !bytes.Equal(publicPhoto.Body.Bytes(), normalized) || bytes.Contains(publicPhoto.Body.Bytes(), []byte("registration-upload-payload")) {
		t.Fatal("public profile picture is not the sanitized captured selfie")
	}
	photoID, err := c.Security.Repo.RegistrationPhoto(context.Background(), u.ID)
	if err != nil || photoID == 0 {
		t.Fatalf("missing private registration photo: %v", err)
	}
	var encryptedOriginal []byte
	if err := db.QueryRow(`SELECT encrypted_image FROM private_verification_photos WHERE id=$1`, photoID).Scan(&encryptedOriginal); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(encryptedOriginal, normalized) || bytes.Equal(encryptedOriginal, selfie.Bytes()) {
		t.Fatal("verification original was stored unencrypted")
	}
	photoPath := fmt.Sprintf("/private/verification/%d", photoID)
	requireStatus(t, request(handler, "GET", photoPath, nil, nil), 303)
	login := request(handler, "POST", "/login", url.Values{"phone": {u.Phone}, "password": {"signup password"}}, nil)
	requireStatus(t, login, 303)
	photo := request(handler, "GET", photoPath, nil, login.Result().Cookies()[0])
	requireStatus(t, photo, 200)
	if !bytes.Equal(photo.Body.Bytes(), normalized) || !strings.Contains(photo.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("private photograph response incorrect")
	}
	profile := request(handler, "GET", "/profile", nil, login.Result().Cookies()[0])
	requireStatus(t, profile, 200)
	profileHTML := testui.HTML(t, profile.Body.String())
	if strings.Contains(profileHTML, photoPath) {
		t.Fatal("private photograph leaked into profile")
	}
	if !strings.Contains(profileHTML, u.PhotoURL) {
		t.Fatal("registration selfie did not appear as the profile picture")
	}
	// Editing the public picture must never replace the comparison original.
	const replacement = "/static/assets/images/un.jpeg"
	if err := c.Auth.UpdateProfile(u.ID, u.FullName, u.Phone, u.Email, u.Location, u.State, u.LGA, "", replacement); err != nil {
		t.Fatal(err)
	}
	updated, err := c.UserRepo.GetByID(u.ID)
	if err != nil || updated == nil || updated.PhotoURL != replacement {
		t.Fatalf("public profile picture was not updated: %v", err)
	}
	retainedID, err := c.Security.Repo.RegistrationPhoto(context.Background(), u.ID)
	if err != nil || retainedID != photoID {
		t.Fatalf("profile edit replaced the registration original: %v", err)
	}
	var retainedOriginal []byte
	if err := db.QueryRow(`SELECT encrypted_image FROM private_verification_photos WHERE id=$1`, photoID).Scan(&retainedOriginal); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encryptedOriginal, retainedOriginal) {
		t.Fatal("profile edit changed the encrypted registration original")
	}
	privateAfterEdit := request(handler, "GET", photoPath, nil, login.Result().Cookies()[0])
	requireStatus(t, privateAfterEdit, http.StatusOK)
	if !bytes.Equal(privateAfterEdit.Body.Bytes(), normalized) {
		t.Fatal("private comparison no longer uses the original registration selfie")
	}
	for _, cookie := range success.Result().Cookies() {
		if cookie.Name == "agroshield_registration" && cookie.MaxAge < 0 {
			return
		}
	}
	t.Fatal("signup challenge cookie was not cleared")
}

func requireRegistrationUploads(t *testing.T, frontendDir string, want int) {
	t.Helper()
	var files []string
	err := filepath.WalkDir(frontendDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
			if filepath.Dir(path) != filepath.Join(frontendDir, "uploads", "profiles") || filepath.Ext(path) != ".jpg" {
				t.Errorf("registration left an unexpected file: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != want {
		t.Fatalf("registration left %d uploaded files, want %d: %v", len(files), want, files)
	}
}
