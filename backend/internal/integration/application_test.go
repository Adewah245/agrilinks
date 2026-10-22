package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	frontend, err := filepath.Abs("../../../frontend")
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{Port: "8080", AppEnv: "test", FrontendDir: frontend, SessionTTL: time.Hour,
		RequestTimeout: time.Minute, ShutdownTimeout: time.Second, MaxRequestBytes: 64 << 20,
		AuthRequestsPerMinute: 1000, AIRequestsPerMinute: 1000, PaymentCurrency: "NGN", SMSProvider: "disabled",
		OTPTTL: 10 * time.Minute, OTPMaxAttempts: 5, OTPResendInterval: time.Minute,
		AIRequestTimeout: time.Second, AIMaxTokens: 500, Negotiation: config.NegotiationPolicy{MaxRounds: 5, Window: 24 * time.Hour}}
}

func application(t *testing.T, db *sql.DB, cfg config.Config) *app.Container {
	t.Helper()
	container, err := app.New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Page checks do not ask a live weather provider about synthetic users.
	container.Weather = nil
	return container
}

func fixtureUser(t *testing.T, c *app.Container, number int, role string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("test password with spaces"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	u := &models.User{FullName: fmt.Sprintf("Test User %d", number), Phone: fmt.Sprintf("234801%07d", number),
		Email: fmt.Sprintf("user%d@example.test", number), PasswordHash: string(hash), Role: role, AccountType: "individual",
		Location: "Otukpo", LGA: "Otukpo", State: "Benue", Country: "Nigeria", PhotoURL: "/static/assets/images/un.jpeg"}
	if _, err := c.UserRepo.Create(u); err != nil {
		t.Fatal(err)
	}
	return u
}

func session(t *testing.T, c *app.Container, user *models.User) *http.Cookie {
	t.Helper()
	token, err := c.Sessions.Create(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "agroshield_session", Value: token}
}

func request(handler http.Handler, method, path string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://example.test"+path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://example.test")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func requireStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP %d, want %d: %.1000s", w.Code, status, w.Body.String())
	}
}

type captureOTP struct{ message services.OTPMessage }

func (d *captureOTP) SendOTP(_ context.Context, message services.OTPMessage) error {
	d.message = message
	return nil
}

