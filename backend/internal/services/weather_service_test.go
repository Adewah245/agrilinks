package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

type weatherRoundTrip func(*http.Request) (*http.Response, error)

func (f weatherRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func weatherTestService(t *testing.T, respond func(*http.Request) (int, any)) *WeatherService {
	t.Helper()
	service := NewWeatherService()
	service.client.Transport = weatherRoundTrip(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > weatherRequestTimeout {
			t.Error("weather request has no bounded total deadline")
		}
		status, body := respond(r)
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(string(encoded))), Header: make(http.Header)}, nil
	})
	return service
}

func weatherTestForecast() map[string]any {
	now := time.Now().Truncate(time.Minute)
	return map[string]any{
		"utc_offset_seconds": 3600, "timezone_abbreviation": "WAT",
		"current": map[string]any{"time": now.Unix(), "temperature_2m": 28, "relative_humidity_2m": 80, "weather_code": 61},
		"hourly":  map[string]any{"time": []int64{now.Truncate(time.Hour).Unix()}, "precipitation_probability": []any{75}},
	}
}

func weatherTestPlace(name, state, lga string) map[string]any {
	return map[string]any{"name": name, "admin1": state, "admin2": lga, "country_code": "NG", "latitude": 7.68967, "longitude": 7.0438}
}

func TestWeatherLocationCandidatesUseStructuredFallbacks(t *testing.T) {
	got := weatherLocationCandidates("Obaganya", "Dekina", "Kogi", "Nigeria")
	want := []string{"Obaganya", "Obaganya, Dekina, Kogi, Nigeria", "Dekina, Kogi, Nigeria", "Lokoja, Kogi, Nigeria"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
}

func TestWeatherLocationCandidatesRemoveDuplicates(t *testing.T) {
	got := weatherLocationCandidates("Kogi", "", "Kogi", "Nigeria")
	want := []string{"Kogi", "Kogi, Nigeria", "Lokoja, Kogi, Nigeria"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
}

func TestWeatherMatchesCountryStateAndLGA(t *testing.T) {
	calls := 0
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		calls++
		if strings.Contains(r.URL.Host, "geocoding") {
			if r.URL.Query().Get("countryCode") != "NG" {
				t.Error("geocoding must be restricted to Nigeria")
			}
			foreign := weatherTestPlace("Dekina", "Kogi State", "Dekina")
			foreign["country_code"] = "CM"
			wrong := weatherTestPlace("Dekina", "Zamfara State", "Dekina")
			wrongLGA := weatherTestPlace("Dekina", "Kogi State", "Lokoja")
			valid := weatherTestPlace("Dekina", "Kogi State", "Dekina")
			valid["latitude"] = 7.7
			return 200, map[string]any{"results": []any{foreign, wrong, wrongLGA, valid}}
		}
		if r.URL.Query().Get("latitude") != "7.7000" {
			t.Error("forecast requested for the wrong place")
		}
		if r.URL.Query().Get("timeformat") != "unixtime" {
			t.Error("forecast times must be timezone independent")
		}
		return 200, weatherTestForecast()
	})
	weather, err := service.CurrentForLocation(context.Background(), "Dekina", "Dekina", "Kogi", "Nigeria")
	if err != nil || !weather.Available || weather.Location != "Dekina, Kogi State, Nigeria" || weather.Icon != "🌧" || weather.RainChance != "75%" {
		t.Fatalf("weather = %+v, err = %v", weather, err)
	}
	if zone, offset := weather.LastUpdated.Zone(); zone != "WAT" || offset != 3600 {
		t.Fatalf("observation timezone = %s/%d", zone, offset)
	}
	cached, err := service.CurrentForLocation(context.Background(), "Dekina", "Dekina", "Kogi", "Nigeria")
	if err != nil || !cached.Cached || cached.Stale || calls != 2 {
		t.Fatalf("fresh cache failed: %+v, calls=%d, err=%v", cached, calls, err)
	}
}

func TestWeatherFallsBackToRecognizedLGA(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		if strings.Contains(r.URL.Host, "geocoding") {
			if strings.HasPrefix(r.URL.Query().Get("name"), "Obaganya") {
				return 200, map[string]any{}
			}
			return 200, map[string]any{"results": []any{weatherTestPlace("Dekina", "Kogi State", "Dekina")}}
		}
		return 200, weatherTestForecast()
	})
	weather, err := service.CurrentForLocation(context.Background(), "Obaganya", "Dekina", "Kogi", "Nigeria")
	if err != nil || !weather.Approximate || weather.Location != "Dekina, Kogi State, Nigeria" {
		t.Fatalf("fallback = %+v, err=%v", weather, err)
	}
}

