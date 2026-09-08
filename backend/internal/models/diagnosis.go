package models

import "time"

// Diagnosis represents an agricultural diagnosis submitted by a user.
//
// A diagnosis may optionally be associated with a farming type.
// The actual diagnosis details are stored in Category, Symptoms,
// Result, and Confidence.
//
// Media attached to the diagnosis is represented separately by
// DiagnosisMedia.
type Diagnosis struct {
	ID int `json:"id"`

	UserID int `json:"user_id"`

	// Optional farming activity associated with this diagnosis.
	FarmingTypeID *int `json:"farming_type_id,omitempty"`

	// Diagnosis information.
	Category   *string  `json:"category,omitempty"`
	Symptoms   *string  `json:"symptoms,omitempty"`
	Result     *string  `json:"result,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Media attached to this diagnosis.
	Media []DiagnosisMedia `json:"media,omitempty"`
}

// DiagnosisMedia represents a file attached to a diagnosis.
//
// The database supports image, video, audio, and document media.
type DiagnosisMedia struct {
	ID int `json:"id"`

	DiagnosisID int `json:"diagnosis_id"`

	MediaType string `json:"media_type"`

	MediaURL string `json:"media_url"`

	// ImageName is retained because the database allows it for
	// backward compatibility with previously uploaded images.
	ImageName *string `json:"image_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}