func TestAuthenticationAndPasswordRecoveryWithPostgres(t *testing.T) {
	db, _ := testutil.Database(t, true)
	cfg := testConfig(t)
	c := application(t, db, cfg)
	u := fixtureUser(t, c, 1, "buyer")
	delivery := &captureOTP{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), delivery)
	otp.Configure(cfg.OTPTTL, cfg.OTPMaxAttempts, cfg.OTPResendInterval)
	c.Auth.SetOTP(otp)
	handler := routes.NewHandler(c)
	login := request(handler, "POST", "/login", url.Values{"phone": {"08010000001"}, "password": {"test password with spaces"}}, nil)
	requireStatus(t, login, http.StatusSeeOther)
	if len(login.Result().Cookies()) != 1 {
		t.Fatal("login did not set a session cookie")
	}
	cookie := login.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.MaxAge != int(cfg.SessionTTL.Seconds()) {
		t.Fatalf("incorrect cookie protection/expiry: %+v", cookie)
	}
	restarted := application(t, db, cfg)
	id, err := restarted.Sessions.UserID(context.Background(), cookie.Value)
	if err != nil || id != u.ID {
		t.Fatalf("session did not survive new application instance: %d %v", id, err)
	}
	if _, err := c.Sessions.Create(context.Background(), 999999); err == nil {
		t.Fatal("session created for missing user")
	}
	reset := url.Values{"phone": {u.Phone}, "new_password": {"replacement password"}, "confirm_password": {"replacement password"}}
	request(handler, "POST", "/forgot-password", reset, nil)
	if _, err := c.Auth.Login(u.Phone, "test password with spaces"); err != nil {
		t.Fatalf("password changed without code: %v", err)
	}
	request(handler, "POST", "/forgot-password", url.Values{"phone": {u.Phone}, "action": {"send_code"}}, nil)
	if delivery.message.UserID != u.ID || len(delivery.message.Code) != 6 {
		t.Fatalf("OTP was not delivered to the account: %+v", delivery.message)
	}
	reset.Set("code", "wrong")
	request(handler, "POST", "/forgot-password", reset, nil)
	if _, err := c.Auth.Login(u.Phone, "test password with spaces"); err != nil {
		t.Fatal("wrong code changed password")
	}
	reset.Set("code", delivery.message.Code)
	request(handler, "POST", "/forgot-password", reset, nil)
	if _, err := c.Auth.Login(u.Phone, "replacement password"); err != nil {
		t.Fatalf("valid reset failed: %v", err)
	}
	if id, err := c.Sessions.UserID(context.Background(), cookie.Value); err != nil || id != 0 {
		t.Fatalf("password reset did not revoke old session: %d %v", id, err)
	}
	if err := c.Auth.ResetPassword(context.Background(), u.Phone, "third password", delivery.message.Code); err == nil {
		t.Fatal("used OTP was accepted again")
	}
	newCookie := session(t, c, u)
	r := httptest.NewRequest("POST", "http://example.test/profile/edit", strings.NewReader("full_name=Changed"))
	r.AddCookie(newCookie)
	r.Header.Set("Origin", "https://attacker.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	requireStatus(t, w, http.StatusForbidden)
	if _, err := db.Exec(`UPDATE user_sessions SET created_at=CURRENT_TIMESTAMP-INTERVAL '2 days',expires_at=CURRENT_TIMESTAMP-INTERVAL '1 day' WHERE user_id=$1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if id, err := c.Sessions.UserID(context.Background(), newCookie.Value); err != nil || id != 0 {
		t.Fatal("expired session was accepted")
	}
}

func TestLearningFromHTTPThroughProviderAndPostgres(t *testing.T) {
	db, _ := testutil.Database(t, true)
	var providerRequests int
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerRequests++
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-only-key" {
			t.Errorf("wrong provider request")
		}
		var body struct {
			Model    string                           `json:"model"`
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "test-learning-model" || len(body.Messages) < 2 || !strings.Contains(body.Messages[0].Content, "Test learning goal") {
			t.Errorf("missing saved learning context: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": "Test provider answer about the learner's question."}}}})
	}))
	defer provider.Close()
	cfg := testConfig(t)
	cfg.GroqAPIKey = "test-only-key"
	cfg.GroqModel = "test-learning-model"
	cfg.GroqBaseURL = provider.URL
	c := application(t, db, cfg)
	learner := fixtureUser(t, c, 1, "learner")
	other := fixtureUser(t, c, 2, "learner")
	types, err := c.Learning.GetFarmingTypes()
	if err != nil || len(types) < 2 {
		t.Fatalf("farming types missing: %v", err)
	}
	journey, err := c.Learning.StartLearningJourney(learner.ID, types[0].ID, "some_experience", "Test learning goal")
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Learning.StartLearningJourney(learner.ID, types[1].ID, "beginner", "")
	if err != nil || second.ID == journey.ID {
		t.Fatalf("multiple farming interests not supported: %v", err)
	}
	again, err := c.Learning.StartLearningJourney(learner.ID, types[0].ID, "beginner", "")
	if err != nil || again.ID != journey.ID {
		t.Fatal("starting the same journey duplicated it")
	}
	if _, err := c.Learning.GetLearningMessages(other.ID, journey.ID); !errors.Is(err, repository.ErrLearningNotFound) {
		t.Fatalf("another learner read private history: %v", err)
	}
	step, err := c.Learning.CreateLearningCheckpoint(learner.ID, journey.ID, "", "Check the water source", "")
	if err != nil {
		t.Fatal(err)
	}
	step.Status = "in_progress"
	step.Progress = 50
	if err := c.Learning.UpdateLearningCheckpoint(other.ID, step); err == nil {
		t.Fatal("another learner edited a checkpoint")
	}
	if err := c.Learning.UpdateLearningCheckpoint(learner.ID, step); err != nil {
		t.Fatal(err)
	}
	stored, err := c.Learning.GetJourneyForUser(learner.ID, journey.ID)
	if err != nil || stored.ReadinessScore != 50 {
		t.Fatalf("progress not based on saved steps: %+v %v", stored, err)
	}
	if err := c.Learning.ChangeJourneyStatus(learner.ID, journey.ID, "completed"); err == nil {
		t.Fatal("incomplete journey marked completed")
	}
	handler := routes.NewHandler(c)
	cookie := session(t, c, learner)
	requireStatus(t, request(handler, "GET", "/dashboard", nil, cookie), http.StatusSeeOther)
	requireStatus(t, request(handler, "GET", "/learn", nil, cookie), http.StatusOK)
	requireStatus(t, request(handler, "GET", fmt.Sprintf("/learn/journey?id=%d", journey.ID), nil, cookie), http.StatusOK)
	requireStatus(t, request(handler, "POST", "/learn/message", url.Values{"journey_id": {strconv.Itoa(journey.ID)}, "content": {"What should I do next?"}}, cookie), http.StatusSeeOther)
	if providerRequests != 1 {
		t.Fatalf("expected a real HTTP provider request, got %d", providerRequests)
	}
	reloaded := services.NewLearningService(repository.NewLearningRepository(db))
	messages, err := reloaded.GetLearningMessages(learner.ID, journey.ID)
	if err != nil || len(messages) != 2 || messages[1].SenderType != "ai" {
		t.Fatalf("conversation not saved: %+v %v", messages, err)
	}
	step.Status = "completed"
	if err := c.Learning.UpdateLearningCheckpoint(learner.ID, step); err != nil {
		t.Fatal(err)
	}
	if err := c.Learning.ChangeJourneyStatus(learner.ID, journey.ID, "completed"); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", fmt.Sprintf("/learn/journey?id=%d", journey.ID), nil, cookie), http.StatusOK)
	roomID, err := c.Learning.CreateRoom(context.Background(), learner.ID, types[0].ID, "Test farming room", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Chat.GetMessages(roomID, other.ID); err == nil {
		t.Fatal("non-member read room messages")
	}
	if err := c.Learning.JoinRoom(context.Background(), other.ID, roomID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Chat.GetMessages(roomID, other.ID); err != nil {
		t.Fatalf("joined member cannot read room: %v", err)
	}
	requireStatus(t, request(handler, "GET", "/learn/rooms", nil, cookie), http.StatusOK)
}

func TestPriceReportsPersistAndUseAuthenticatedIdentity(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	u := fixtureUser(t, c, 1, "farmer")
	other := fixtureUser(t, c, 2, "farmer")
	r := httptest.NewRequest("POST", "http://example.test/api/v1/prices/report", strings.NewReader(fmt.Sprintf(`{"commodity_id":"test-maize","market_id":"test-market","submitter_id":"%d","price":123.45,"unit":"kg","currency":"NGN"}`, other.ID)))
	r.Header.Set("Origin", "http://example.test")
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(session(t, c, u))
	w := httptest.NewRecorder()
	routes.NewHandler(c).ServeHTTP(w, r)
	requireStatus(t, w, http.StatusOK)
	var result struct {
		ReportID string `json:"report_id"`
	}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	prices := repository.NewPostgresPriceRepository(db)
	report, err := prices.GetReport(context.Background(), result.ReportID)
	if err != nil || report.SubmitterID != strconv.Itoa(u.ID) || report.Status != models.StatusPending {
		t.Fatalf("wrong saved identity or initial review status: %+v %v", report, err)
	}
	if _, err := c.Price.ReviewReport(context.Background(), report.ID, models.StatusAccepted, "checked in test"); err != nil {
		t.Fatal(err)
	}
	restarted := application(t, db, testConfig(t))
	aggregate, err := restarted.Price.GetCurrentPrice(context.Background(), "test-maize", "test-market")
	if err != nil || aggregate.SampleSize != 1 || aggregate.AvgPrice != 123.45 || aggregate.Unit != "kg" {
		t.Fatalf("saved prices not used after restart: %+v %v", aggregate, err)
	}
	_, err = c.Price.SubmitPrice(context.Background(), services.SubmitPriceInput{CommodityID: "test-maize", MarketID: "test-market", SubmitterID: strconv.Itoa(u.ID), Price: 20, Currency: "NGN", Unit: "bag"})
	if err == nil {
		t.Fatal("different units mixed in the same price series")
	}
	if err := database.RunMigration(db); err != nil {
		t.Fatalf("repeat migration failed: %v", err)
	}
}
