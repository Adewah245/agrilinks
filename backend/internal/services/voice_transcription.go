package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/config"
)

var ErrVoiceUnavailable = errors.New("we could not hear your question right now; please try again or type it")

type VoiceTranscriber interface {
	Transcribe(context.Context, []byte, string) (string, error)
}
type GroqVoiceTranscriber struct {
	endpoint, key, model string
	client               *http.Client
}

func NewVoiceTranscriber(cfg config.Config) VoiceTranscriber {
	if cfg.GroqAPIKey == "" {
		return nil
	}
	model := cfg.GroqTranscriptionModel
	if model == "" {
		model = "whisper-large-v3-turbo"
	}
	base := cfg.GroqBaseURL
	if base == "" {
		base = "https://api.groq.com/openai/v1"
	}
	timeout := cfg.AIRequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &GroqVoiceTranscriber{endpoint: strings.TrimRight(base, "/") + "/audio/transcriptions", key: cfg.GroqAPIKey, model: model, client: &http.Client{Timeout: timeout}}
}
func (p *GroqVoiceTranscriber) Transcribe(ctx context.Context, audio []byte, mime string) (string, error) {
	if len(audio) == 0 || len(audio) > 2<<20 || !strings.HasPrefix(mime, "audio/") {
		return "", ErrVoiceUnavailable
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", p.model)
	_ = writer.WriteField("language", "en")
	_ = writer.WriteField("response_format", "json")
	extension := "webm"
	if strings.Contains(mime, "mp4") {
		extension = "mp4"
	} else if strings.Contains(mime, "ogg") {
		extension = "ogg"
	} else if strings.Contains(mime, "wav") {
		extension = "wav"
	} else if strings.Contains(mime, "mpeg") {
		extension = "mp3"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="question.`+extension+`"`)
	header.Set("Content-Type", strings.Split(mime, ";")[0])
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	_, _ = part.Write(audio)
	_ = writer.Close()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+p.key)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := p.client.Do(request)
	if err != nil {
		return "", ErrVoiceUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ErrVoiceUnavailable
	}
	var result struct {
		Text string `json:"text"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil {
		return "", ErrVoiceUnavailable
	}
	text := strings.TrimSpace(result.Text)
	if text == "" || utf8.RuneCountInString(text) > 1200 {
		return "", ErrVoiceUnavailable
	}
	return text, nil
}
