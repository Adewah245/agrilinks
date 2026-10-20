package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/render"
)

func TestAIDiagnosisHistoryTemplateUsesSymptoms(t *testing.T) {
	category := "Crops"
	symptoms := "Yellow leaf spots"
	result := "Leaf rust"

	response := httptest.NewRecorder()
	data := AIDiagnosisHistoryPageData{
		History: []models.Diagnosis{
			{
				ID:        1,
				Category:  &category,
				Symptoms:  &symptoms,
				Result:    &result,
				CreatedAt: time.Now(),
			},
		},
	}

	if err := render.RenderTemplates(response, "ai-diagnosis-history.html", data); err != nil {
		t.Fatalf("diagnosis history template should render: %v", err)
	}

	body := response.Body.String()
	if !strings.Contains(body, symptoms) {
		t.Fatal("diagnosis history template did not render symptoms")
	}
}
