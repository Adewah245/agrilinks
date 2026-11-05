package services

import (
	_ "embed"
	"strings"
)

// These are Agro-Shield's provider-specific teaching prompts, with the
// upgraded app's name. Both public chat and saved journeys use them.
//
//go:embed learning_prompts/groq.txt
var groqLearningPrompt string

//go:embed learning_prompts/openrouter.txt
var openRouterLearningPrompt string

func learningPrompt(template, farmingType string) string {
	return strings.ReplaceAll(template, "{{farming_type}}", farmingType)
}
