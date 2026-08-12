package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ClaudeProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClaudeProvider(apiKey string) *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: apiKey,
		model:  "claude-sonnet-4-20250514",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *ClaudeProvider) AnalyzeImage(
	imageBytes []byte,
) (*DiagnosisResult, error) {
	if p.apiKey == "" {
		return nil, errors.New("Anthropic API key is missing")
	}

	encodedImage := base64.StdEncoding.EncodeToString(imageBytes)

	requestBody := map[string]any{
		"model":      p.model,
		"max_tokens": 500,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/jpeg",
							"data":       encodedImage,
						},
					},
					map[string]any{
						"type": "text",
						"text": `Analyze this crop leaf image.

Give:
1. The likely disease, pest, deficiency, or healthy status.
2. The visible signs that support your answer.
3. One practical recommended action.

If the image is unclear, say that the result is uncertain.
Do not claim this is a guaranteed diagnosis.`,
					},
				},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal Claude request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create Claude request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to Claude: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Claude response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"Claude returned status %d: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode Claude response: %w", err)
	}

	for _, content := range response.Content {
		if content.Type == "text" && content.Text != "" {
			return &DiagnosisResult{
				Result: content.Text,
			}, nil
		}
	}

	return nil, errors.New("Claude returned no diagnosis text")
}