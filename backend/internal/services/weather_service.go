package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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

const (
	weatherCacheTTL       = 15 * time.Minute
	weatherMaxAge         = time.Hour
	weatherRequestTimeout = 8 * time.Second
)

var (
	errWeatherLocationRequired = errors.New("weather location is required")
	errWeatherLocationNotFound = errors.New("weather location not found")
)

// Current retrieves weather for a Nigerian place name. Structured profile
// locations should use CurrentForLocation to disambiguate towns with the same name.
func (s *WeatherService) Current(ctx context.Context, location string) (models.Weather, error) {
	return s.CurrentForLocation(ctx, location, "", "", "Nigeria")
}

// CurrentAtCoordinates uses only the device's coordinates. Its cache is scoped
// to that point, so moving to another area cannot reuse the previous area's
// conditions or fall back to the account's saved state or LGA.
func (s *WeatherService) CurrentAtCoordinates(ctx context.Context, latitude, longitude float64) (models.Weather, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || math.Abs(latitude) > 90 ||
		math.IsNaN(longitude) || math.IsInf(longitude, 0) || math.Abs(longitude) > 180 {
		return models.Weather{}, errors.New("valid device coordinates are required")
	}
	key := fmt.Sprintf("coordinates:%.4f,%.4f", latitude, longitude)
	cached, hasCache := s.cached(key)
	if hasCache && time.Since(cached.fetchedAt) < weatherCacheTTL && time.Since(cached.weather.LastUpdated) < weatherMaxAge {
		weather := cached.weather
		weather.Cached = true
		return weather, nil
	}
	ctx, cancel := context.WithTimeout(ctx, weatherRequestTimeout)
	defer cancel()
	weather, err := s.fetchCoordinates(ctx, latitude, longitude, "Your current location")
	if err != nil {
		if hasCache && time.Since(cached.weather.LastUpdated) < weatherMaxAge {
			weather = cached.weather
			weather.Cached, weather.Stale = true, true
			weather.Error = "Live weather could not be refreshed. Showing the last update for this location."
			return weather, err
		}
		return weatherUnavailable(err), err
	}
	s.mu.Lock()
	s.cache[key] = cachedWeather{weather: weather, fetchedAt: time.Now()}
	s.mu.Unlock()
	return weather, nil
}

// CurrentForLocation checks the country, state and LGA of geocoding results.
// A single deadline covers every lookup and the forecast request, so an outage
// cannot hold the dashboard open for a separate timeout per candidate.
func (s *WeatherService) CurrentForLocation(ctx context.Context, location, lga, state, country string) (models.Weather, error) {
	location, lga = strings.TrimSpace(location), strings.TrimSpace(lga)
	state, country = strings.TrimSpace(state), strings.TrimSpace(country)
	if country == "" {
		country = "Nigeria"
	}
	candidates := weatherLocationCandidates(location, lga, state, country)
	if len(candidates) == 0 {
		return weatherUnavailable(errWeatherLocationRequired), errWeatherLocationRequired
	}
	cacheKey := normalizeWeatherLocation(strings.Join([]string{location, lga, state, country}, "|"))
	cached, hasCache := s.cached(cacheKey)
	if hasCache && time.Since(cached.fetchedAt) < weatherCacheTTL && time.Since(cached.weather.LastUpdated) < weatherMaxAge {
		weather := cached.weather
		weather.Cached = true
		return weather, nil
	}

	ctx, cancel := context.WithTimeout(ctx, weatherRequestTimeout)
	defer cancel()
	var lastErr error
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			lastErr = err
			break
		}
		candidateLGA := lga
		// A state-level fallback cannot be required to belong to a single LGA.
		stateQuery := weatherStateQuery(state, country)
		if candidate == stateQuery {
			candidateLGA = ""
		}
		weather, err := s.fetch(ctx, candidate, candidateLGA, state, country)
		if err == nil {
			weather.Approximate = candidate != location && candidate != joinWeatherLocation(location, lga, state, country)
			if candidate == stateQuery {
				weather.Approximate = true
				weather.LocationNote = "Weather for the state capital"
			} else if weather.Approximate {
				weather.LocationNote = "Weather for your LGA"
			}
			s.mu.Lock()
			s.cache[cacheKey] = cachedWeather{weather: weather, fetchedAt: time.Now()}
			s.mu.Unlock()
			return weather, nil
		}
		lastErr = err
		// Try broader areas only when the provider could not resolve the place.
		// Repeating requests after an outage or rate limit cannot help.
		if !errors.Is(err, errWeatherLocationNotFound) {
			break
		}
	}
	if hasCache && time.Since(cached.weather.LastUpdated) < weatherMaxAge {
		weather := cached.weather
		weather.Cached, weather.Stale = true, true
		weather.Error = "Live weather could not be refreshed. Showing the last available update."
		return weather, lastErr
	}
	return weatherUnavailable(lastErr), lastErr
}

