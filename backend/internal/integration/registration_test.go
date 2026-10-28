package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
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
		json.NewEncoder(w).Encode(map[string]string{"message_id": "signup-message", "phone": body.Data.Phone})
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.SMSProvider, cfg.OTPDevAPIKey = "otpdev", "signup-test-key"
	cfg.OTPDevSender, cfg.OTPDevTemplateID, cfg.OTPDevBaseURL = "AgroShield", "test-template", provider.URL
	c := application(t, db, cfg)
	existing := fixtureUser(t, c, 1, "buyer")
	handler := routes.NewHandler(c)
	phone := "2348012345678"
	register := func(token, enteredCode, targetPhone, email string) error {
		return c.Auth.Register(ctx, token, enteredCode, "New", "Buyer", targetPhone, email,
			"signup password", "Makurdi", "Makurdi", "Benue", "Nigeria", "buyer", "/static/assets/images/un.jpeg", "individual", "", nil)
	}
	noNewAccount := func() {
		t.Helper()
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("unconfirmed signup created an account: %d %v", count, err)
		}
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

func TestRegistrationHTTPRequiresConfirmation(t *testing.T) {
	db, _ := testutil.Database(t, true)
	cfg := testConfig(t)
	workspace := t.TempDir()
	t.Chdir(workspace)
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
	if err := jpeg.Encode(&selfie, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	post := func(cookie *http.Cookie, code string, includeSelfie bool) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for key, value := range map[string]string{"first-name": "HTTP", "last-name": "Buyer", "phone": "08012345678", "email": "http@example.test", "password": "signup password", "confirm-password": "signup password", "country": "Nigeria", "state": "Benue", "lga": "Makurdi", "community": "Makurdi", "purpose": "buy", "account_type": "individual", "registration_code": code} {
			if err := form.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
		if includeSelfie {
			file, err := form.CreateFormFile("profile_picture", "selfie.jpg")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(selfie.Bytes()); err != nil {
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
	requireStatus(t, post(nil, capture.message.Code, true), 400)
	requireStatus(t, post(cookie, "", true), 400)
	bad := post(cookie, "abcdef", true)
	requireStatus(t, bad, 400)
	requireStatus(t, post(cookie, capture.message.Code, false), 400)
	success := post(cookie, capture.message.Code, true)
	requireStatus(t, success, 303)
	if success.Header().Get("Location") != "/login" {
		t.Fatal("confirmed signup did not reach login")
	}
	u, err := c.Auth.FindByPhone("08012345678")
	if err != nil || u == nil || u.PhoneVerifiedAt == nil {
		t.Fatalf("phone was not confirmed: %v", err)
	}
	// Only the successful request owns an uploaded file; failures remove theirs.
	files, err := os.ReadDir(filepath.Join(workspace, "frontend", "uploads", "users"))
	if err != nil || len(files) != 1 {
		t.Fatalf("failed signup left an orphaned photo: %d %v", len(files), err)
	}
	if u.PhotoURL != "/static/uploads/users/"+files[0].Name() {
		t.Fatalf("captured selfie was not saved as the profile picture: %q", u.PhotoURL)
	}
	photo := request(handler, "GET", u.PhotoURL, nil, nil)
	requireStatus(t, photo, http.StatusOK)
	if photo.Header().Get("Content-Type") != "image/jpeg" || !bytes.Equal(photo.Body.Bytes(), selfie.Bytes()) {
		t.Fatal("saved profile picture does not match the captured selfie")
	}
	login := request(handler, "POST", "/login", url.Values{"phone": {u.Phone}, "password": {"signup password"}}, nil)
	requireStatus(t, login, http.StatusSeeOther)
	profile := request(handler, "GET", "/profile", nil, login.Result().Cookies()[0])
	requireStatus(t, profile, http.StatusOK)
	if !strings.Contains(profile.Body.String(), `src="`+u.PhotoURL+`"`) {
		t.Fatal("profile does not show the captured selfie after login")
	}
	for _, cookie := range success.Result().Cookies() {
		if cookie.Name == "agroshield_registration" && cookie.MaxAge < 0 {
			return
		}
	}
	t.Fatal("signup challenge cookie was not cleared")
}
