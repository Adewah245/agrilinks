package handlers

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"

	"backend/internal/models"
)

type coordinateWeatherService interface {
	CurrentAtCoordinates(context.Context, float64, float64) (models.Weather, error)
}

// CurrentWeather reads a device location for this request only. Coordinates
// stay out of URLs, logs, and the saved account profile.
func (h *Dashboard) CurrentWeather(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST to request local weather."})
		return
	}
	var input struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Accuracy  *float64 `json:"accuracy"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A valid device location is required."})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF || input.Latitude == nil || input.Longitude == nil || input.Accuracy == nil ||
		math.IsNaN(*input.Latitude) || math.IsInf(*input.Latitude, 0) || math.Abs(*input.Latitude) > 90 ||
		math.IsNaN(*input.Longitude) || math.IsInf(*input.Longitude, 0) || math.Abs(*input.Longitude) > 180 ||
		math.IsNaN(*input.Accuracy) || math.IsInf(*input.Accuracy, 0) || *input.Accuracy < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A valid device location is required."})
		return
	}
	if *input.Accuracy > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Your device location is too approximate. Enable precise location and try again."})
		return
	}
	if h.weather == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Weather is temporarily unavailable. Please try again shortly."})
		return
	}
	weather, err := h.weather.CurrentAtCoordinates(r.Context(), *input.Latitude, *input.Longitude)
	if err != nil && !weather.Available {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Weather is temporarily unavailable for your current location. Please try again shortly."})
		return
	}
	writeJSON(w, http.StatusOK, weather)
}