func weatherUnavailable(err error) models.Weather {
	weather := models.Weather{Error: "Weather is temporarily unavailable. Please try again shortly."}
	switch {
	case errors.Is(err, errWeatherLocationRequired):
		weather.NeedsLocation = true
		weather.Error = "Add your town, LGA and state to your profile to receive local weather."
	case errors.Is(err, errWeatherLocationNotFound):
		weather.NeedsLocation = true
		weather.Error = "Weather could not be found for your saved location. Check your town, LGA and state."
	}
	return weather
}

// weatherLocationCandidates returns unique, increasingly broader queries for
// a user's structured location. The source text is retained as the first
// choice, because it is the most precise when a provider recognizes it.
func weatherLocationCandidates(
	location string,
	lga string,
	state string,
	country string,
) []string {
	location = strings.TrimSpace(location)
	lga = strings.TrimSpace(lga)
	state = strings.TrimSpace(state)
	country = strings.TrimSpace(country)

	if country == "" {
		country = "Nigeria"
	}

	if location == "" && lga == "" && state == "" {
		return nil
	}
	parts := []string{location}
	if location != "" {
		parts = append(parts, joinWeatherLocation(location, lga, state, country))
	}
	if lga != "" {
		parts = append(parts, joinWeatherLocation(lga, state, country))
	}
	if state != "" {
		parts = append(parts, weatherStateQuery(state, country))
	}
	seen := make(map[string]bool)
	candidates := make([]string, 0, len(parts))
	for _, candidate := range parts {
		key := normalizeWeatherLocation(candidate)
		if key != "" && !seen[key] {
			seen[key] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func joinWeatherLocation(fields ...string) string {
	seen := make(map[string]bool)
	var parts []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		key := normalizeWeatherLocation(field)
		if key != "" && !seen[key] {
			parts = append(parts, field)
			seen[key] = true
		}
	}
	return strings.Join(parts, ", ")
}

func weatherStateQuery(state, country string) string {
	if capital := weatherStateCapitals[normalizeWeatherRegion(state)]; capital != "" {
		return joinWeatherLocation(capital, state, country)
	}
	if state != "" && !strings.HasSuffix(strings.ToLower(state), " state") {
		state += " State"
	}
	return joinWeatherLocation(state, country)
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

// fetch resolves a place and validates the provider's current observation.
func (s *WeatherService) fetch(ctx context.Context, location, lga, state, country string) (models.Weather, error) {
	place, err := s.geocode(ctx, location, lga, state, country)
	if err != nil {
		return models.Weather{}, err
	}
	return s.fetchCoordinates(ctx, place.Latitude, place.Longitude, joinWeatherLocation(place.Name, place.Admin1, "Nigeria"))
}

func (s *WeatherService) fetchCoordinates(ctx context.Context, latitude, longitude float64, location string) (models.Weather, error) {
	query := url.Values{
		"latitude":      {strconv.FormatFloat(latitude, 'f', 4, 64)},
		"longitude":     {strconv.FormatFloat(longitude, 'f', 4, 64)},
		"current":       {"temperature_2m,relative_humidity_2m,weather_code,is_day"},
		"hourly":        {"precipitation_probability"},
		"forecast_days": {"1"},
		"timezone":      {"auto"},
		"timeformat":    {"unixtime"},
	}
	var raw struct {
		UTCOffset            int    `json:"utc_offset_seconds"`
		TimezoneAbbreviation string `json:"timezone_abbreviation"`
		Current              struct {
			Time        *int64   `json:"time"`
			Temperature *float64 `json:"temperature_2m"`
			Humidity    *float64 `json:"relative_humidity_2m"`
			Code        *int     `json:"weather_code"`
			IsDay       *int     `json:"is_day"`
		} `json:"current"`
		Hourly struct {
			Time []int64    `json:"time"`
			Rain []*float64 `json:"precipitation_probability"`
		} `json:"hourly"`
	}
	if err := s.getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+query.Encode(), &raw); err != nil {
		return models.Weather{}, err
	}
	current := raw.Current
	if current.Time == nil || current.Temperature == nil || current.Humidity == nil || current.Code == nil ||
		math.IsNaN(*current.Temperature) || math.IsInf(*current.Temperature, 0) ||
		*current.Humidity < 0 || *current.Humidity > 100 {
		return models.Weather{}, errors.New("weather API returned incomplete current conditions")
	}
	observed := time.Unix(*current.Time, 0).In(time.FixedZone(raw.TimezoneAbbreviation, raw.UTCOffset))
	if age := time.Since(observed); age > weatherMaxAge || age < -15*time.Minute {
		return models.Weather{}, errors.New("weather API returned an outdated observation")
	}
	rainChance := "—"
	if rain, ok := currentPrecipitationProbability(raw.Hourly.Time, raw.Hourly.Rain, observed); ok {
		rainChance = fmt.Sprintf("%.0f%%", rain)
	}
	summary := weatherSummary(*current.Code)
	icon := weatherIcon(*current.Code)
	if current.IsDay != nil && *current.IsDay == 0 && (*current.Code == 0 || *current.Code == 1) {
		icon = "☾"
	}
	return models.Weather{
		Available:      true,
		Location:       location,
		Source:         "Open-Meteo",
		Temperature:    fmt.Sprintf("%.0f°C", *current.Temperature),
		Summary:        summary,
		Humidity:       fmt.Sprintf("%.0f%%", *current.Humidity),
		RainChance:     rainChance,
		Conditions:     summary,
		Icon:           icon,
		Recommendation: weatherRecommendation(*current.Code),
		LastUpdated:    observed,
	}, nil
}

type weatherPlace struct {
	Name        string  `json:"name"`
	CountryCode string  `json:"country_code"`
	Admin1      string  `json:"admin1"`
	Admin2      string  `json:"admin2"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func (s *WeatherService) geocode(ctx context.Context, location, lga, state, country string) (weatherPlace, error) {
	if !strings.EqualFold(country, "Nigeria") && !strings.EqualFold(country, "NG") {
		return weatherPlace{}, errWeatherLocationNotFound
	}
	query := url.Values{"name": {location}, "count": {"20"}, "language": {"en"}, "format": {"json"}, "countryCode": {"NG"}}
	var geo struct {
		Results []weatherPlace `json:"results"`
	}
	if err := s.getJSON(ctx, "https://geocoding-api.open-meteo.com/v1/search?"+query.Encode(), &geo); err != nil {
		return weatherPlace{}, err
	}
	var best *weatherPlace
	bestScore := -1
	for _, place := range geo.Results {
		if !strings.EqualFold(place.CountryCode, "NG") {
			continue
		}
		if state != "" && normalizeWeatherRegion(place.Admin1) != normalizeWeatherRegion(state) {
			continue
		}
		if lga != "" && place.Admin2 != "" && normalizeWeatherLGA(place.Admin2) != normalizeWeatherLGA(lga) {
			continue
		}
		if math.IsNaN(place.Latitude) || math.IsNaN(place.Longitude) || math.Abs(place.Latitude) > 90 || math.Abs(place.Longitude) > 180 {
			continue
		}
		score := 0
		if lga != "" && normalizeWeatherLGA(place.Admin2) == normalizeWeatherLGA(lga) {
			score++
		}
		if score > bestScore {
			candidate := place
			best, bestScore = &candidate, score
		}
	}
	if best == nil {
		return weatherPlace{}, fmt.Errorf("%w: %q", errWeatherLocationNotFound, location)
	}
	return *best, nil
}

func normalizeWeatherRegion(region string) string {
	region = strings.TrimSuffix(normalizeWeatherLocation(region), " state")
	switch region {
	case "fct", "abuja", "federal capital territory", "abuja federal capital territory":
		return "fct"
	}
	return region
}

// Unix timestamps keep the selected rain probability independent of the
// server's timezone. Missing or unmatched values must not become a false 0%.
func currentPrecipitationProbability(times []int64, probabilities []*float64, observed time.Time) (float64, bool) {
	for i, timestamp := range times {
		if i >= len(probabilities) {
			break
		}
		if observed.Unix() >= timestamp && observed.Unix() < timestamp+3600 {
			value := probabilities[i]
			if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
				return 0, false
			}
			return clampPercentage(*value), true
		}
	}
	return 0, false
}

func weatherIcon(code int) string {
	switch code {
	case 0, 1:
		return "☀"
	case 2:
		return "⛅"
	case 3:
		return "☁"
	case 45, 48:
		return "🌫"
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return "🌧"
	case 71, 73, 75, 77, 85, 86:
		return "🌨"
	case 95, 96, 99:
		return "⛈"
	default:
		return "—"
	}
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
		io.LimitReader(resp.Body, 1024*1024),
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
