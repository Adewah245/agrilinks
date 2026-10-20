package models

// AIRequest contains the data submitted to an AI provider
// for agricultural analysis.
//
// User identity is included as request context so the AI layer
// can personalize its response without duplicating user data
// inside the diagnosis database model.
type AIRequest struct {
	// User context.
	UserID   int
	UserName string

	// Agricultural context.
	Category    string
	Description string
	Thinking    bool

	// Image input.
	Image     []byte
	ImageType string

	// Audio input.
	Audio     []byte
	AudioType string

	// Video input.
	Video     []byte
	VideoType string
}

// DiagnosisResult contains the result returned by an AI provider
// after analyzing an agricultural diagnosis request.
type DiagnosisResult struct {
	Result string
}