func TestWeatherStateFallbackDoesNotUseWrongKogiTown(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		if strings.Contains(r.URL.Host, "geocoding") {
			if r.URL.Query().Get("name") == "Lokoja, Kogi, Nigeria" {
				return 200, map[string]any{"results": []any{weatherTestPlace("Kogi", "Zamfara State", "Kaura Namoda"), weatherTestPlace("Lokoja", "Kogi State", "Lokoja")}}
			}
			return 200, map[string]any{}
		}
		return 200, weatherTestForecast()
	})
	weather, err := service.CurrentForLocation(context.Background(), "Unknown village", "Unknown LGA", "Kogi", "Nigeria")
	if err != nil || !weather.Approximate || weather.Location != "Lokoja, Kogi State, Nigeria" {
		t.Fatalf("state fallback = %+v, err=%v", weather, err)
	}
}

func TestWeatherCacheDoesNotMixStates(t *testing.T) {
	forecasts := 0
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		if strings.Contains(r.URL.Host, "geocoding") {
			return 200, map[string]any{"results": []any{weatherTestPlace("Oke", "Ogun State", ""), weatherTestPlace("Oke", "Osun State", "")}}
		}
		forecasts++
		return 200, weatherTestForecast()
	})
	for _, state := range []string{"Ogun", "Osun"} {
		weather, err := service.CurrentForLocation(context.Background(), "Oke", "", state, "Nigeria")
		if err != nil || !strings.Contains(weather.Location, state) {
			t.Fatalf("state %s: %+v, %v", state, weather, err)
		}
	}
	if forecasts != 2 {
		t.Fatalf("forecasts = %d; state-specific cache entries required", forecasts)
	}
}

func TestWeatherPreservesRecentCacheDuringProviderOutage(t *testing.T) {
	fail := false
	calls := 0
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		calls++
		if fail {
			return 503, map[string]any{}
		}
		if strings.Contains(r.URL.Host, "geocoding") {
			return 200, map[string]any{"results": []any{weatherTestPlace("Dekina", "Kogi State", "Dekina")}}
		}
		return 200, weatherTestForecast()
	})
	first, err := service.CurrentForLocation(context.Background(), "Obaganya", "Dekina", "Kogi", "Nigeria")
	if err != nil {
		t.Fatal(err)
	}
	for key, entry := range service.cache {
		entry.fetchedAt = time.Now().Add(-20 * time.Minute)
		service.cache[key] = entry
	}
	fail = true
	calls = 0
	stale, err := service.CurrentForLocation(context.Background(), "Obaganya", "Dekina", "Kogi", "Nigeria")
	if err == nil || !stale.Available || !stale.Cached || !stale.Stale || stale.Error == "" || stale.Temperature != first.Temperature || calls != 1 {
		t.Fatalf("stale cache = %+v, err=%v, requests=%d", stale, err, calls)
	}
	for key, entry := range service.cache {
		entry.weather.LastUpdated = time.Now().Add(-2 * time.Hour)
		service.cache[key] = entry
	}
	expired, err := service.CurrentForLocation(context.Background(), "Obaganya", "Dekina", "Kogi", "Nigeria")
	if err == nil || expired.Available || expired.NeedsLocation || expired.Error == "" {
		t.Fatalf("expired cache = %+v, err=%v", expired, err)
	}
}

func TestWeatherMissingLocationDoesNotQueryNigeria(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		t.Fatal("country-only location must not query weather")
		return 200, nil
	})
	weather, err := service.CurrentForLocation(context.Background(), "", "", "", "Nigeria")
	if !errors.Is(err, errWeatherLocationRequired) || !weather.NeedsLocation || weather.Available {
		t.Fatalf("weather = %+v, err=%v", weather, err)
	}
}

func TestWeatherRejectsIncompleteCurrentConditions(t *testing.T) {
	for _, field := range []string{"time", "temperature_2m", "relative_humidity_2m", "weather_code"} {
		t.Run(field, func(t *testing.T) {
			service := weatherTestService(t, func(r *http.Request) (int, any) {
				if strings.Contains(r.URL.Host, "geocoding") {
					return 200, map[string]any{"results": []any{weatherTestPlace("Lagos", "Lagos", "")}}
				}
				data := weatherTestForecast()
				data["current"].(map[string]any)[field] = nil
				return 200, data
			})
			weather, err := service.Current(context.Background(), "Lagos")
			if err == nil || weather.Available || weather.NeedsLocation {
				t.Fatalf("invalid %s accepted: %+v, err=%v", field, weather, err)
			}
		})
	}
}

func TestWeatherMissingRainProbabilityIsNotZeroPercent(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		if strings.Contains(r.URL.Host, "geocoding") {
			return 200, map[string]any{"results": []any{weatherTestPlace("Lagos", "Lagos", "")}}
		}
		data := weatherTestForecast()
		data["hourly"].(map[string]any)["precipitation_probability"] = []any{nil}
		return 200, data
	})
	weather, err := service.Current(context.Background(), "Lagos")
	if err != nil || !weather.Available || weather.RainChance != "—" {
		t.Fatalf("weather = %+v, err=%v", weather, err)
	}
}

