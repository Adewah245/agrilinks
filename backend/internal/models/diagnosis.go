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

// CategoryText returns the category in a form templates can compare and
// display without dereferencing the nullable database field.
func (d Diagnosis) CategoryText() string { return diagnosisText(d.Category) }

// SymptomsText returns the diagnosis symptoms for display.
func (d Diagnosis) SymptomsText() string { return diagnosisText(d.Symptoms) }

// ResultText returns the diagnosis result for display.
func (d Diagnosis) ResultText() string { return diagnosisText(d.Result) }

func diagnosisText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
