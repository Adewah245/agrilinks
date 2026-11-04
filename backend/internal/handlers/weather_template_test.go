package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/testui"
)

func TestDashboardWeatherWaitsForDeviceLocation(t *testing.T) {
	for _, buyer := range []bool{false, true} {
		response := httptest.NewRecorder()
		data := DashboardData{IsBuyer: buyer, FullName: "Location check", HealthMetrics: make([]HealthMetric, 4), Weather: models.Weather{Available: true, Location: "Kogi State", Temperature: "39°C"}}
		if err := render.RenderPage(response, "dashboard.html", data); err != nil {
			t.Fatal(err)
		}
		body := testui.HTML(t, response.Body.String())
		if strings.Count(body, "data-device-weather") != 1 {
			t.Fatal("dashboard must render one device weather card")
		}
		for _, expected := range []string{"Allow location access", "data-weather-refresh", `data-weather-values="" hidden=""`} {
			if !strings.Contains(body, expected) {
				t.Errorf("missing device location flow %q", expected)
			}
		}
		for _, unwanted := range []string{"Kogi State", "39°C", "Weather for the state capital", "Update location"} {
			if strings.Contains(body, unwanted) {
				t.Errorf("dashboard still displays saved-location weather: %q", unwanted)
			}
		}
	}
}
