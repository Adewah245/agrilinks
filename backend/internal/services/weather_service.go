package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/internal/models"
)

// ============================================================
// WEATHER SERVICE
//
// Responsibility:
// - Retrieves current weather information.
// - Converts a user location into geographic coordinates.
// - Communicates with Open-Meteo.
// - Caches weather responses for a short period.
// - Provides farming-oriented recommendations.
//
// The service does not communicate with PostgreSQL.
// Weather data comes from the external Open-Meteo API.
//
// Important:
// Weather information is advisory. It should help farmers make
// decisions but should not be presented as a guarantee of safe
// farming conditions.
// ============================================================

// cachedWeather stores weather data together with the time at
// which the data was fetched.
type cachedWeather struct {
	weather   models.Weather
	fetchedAt time.Time
}

// WeatherService retrieves and caches weather information.
type WeatherService struct {
	client *http.Client

	mu    sync.RWMutex
	cache map[string]cachedWeather
}

// NewWeatherService creates a new WeatherService.
//
// A dedicated HTTP client is used so that weather requests have
// a controlled timeout and do not depend on the default HTTP client.
func NewWeatherService() *WeatherService {
	return &WeatherService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]cachedWeather),
	}
}

// Current returns current weather information for a location.
//
// The location is first geocoded through Open-Meteo's geocoding
// service. The resulting coordinates are then used to retrieve
// current weather conditions.
//
// Successfully retrieved weather data is cached for 15 minutes.
//
// If a fresh API request fails but previously cached data exists,
// the cached weather is returned together with the API error so
// callers know that the information is stale.
func (s *WeatherService) Current(
	ctx context.Context,
	location string,
) (models.Weather, error) {
	location = strings.TrimSpace(location)

	if location == "" {
		return models.Weather{}, errors.New(
			"weather location is required",
		)
	}

	cacheKey := normalizeWeatherLocation(location)

	if cached, ok := s.cached(cacheKey); ok {
		if time.Since(cached.fetchedAt) < 15*time.Minute {
			weather := cached.weather
			weather.Cached = true

			return weather, nil
		}
	}

	weather, err := s.fetch(ctx, location)

	if err != nil {
		// A stale cached response is still useful as a fallback,
		// but Cached is marked true so the caller knows it is not
		// freshly retrieved.
		if cached, ok := s.cached(cacheKey); ok {
			fallback := cached.weather
			fallback.Cached = true

			return fallback, err
		}

		return models.Weather{}, err
	}

	s.mu.Lock()

	s.cache[cacheKey] = cachedWeather{
		weather:   weather,
		fetchedAt: weather.LastUpdated,
	}

	s.mu.Unlock()

	return weather, nil
}

// cached retrieves weather data from the in-memory cache.
//
// The read lock allows multiple weather requests to read the
// cache concurrently.
func (s *WeatherService) cached(
	key string,
) (cachedWeather, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	weather, ok := s.cache[key]

	return weather, ok
}

// fetch retrieves weather data from Open-Meteo.
//
// The location is first converted to coordinates using the
// Open-Meteo geocoding endpoint. Those coordinates are then
// supplied to the forecast endpoint.
func (s *WeatherService) fetch(
	ctx context.Context,
	location string,
) (models.Weather, error) {
	latitude, longitude, err := s.geocode(
		ctx,
		location,
	)
	if err != nil {
		return models.Weather{}, err
	}

	// Current conditions are returned directly from Open-Meteo.
	//
	// Hourly precipitation probability is also requested because
	// it provides a useful short-term precipitation probability
	// for the farming recommendation and dashboard.
	wq := url.Values{}

	wq.Set(
		"latitude",
		strconv.FormatFloat(
			latitude,
			'f',
			4,
			64,
		),
	)

	wq.Set(
		"longitude",
		strconv.FormatFloat(
			longitude,
			'f',
			4,
			64,
		),
	)

	wq.Set(
		"current",
		"temperature_2m,relative_humidity_2m,weather_code",
	)

	wq.Set(
		"hourly",
		"precipitation_probability",
	)

	wq.Set(
		"forecast_days",
		"1",
	)

	// Open-Meteo returns times using the requested local timezone.
	wq.Set("timezone", "auto")

	var raw struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			Humidity    float64 `json:"relative_humidity_2m"`
			Code        int     `json:"weather_code"`
		} `json:"current"`

		Hourly struct {
			Time []string  `json:"time"`
			Rain []float64 `json:"precipitation_probability"`
		} `json:"hourly"`
	}

	endpoint :=
		"https://api.open-meteo.com/v1/forecast?" +
			wq.Encode()

	if err := s.getJSON(
		ctx,
		endpoint,
		&raw,
	); err != nil {
		return models.Weather{}, err
	}

	rainChance := currentPrecipitationProbability(
		raw.Hourly.Time,
		raw.Hourly.Rain,
	)

	summary := weatherSummary(
		raw.Current.Code,
	)

	return models.Weather{
		Available:      true,
		Location:       location,
		Source:         "Open-Meteo",
		Temperature:    fmt.Sprintf("%.0f°C", raw.Current.Temperature),
		Summary:        summary,
		Humidity:       fmt.Sprintf("%.0f%%", raw.Current.Humidity),
		RainChance:     fmt.Sprintf("%.0f%%", rainChance),
		Conditions:     summary,
		Recommendation: weatherRecommendation(raw.Current.Code),
		LastUpdated:    time.Now(),
		Cached:         false,
	}, nil
}

