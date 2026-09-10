package handlers

import (
	"net/http/httptest"
	"testing"

	"backend/render"
)

// TestDashboardTemplateHasAllRequiredDataFields prevents template-only fields
// from reaching production without a corresponding DashboardData field.
func TestDashboardTemplateHasAllRequiredDataFields(t *testing.T) {
	response := httptest.NewRecorder()

	data := DashboardData{
		FullName:     "Test Farmer",
		FarmingTypes: []string{"Crop Farming"},
		HealthMetrics: []HealthMetric{
			{}, {}, {}, {},
		},
	}

	if err := render.RenderTemplates(response, "dashboard.html", data); err != nil {
		t.Fatalf("dashboard template should render: %v", err)
	}
}
