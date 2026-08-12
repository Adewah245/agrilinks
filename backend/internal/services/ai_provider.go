package services

type DiagnosisResult struct {
	Result string
}

type AIProvider interface {
	AnalyzeImage(imageBytes []byte) (*DiagnosisResult, error)
}
