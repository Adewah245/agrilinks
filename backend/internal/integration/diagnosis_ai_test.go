package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
)

type diagnosisAdviceProvider struct {
	input models.AIRequest
	err   error
}

func (p *diagnosisAdviceProvider) Analyze(_ context.Context, input models.AIRequest) (*models.DiagnosisResult, error) {
	p.input = input
	if p.err != nil {
		return nil, p.err
	}
	return &models.DiagnosisResult{Result: "Check the soil. Add water only if it feels dry."}, nil
}

func TestDiagnosisCategoriesJSONAndPictureOwnership(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	owner := fixtureUser(t, c, 1, "farmer")
	other := fixtureUser(t, c, 2, "farmer")
	cookie := session(t, c, owner)
	otherCookie := session(t, c, other)
	provider := &diagnosisAdviceProvider{}
	images := &examplePictureProvider{}
	c.AI = services.NewAIService(repository.NewDiagnosisRepository(db), provider)
	c.AI.SetImageProvider(images)
	handler := routes.NewHandler(c)
	categories := services.DiagnosisCategories()
	if len(categories) != 19 {
		t.Fatal("UI farming categories missing")
	}
	var result struct {
		Result   string `json:"result"`
		ID       int    `json:"diagnosis_id"`
		Examples bool   `json:"examples_available"`
	}
	for _, category := range categories {
		t.Run(category, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			_ = writer.WriteField("category", category)
			_ = writer.WriteField("description", "Why are the leaves yellow?")
			_ = writer.Close()
			r := httptest.NewRequest("POST", "http://example.test/ai-assistant", &body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Origin", "http://example.test")
			r.AddCookie(cookie)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			requireStatus(t, w, http.StatusOK)
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.ID <= 0 || result.Result == "" || !result.Examples || provider.input.Category != category {
				t.Fatal("diagnosis response or category missing")
			}
		})
	}
	for _, test := range []struct {
		name   string
		cookie *http.Cookie
		status int
	}{
		{"owner", cookie, http.StatusOK}, {"other user", otherCookie, http.StatusNotFound}, {"guest", nil, http.StatusSeeOther},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://example.test/ai-assistant/example", strings.NewReader(fmt.Sprintf(`{"diagnosis_id":%d}`, result.ID)))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", "http://example.test")
			if test.cookie != nil {
				r.AddCookie(test.cookie)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			requireStatus(t, w, test.status)
		})
	}
	if images.calls != 1 || !images.input.Diagnosis || images.input.Question != "Why are the leaves yellow?" || images.input.Answer != result.Result {
		t.Fatal("picture request leaked or lost owned diagnosis")
	}
	// A failed provider must produce a retryable JSON error, without saving advice.
	provider.err = errors.New("The farming assistant is busy. Please try again.")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("category", "Crops")
	_ = writer.WriteField("description", "yellow leaves")
	_ = writer.Close()
	r := httptest.NewRequest("POST", "http://example.test/ai-assistant", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Origin", "http://example.test")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	requireStatus(t, w, http.StatusServiceUnavailable)
	if !strings.Contains(w.Body.String(), "Please try again") {
		t.Fatal("provider failure did not offer retry")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM diagnoses WHERE user_id=$1", owner.ID).Scan(&count); err != nil || count != 19 {
		t.Fatalf("failed advice saved: count=%d error=%v", count, err)
	}
}
