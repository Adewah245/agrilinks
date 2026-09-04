package models

// AIRequest contains the data submitted to an AI provider
// for agricultural analysis.
type AIRequest struct {
	Category    string
	Description string
	Thinking    bool

	Image     []byte
	ImageType string

	Audio     []byte
	AudioType string

	Video     []byte
	VideoType string
}

// DiagnosisResult contains the result returned by an AI provider
// after analyzing an agricultural diagnosis request.
type DiagnosisResult struct {
	Result string
}
