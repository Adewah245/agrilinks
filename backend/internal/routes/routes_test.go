package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"backend/internal/app"
)

func TestDeviceWeatherRequiresLogin(t *testing.T) {
	handler := NewHandler(&app.Container{})
	request := httptest.NewRequest(http.MethodPost, "/api/weather/current", strings.NewReader(`{"latitude":7.7322,"longitude":8.5391,"accuracy":25}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated weather request: %d %s", response.Code, response.Body.String())
	}
}

func TestDashboardLinksHaveRegisteredRoutes(t *testing.T) {
	content, err := os.ReadFile("../../../frontend/pages/dashboard.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(content)
	for _, href := range []string{"/dashboard", "/storage", "/marketplace", "/market-insights", "/cart", "/ai-assistant", "/ai-diagnosis-history", "/profile", "/logout"} {
		if !strings.Contains(page, `href="`+href+`"`) {
			t.Errorf("dashboard does not contain expected link %s", href)
		}
	}
}

// TestPageStaticAssetsExist catches broken CSS, JavaScript, image, and upload
// links before a browser reaches a page that references them.
func TestPageStaticAssetsExist(t *testing.T) {
	pages, err := filepath.Glob("../../../frontend/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}

	staticURL := regexp.MustCompile(`/(?:static)/[^"'\s?]+`)

	for _, page := range pages {
		content, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}

		for _, url := range staticURL.FindAllString(string(content), -1) {
			asset := filepath.Join("../../..", "frontend", strings.TrimPrefix(url, "/static/"))
			if _, err := os.Stat(asset); err != nil {
				t.Errorf("%s references missing static asset %s", page, url)
			}
		}
	}
}
