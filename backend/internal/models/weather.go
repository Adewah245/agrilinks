package models

import "time"

// Weather represents current weather information used by Agro-Shield.
//
// The structure contains both the raw weather information needed by
// the application and a farming-oriented recommendation derived from
// the current weather conditions.
type Weather struct {
	Available      bool      `json:"available"`
	Location       string    `json:"location"`
	Source         string    `json:"source"`
	Temperature    string    `json:"temperature"`
	Summary        string    `json:"summary"`
	Humidity       string    `json:"humidity"`
	RainChance     string    `json:"rain_chance"`
	Conditions     string    `json:"conditions"`
	Recommendation string    `json:"recommendation"`
	LastUpdated    time.Time `json:"last_updated"`
	Cached         bool      `json:"cached"`
}