func TestCurrentPrecipitationProbabilityUsesObservationHour(t *testing.T) {
	hour := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	midnight, current := 5.0, 80.0
	timestamps := []int64{hour.Unix(), hour.Add(time.Hour).Unix()}
	rain := []*float64{&midnight, &current}
	observed := hour.Add(75 * time.Minute).In(time.FixedZone("other server timezone", -7*3600))
	if got, ok := currentPrecipitationProbability(timestamps, rain, observed); !ok || got != current {
		t.Fatalf("probability = %v, available=%v", got, ok)
	}
	if got, ok := currentPrecipitationProbability(timestamps, rain, hour.Add(5*time.Hour)); ok {
		t.Fatalf("unmatched time must not use midnight probability: %v", got)
	}
}

func TestWeatherRecognizesAbujaMunicipalAreaCouncil(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		if strings.Contains(r.URL.Host, "geocoding") {
			return 200, map[string]any{"results": []any{weatherTestPlace("Abuja", "FCT", "Abaji"), weatherTestPlace("Abuja", "FCT", "Municipal Area Council")}}
		}
		return 200, weatherTestForecast()
	})
	weather, err := service.CurrentForLocation(context.Background(), "Abuja", "Abuja Municipal", "Federal Capital Territory", "Nigeria")
	if err != nil || !weather.Available || weather.Approximate || weather.Location != "Abuja, FCT, Nigeria" {
		t.Fatalf("Abuja weather = %+v, err=%v", weather, err)
	}
}

func TestWeatherCancellationStopsFallbackRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := weatherTestService(t, func(r *http.Request) (int, any) { t.Fatal("cancelled context made a request"); return 200, nil })
	weather, err := service.CurrentForLocation(ctx, "Obaganya", "Dekina", "Kogi", "Nigeria")
	if !errors.Is(err, context.Canceled) || weather.Available || weather.NeedsLocation {
		t.Fatalf("cancelled weather = %+v, err=%v", weather, err)
	}
}

func TestDeviceWeatherUsesCoordinatesWithoutGeocoding(t *testing.T) {
	requests := 0
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		requests++
		if r.URL.Host != "api.open-meteo.com" {
			t.Fatalf("device weather attempted a name lookup: %s", r.URL.Host)
		}
		if r.URL.Query().Get("latitude") != "7.7322" || r.URL.Query().Get("longitude") != "8.5391" {
			t.Fatalf("wrong coordinates: %s", r.URL.RawQuery)
		}
		return 200, weatherTestForecast()
	})
	for i := 0; i < 2; i++ {
		weather, err := service.CurrentAtCoordinates(context.Background(), 7.7322, 8.5391)
		if err != nil || !weather.Available || weather.Approximate || weather.Location != "Your current location" {
			t.Fatalf("device weather=%+v, err=%v", weather, err)
		}
	}
	if requests != 1 {
		t.Fatalf("same-location cache made %d requests", requests)
	}
}

func TestDeviceWeatherMovingBetweenStatesDoesNotReuseOldWeather(t *testing.T) {
	fail := false
	requests := 0
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		requests++
		if strings.Contains(r.URL.Host, "geocoding") {
			t.Fatal("device coordinates must not be geocoded")
		}
		if fail {
			return 503, map[string]any{}
		}
		data := weatherTestForecast()
		if r.URL.Query().Get("longitude") == "6.7333" {
			data["current"].(map[string]any)["temperature_2m"] = 35
		}
		return 200, data
	})
	benue, err := service.CurrentAtCoordinates(context.Background(), 7.7322, 8.5391)
	if err != nil {
		t.Fatal(err)
	}
	kogi, err := service.CurrentAtCoordinates(context.Background(), 7.8024, 6.7333)
	if err != nil || kogi.Temperature == benue.Temperature || requests != 2 {
		t.Fatalf("moved weather=%+v, old=%+v, requests=%d err=%v", kogi, benue, requests, err)
	}
	fail = true
	other, err := service.CurrentAtCoordinates(context.Background(), 7.4, 8.1)
	if err == nil || other.Available {
		t.Fatalf("new location reused another place's weather: %+v %v", other, err)
	}
}

func TestDeviceWeatherRejectsInvalidCoordinates(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) { t.Fatal("invalid coordinates reached provider"); return 200, nil })
	for _, point := range [][2]float64{{91, 8}, {7, 181}, {-91, 8}, {7, -181}} {
		weather, err := service.CurrentAtCoordinates(context.Background(), point[0], point[1])
		if err == nil || weather.Available {
			t.Fatalf("accepted coordinates %v", point)
		}
	}
}

func TestDeviceWeatherUsesNightIconAndProviderTimezone(t *testing.T) {
	service := weatherTestService(t, func(r *http.Request) (int, any) {
		data := weatherTestForecast()
		data["utc_offset_seconds"] = 7200
		data["timezone_abbreviation"] = "EET"
		current := data["current"].(map[string]any)
		current["weather_code"], current["is_day"] = 0, 0
		return 200, data
	})
	weather, err := service.CurrentAtCoordinates(context.Background(), 30.0, 31.0)
	if err != nil || weather.Icon != "☾" {
		t.Fatalf("night weather=%+v, err=%v", weather, err)
	}
	if _, offset := weather.LastUpdated.Zone(); offset != 7200 {
		t.Fatalf("wrong observation timezone: %s", weather.LastUpdated)
	}
}
