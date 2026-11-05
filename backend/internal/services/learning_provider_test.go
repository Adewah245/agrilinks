package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/models"
)

func TestLearningProvidersUseLegacyModelsAndFailOver(t *testing.T) {
	for _, scenario := range []struct {
		name, primaryResponse, backupResponse, want string
		primaryStatus                               int
		primaryMissing, timeout                     bool
		wantCalls                                   string
	}{
		{name: "Groq answers", primaryResponse: "primary answer", backupResponse: "backup answer", want: "primary answer", wantCalls: "groq"},
		{name: "OpenRouter handles Groq failure", primaryStatus: 503, backupResponse: "backup answer", want: "backup answer", wantCalls: "groq,openrouter"},
		{name: "OpenRouter handles empty Groq answer", backupResponse: "backup answer", want: "backup answer", wantCalls: "groq,openrouter"},
		{name: "OpenRouter handles Groq timeout", timeout: true, backupResponse: "backup answer", want: "backup answer", wantCalls: "groq,openrouter"},
		{name: "OpenRouter works alone", primaryMissing: true, backupResponse: "backup answer", want: "backup answer", wantCalls: "openrouter"},
		{name: "Both unavailable", primaryStatus: 503, wantCalls: "groq,openrouter"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var mu sync.Mutex
			var calls []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				provider := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
				mu.Lock()
				calls = append(calls, provider)
				mu.Unlock()
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-"+provider {
					t.Error("incorrect provider method or credentials")
				}
				var body struct {
					Model               string                           `json:"model"`
					Messages            []struct{ Role, Content string } `json:"messages"`
					MaxCompletionTokens int                              `json:"max_completion_tokens"`
					MaxTokens           int                              `json:"max_tokens"`
					Temperature         float64                          `json:"temperature"`
					ReasoningEffort     string                           `json:"reasoning_effort"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if body.Model != "openai/gpt-oss-20b" || body.Temperature != 0.4 || body.ReasoningEffort != "low" {
					t.Errorf("legacy model parameters were not transferred: model=%s temperature=%v reasoning=%s", body.Model, body.Temperature, body.ReasoningEffort)
				}
				if (provider == "groq" && (body.MaxCompletionTokens != 1000 || body.MaxTokens != 0)) || (provider == "openrouter" && (body.MaxTokens != 1000 || body.MaxCompletionTokens != 0)) {
					t.Error("wrong provider token limit field")
				}
				if len(body.Messages) != 4 {
					t.Errorf("conversation history lost: got %d messages", len(body.Messages))
					return
				}
				prompt := body.Messages[0].Content
				if body.Messages[0].Role != "system" || !strings.Contains(prompt, "Agrilinks Nigeria") || !strings.Contains(prompt, "Beekeeping") || !strings.Contains(prompt, "Start my first hive") || strings.Contains(prompt, "Agro-Shield") || strings.Contains(prompt, "{{farming_type}}") {
					t.Error("teaching prompt, topic, or saved context missing")
				}
				marker := "TEACHING STYLE"
				if provider == "openrouter" {
					marker = "13. CONVERSATION RULES"
				}
				if !strings.Contains(prompt, marker) {
					t.Error("provider-specific legacy prompt missing")
				}
				if !strings.Contains(prompt, "very simple English") || !strings.Contains(prompt, "explain it straight away") {
					t.Error("plain-English instructions missing from learning provider")
				}
				if body.Messages[1].Role != "user" || body.Messages[2].Role != "assistant" || body.Messages[3].Content != "What equipment do I need?" {
					t.Error("follow-up conversation changed")
				}
				answer := scenario.backupResponse
				if provider == "groq" {
					if scenario.timeout {
						time.Sleep(100 * time.Millisecond)
					}
					if scenario.primaryStatus != 0 {
						w.WriteHeader(scenario.primaryStatus)
						return
					}
					answer = scenario.primaryResponse
				}
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
			}))
			defer server.Close()
			cfg := config.Config{GroqAPIKey: "test-groq", OpenRouterAPIKey: "test-openrouter", GroqBaseURL: server.URL + "/groq", OpenRouterBaseURL: server.URL + "/openrouter", AIMaxTokens: 1000, AIRequestTimeout: time.Second}
			if scenario.primaryMissing {
				cfg.GroqAPIKey = ""
			}
			if scenario.timeout {
				cfg.AIRequestTimeout = 30 * time.Millisecond
			}
			answer, err := NewLearningProvider(cfg).Learn(context.Background(), LearningContext{FarmingType: "Beekeeping", Goal: "Start my first hive", Messages: []models.LearningMessage{
				{SenderType: "user", Content: "Can I keep bees?"},
				{SenderType: "ai", Content: "Choose a suitable place for your hive."},
				{SenderType: "user", Content: "What equipment do I need?"},
			}})
			if scenario.want == "" {
				if !errors.Is(err, ErrLearningUnavailable) {
					t.Fatalf("expected unavailable, got %v", err)
				}
			} else if err != nil || answer != scenario.want {
				t.Fatalf("answer=%q error=%v", answer, err)
			}
			mu.Lock()
			defer mu.Unlock()
			if strings.Join(calls, ",") != scenario.wantCalls {
				t.Fatalf("calls=%v, want %s", calls, scenario.wantCalls)
			}
		})
	}
}

func TestLearningWithoutConfiguredProvidersIsUnavailable(t *testing.T) {
	if provider := NewLearningProvider(config.Config{}); provider != nil {
		t.Fatal("unconfigured learning reported available")
	}
}
