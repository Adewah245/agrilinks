package models

import "time"

// FarmingType represents an agricultural activity
// supported by Agro-Shield.
//
// Examples:
//   - Crop Farming
//   - Livestock Farming
//   - Poultry Farming
//   - Fisheries
//   - Beekeeping
//   - Other Farming
type FarmingType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserFarmingType represents the relationship between
// a user and a farming type.
//
// This is NOT a user role.
//
// A user can have multiple farming types with different
// statuses.
//
// Example:
//
//	Crop Farming  -> confirmed
//	Poultry      -> learning
//	Fisheries    -> interested
type UserFarmingType struct {
	ID            int `json:"id"`
	UserID        int `json:"user_id"`
	FarmingTypeID int `json:"farming_type_id"`

	Status string `json:"status"`
	Source string `json:"source"`

	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Optional related information used when returning
	// user-facing learning data.
	FarmingType *FarmingType `json:"farming_type,omitempty"`
}

// LearningJourney represents a user's learning and
// preparation journey for one farming type.
//
// A user can have multiple journeys for different
// farming types, but only one active/paused journey
// for the same farming type at a time.
type LearningJourney struct {
	ID                int `json:"id"`
	UserID            int `json:"user_id"`
	FarmingTypeID     int `json:"farming_type_id"`
	UserFarmingTypeID int `json:"user_farming_type_id"`

	Status          string `json:"status"`
	ExperienceLevel string `json:"experience_level"`

	Goal string `json:"goal,omitempty"`

	CurrentStage string `json:"current_stage"`

	ReadinessScore int `json:"readiness_score"`

	StartedAt      time.Time  `json:"started_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Optional related farming type information.
	FarmingType *FarmingType `json:"farming_type,omitempty"`
}

// LearningMessage represents a message inside a
// user's AI learning journey.
//
// sender_type:
//
//	user
//	ai
//	system
//
// message_type:
//
//	text
//	guidance
//	recommendation
//	question
//	milestone
//	warning
type LearningMessage struct {
	ID        int `json:"id"`
	JourneyID int `json:"journey_id"`

	SenderType  string `json:"sender_type"`
	MessageType string `json:"message_type"`

	Content string `json:"content"`

	CreatedAt time.Time `json:"created_at"`
}

// LearningCheckpoint represents one preparation or
// readiness requirement inside a learning journey.
//
// The checkpoint_key is intentionally generic because
// different farming types require different preparation.
//
// Example:
//
//	Poultry:
//	  housing_ready
//	  water_available
//
//	Fisheries:
//	  pond_ready
//	  water_source_available
//
//	Crop:
//	  land_available
//	  seed_source_identified
type LearningCheckpoint struct {
	ID        int `json:"id"`
	JourneyID int `json:"journey_id"`

	CheckpointKey string `json:"checkpoint_key"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`

	Status   string `json:"status"`
	Progress int    `json:"progress"`

	Notes string `json:"notes,omitempty"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