// geocode converts a human-readable location into latitude and
// longitude using the Open-Meteo geocoding API.
func (s *WeatherService) geocode(
	ctx context.Context,
	location string,
) (float64, float64, error) {
	q := url.Values{}

	q.Set("name", location)
	q.Set("count", "1")
	q.Set("language", "en")
	q.Set("format", "json")

	endpoint :=
		"https://geocoding-api.open-meteo.com/v1/search?" +
			q.Encode()

	var geo struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}

	if err := s.getJSON(
		ctx,
		endpoint,
		&geo,
	); err != nil {
		return 0, 0, err
	}

	if len(geo.Results) == 0 {
		return 0, 0, fmt.Errorf(
			"location %q not found",
			location,
		)
	}

	return geo.Results[0].Latitude,
		geo.Results[0].Longitude,
		nil
}

// currentPrecipitationProbability selects the precipitation
// probability corresponding to the current local hour.
//
// Open-Meteo's hourly response contains both timestamps and
// precipitation probabilities. Matching the current hour to
// the returned timestamp is safer than assuming that the first
// array element represents the current hour.
func currentPrecipitationProbability(
	times []string,
	probabilities []float64,
) float64 {
	if len(times) == 0 ||
		len(probabilities) == 0 {
		return 0
	}

	count := len(times)

	if len(probabilities) < count {
		count = len(probabilities)
	}

	now := time.Now()

	for i := 0; i < count; i++ {
		t, err := time.Parse(
			"2006-01-02T15:04",
			times[i],
		)

		if err != nil {
			continue
		}

		// Match the same local year, month, day and hour.
		if t.Year() == now.Year() &&
			t.Month() == now.Month() &&
			t.Day() == now.Day() &&
			t.Hour() == now.Hour() {
			return clampPercentage(
				probabilities[i],
			)
		}
	}

	// If an exact local-hour match cannot be found, use the
	// first returned value rather than failing the whole weather
	// request.
	return clampPercentage(
		probabilities[0],
	)
}

// getJSON performs a GET request and decodes the JSON response.
//
// Non-200 HTTP responses are returned as errors so callers do not
// accidentally process an unsuccessful API response as valid data.
func (s *WeatherService) getJSON(
	ctx context.Context,
	endpoint string,
	dst any,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)

	if err != nil {
		return err
	}

	req.Header.Set(
		"Accept",
		"application/json",
	)

	resp, err := s.client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"weather API returned %s",
			resp.Status,
		)
	}

	if err := json.NewDecoder(
		resp.Body,
	).Decode(dst); err != nil {
		return fmt.Errorf(
			"failed to decode weather API response: %w",
			err,
		)
	}

	return nil
}

// weatherSummary converts an Open-Meteo WMO weather code into
// a human-readable weather condition.
//
// The mapping follows the WMO weather interpretation categories
// used by Open-Meteo.
func weatherSummary(code int) string {
	switch code {
	case 0:
		return "Clear sky"

	case 1:
		return "Mainly clear"

	case 2:
		return "Partly cloudy"

	case 3:
		return "Overcast"

	case 45, 48:
		return "Foggy"

	case 51, 53, 55:
		return "Drizzle"

	case 56, 57:
		return "Freezing drizzle"

	case 61, 63, 65:
		return "Rain"

	case 66, 67:
		return "Freezing rain"

	case 71, 73, 75, 77:
		return "Snow"

	case 80, 81, 82:
		return "Rain showers"

	case 85, 86:
		return "Snow showers"

	case 95:
		return "Thunderstorm"

	case 96, 99:
		return "Thunderstorm with hail"

	default:
		return "Unknown weather conditions"
	}
}

// weatherRecommendation generates a conservative farming
// recommendation based on the current WMO weather code.
//
// The recommendation is advisory only. It should not be treated
// as a guarantee of safe weather or as a replacement for local
// weather warnings.
func weatherRecommendation(code int) string {
	switch code {
	case 0, 1:
		return "Generally suitable for routine farm activities. Continue monitoring local weather conditions."

	case 2, 3:
		return "Conditions are generally suitable for routine farm activities, but continue monitoring for changes."

	case 45, 48:
		return "Reduced visibility is possible. Take care with road travel, machinery, and field operations."

	case 51, 53, 55:
		return "Light to moderate drizzle is possible. Consider delaying moisture-sensitive work where necessary."

	case 56, 57:
		return "Freezing drizzle is reported by the weather model. Protect sensitive operations and monitor conditions."

	case 61, 63:
		return "Rain is expected. Consider delaying moisture-sensitive field work and protect harvested produce."

	case 65:
		return "Heavy rain is possible. Avoid unnecessary field operations and protect stored or harvested produce."

	case 66, 67:
		return "Freezing rain conditions are indicated. Exercise caution with outdoor work and transport."

	case 71, 73, 75, 77:
		return "Snow or wintry precipitation is indicated. Protect livestock, equipment, and sensitive agricultural activities."

	case 80, 81:
		return "Rain showers are possible. Consider postponing moisture-sensitive work and protect harvested produce."

	case 82:
		return "Heavy rain showers are possible. Consider postponing field operations and protect crops and stored produce."

	case 85, 86:
		return "Snow showers are possible. Protect sensitive agricultural activities and equipment."

	case 95:
		return "Thunderstorms are possible. Avoid exposed field work during storms and protect livestock, workers, and equipment."

	case 96, 99:
		return "Thunderstorms with hail are possible. Protect crops, livestock, workers, and equipment and avoid exposed areas."

	default:
		return "Weather conditions are uncertain. Monitor updated local weather information before important farm activities."
	}
}

// normalizeWeatherLocation creates a consistent cache key from
// a user-supplied location.
func normalizeWeatherLocation(location string) string {
	return strings.ToLower(
		strings.Join(
			strings.Fields(location),
			" ",
		),
	)
}

// clampPercentage ensures a probability value remains within
// the valid 0-100 percentage range.
func clampPercentage(value float64) float64 {
	if value < 0 {
		return 0
	}

	if value > 100 {
		return 100
	}

	return value
}
