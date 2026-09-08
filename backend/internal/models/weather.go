package models

import "time"

// Weather represents current weather information used by Agro-Shield.
//
// Weather is an external-service/application model. It is not stored
// directly in the Agro-Shield database.
//
// Available tells the application whether usable weather information
// was successfully obtained. When weather is unavailable, the dashboard
// should continue working and can display a graceful fallback instead
// of failing the entire page.
type Weather struct {
	Available bool `json:"available"`

	// Location identifies the place for which the weather was obtained.
	Location string `json:"location"`

	// Source identifies the weather provider.
	Source string `json:"source"`

	// Human-readable weather values used by the current dashboard UI.
	Temperature string `json:"temperature"`
	Summary     string `json:"summary"`
	Humidity    string `json:"humidity"`
	RainChance  string `json:"rain_chance"`
	Conditions  string `json:"conditions"`

	// Farming-oriented guidance generated from the weather conditions.
	Recommendation string `json:"recommendation"`

	// LastUpdated records when the weather information was obtained.
	//
	// A zero value means that no valid weather update has been received.
	LastUpdated time.Time `json:"last_updated"`

	// Cached indicates that the displayed weather came from a cached
	// value rather than a fresh provider request.
	Cached bool `json:"cached"`

	// Error contains a safe, user-facing description when weather
	// information could not be obtained.
	//
	// It should never contain sensitive provider credentials,
	// internal stack traces, or raw infrastructure errors.
	Error string `json:"error,omitempty"`
}

// IsUsable reports whether Agro-Shield has weather information that
// can safely be displayed to the user.
func (w *Weather) IsUsable() bool {
	return w != nil && w.Available
}

// HasError reports whether the weather request produced an error.
func (w *Weather) HasError() bool {
	return w != nil && w.Error != ""
}

// IsFresh reports whether the weather object contains a valid
// update timestamp.
//
// Freshness duration itself should be decided by the weather service,
// because different providers may have different update intervals.
func (w *Weather) IsFresh() bool {
	return w != nil && !w.LastUpdated.IsZero()
}
