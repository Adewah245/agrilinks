package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
)

type fakeCoordinateWeather struct {
	called              bool
	latitude, longitude float64
	weather             models.Weather
	err                 error
}

func (f *fakeCoordinateWeather) CurrentAtCoordinates(_ context.Context, latitude, longitude float64) (models.Weather, error) {
	f.called, f.latitude, f.longitude = true, latitude, longitude
	return f.weather, f.err
}
func TestCurrentWeatherPassesDeviceCoordinates(t *testing.T) {
	source := &fakeCoordinateWeather{weather: models.Weather{Available: true, Location: "Your current location", Temperature: "28°C"}}
	handler := &Dashboard{weather: source}
	request := httptest.NewRequest(http.MethodPost, "/api/weather/current", strings.NewReader(`{"latitude":7.7322,"longitude":8.5391,"accuracy":25}`))
	response := httptest.NewRecorder()
	handler.CurrentWeather(response, request)
	if response.Code != 200 || !source.called || source.latitude != 7.7322 || source.longitude != 8.5391 {
		t.Fatalf("response=%d %s, source=%+v", response.Code, response.Body.String(), source)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("weather response must not be cached by the browser")
	}
}
func TestCurrentWeatherRejectsMissingOrImpreciseLocation(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"latitude":7.7,"longitude":8.5}`, `{"latitude":91,"longitude":8.5,"accuracy":25}`,
		`{"latitude":7.7,"longitude":181,"accuracy":25}`, `{"latitude":7.7,"longitude":8.5,"accuracy":-1}`,
		`{"latitude":7.7,"longitude":8.5,"accuracy":50000}`, `{"latitude":null,"longitude":8.5,"accuracy":25}`,
		`{"latitude":7.7,"longitude":8.5,"accuracy":25,"state":"Kogi"}`, `{"latitude":7.7,"longitude":8.5,"accuracy":25} {}`,
	} {
		t.Run(body, func(t *testing.T) {
			source := &fakeCoordinateWeather{}
			response := httptest.NewRecorder()
			(&Dashboard{weather: source}).CurrentWeather(response, httptest.NewRequest(http.MethodPost, "/api/weather/current", strings.NewReader(body)))
			if response.Code != 400 || source.called {
				t.Fatalf("response=%d, provider called=%t", response.Code, source.called)
			}
		})
	}
}
func TestCurrentWeatherOutageDoesNotExposeProviderDetails(t *testing.T) {
	source := &fakeCoordinateWeather{err: errors.New("internal provider URL and private coordinates")}
	response := httptest.NewRecorder()
	(&Dashboard{weather: source}).CurrentWeather(response, httptest.NewRequest(http.MethodPost, "/api/weather/current", strings.NewReader(`{"latitude":7.7,"longitude":8.5,"accuracy":25}`)))
	if response.Code != 503 || strings.Contains(response.Body.String(), "private coordinates") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}
