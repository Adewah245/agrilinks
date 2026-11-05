package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/models"
)

func examplePNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestGeminiDiagnosisRequestAndCompleteReply(t *testing.T) {
	photo := examplePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1beta/models/gemini-3.6-flash:generateContent" || r.Header.Get("x-goog-api-key") != "test-key" {
			t.Errorf("unexpected Gemini request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			SystemInstruction struct{ Parts []struct{ Text string } }
			Contents          []struct {
				Role  string
				Parts []struct {
					Text       string
					InlineData *struct {
						Data     string
						MIMEType string
					}
				}
			}
			GenerationConfig struct {
				MaxOutputTokens int
				ThinkingConfig  struct{ ThinkingLevel string }
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if len(body.SystemInstruction.Parts) != 1 || !strings.Contains(body.SystemInstruction.Parts[0].Text, "very simple English") || !strings.Contains(body.SystemInstruction.Parts[0].Text, "For other farming questions, answer directly") {
			t.Error("missing plain-English and general-question instructions")
		}
		if len(body.Contents) != 1 || body.Contents[0].Role != "user" || len(body.Contents[0].Parts) != 4 {
			t.Error("missing farming question or media")
			return
		}
		parts := body.Contents[0].Parts
		if !strings.Contains(parts[0].Text, "yellow leaves") {
			t.Error("description lost")
		}
		for i, want := range []string{"image/jpeg", "audio/webm", "video/mp4"} {
			if parts[i+1].InlineData == nil || parts[i+1].InlineData.MIMEType != want || parts[i+1].InlineData.Data == "" {
				t.Errorf("incorrect media %d", i)
			}
		}
		if body.GenerationConfig.MaxOutputTokens < 1600 || body.GenerationConfig.ThinkingConfig.ThinkingLevel != "MEDIUM" {
			t.Error("incorrect output or thinking settings")
		}
		_, _ = io.WriteString(w, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"hidden reasoning","thought":true},{"text":"Yellow leaves can have more than one cause."},{"text":"Check the soil before adding water."}]}}]}`)
	}))
	defer server.Close()
	provider, err := NewConfiguredGeminiProvider(config.Config{GeminiAPIKey: "test-key", GeminiModel: "models/gemini-3.6-flash", GeminiBaseURL: server.URL, AIRequestTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	answer, err := provider.Analyze(context.Background(), models.AIRequest{Category: "Crops", Description: "yellow leaves", Image: photo, Audio: []byte("audio"), AudioType: "audio/webm;codecs=opus", Video: []byte("video"), VideoType: "video/mp4", Thinking: true})
	if err != nil || answer == nil || answer.Result != "Yellow leaves can have more than one cause.\n\nCheck the soil before adding water." {
		t.Fatalf("answer=%v error=%v", answer, err)
	}
}

func TestGeminiDiagnosisRejectsIncompleteOrEmptyReplies(t *testing.T) {
	for _, body := range []string{
		`{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[{"text":"Give the animal"}]}}]}`,
		`{"candidates":[{"finishReason":"SAFETY","content":{"parts":[]}}]}`,
		`{"candidates":[{"content":{"parts":[{"thought":true,"text":"thinking only"}]}}]}`,
	} {
		t.Run(body[:30], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
			defer server.Close()
			provider, err := NewConfiguredGeminiProvider(config.Config{GeminiAPIKey: "test-key", GeminiBaseURL: server.URL, AIRequestTimeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			if result, err := provider.Analyze(context.Background(), models.AIRequest{Description: "How should I store grain?"}); err == nil || result != nil {
				t.Fatal("incomplete advice was displayed")
			}
		})
	}
}

func TestGeminiDiagnosisHonoursCallerCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer func() { close(release); server.Close() }()
	provider, err := NewConfiguredGeminiProvider(config.Config{GeminiAPIKey: "test-key", GeminiBaseURL: server.URL, AIRequestTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := provider.Analyze(ctx, models.AIRequest{Description: "How do I plant maize?"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("caller cancellation ignored")
	}
}

func TestGeminiThinkingCompatibility(t *testing.T) {
	if cfg := diagnosisGenerationConfig("gemini-2.5-flash", false); cfg.ThinkingConfig.ThinkingBudget == nil || *cfg.ThinkingConfig.ThinkingBudget != 0 {
		t.Fatal("2.5 non-thinking budget wrong")
	}
	if cfg := diagnosisGenerationConfig("gemini-2.5-flash", true); *cfg.ThinkingConfig.ThinkingBudget != 1024 || cfg.MaxOutputTokens < 2600 {
		t.Fatal("2.5 thinking budget wrong")
	}
	if cfg := diagnosisGenerationConfig("gemini-3.6-flash", false); cfg.ThinkingConfig.ThinkingLevel != "LOW" || cfg.ThinkingConfig.ThinkingBudget != nil {
		t.Fatal("3.x uses unsupported budget")
	}
}

func TestGeminiExamplePicturesContract(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(examplePNG(t))
	for _, test := range []struct {
		name, body string
		status     int
		valid      bool
	}{
		{"output image", `{"output_image":{"data":"` + encoded + `","mime_type":"image/png"}}`, 200, true},
		{"image output blocks", `{"outputs":[{"type":"text","data":"ignored"},{"type":"image","data":"` + encoded + `","mime_type":"image/jpeg"}]}`, 200, true},
		{"missing", `{"outputs":[]}`, 200, false},
		{"invalid encoding", `{"output_image":{"data":"not base64"}}`, 200, false},
		{"not an image", `{"output_image":{"data":"` + base64.StdEncoding.EncodeToString([]byte("<script>bad</script>")) + `","mime_type":"image/png"}}`, 200, false},
		{"broken json", "{", 200, false},
		{"upstream unavailable", `{"error":{"message":"private provider details"}}`, 403, false},
		{"oversized response", strings.Repeat(" ", (16<<20)+1), 200, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1beta/interactions" || r.Header.Get("Api-Revision") != "2026-05-20" || r.Header.Get("x-goog-api-key") != "test-key" {
					t.Error("incorrect image endpoint/headers")
				}
				var body struct {
					Model, Input   string
					Store          bool
					ResponseFormat map[string]string `json:"response_format"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Model != config.DefaultGeminiImageModel || body.Store || body.ResponseFormat["type"] != "image" || body.ResponseFormat["aspect_ratio"] != "4:3" || body.ResponseFormat["image_size"] != "1K" {
					t.Error("incorrect image payload")
				}
				if !strings.Contains(body.Input, "never diagnostic evidence") || !strings.Contains(body.Input, "everyday English") || !strings.Contains(body.Input, "yellow leaves") {
					t.Error("missing example context/instructions")
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			provider := NewExampleImageProvider(config.Config{GeminiAPIKey: "test-key", GeminiBaseURL: server.URL, AIRequestTimeout: time.Second})
			result, err := provider.Generate(context.Background(), ExampleImageRequest{Topic: "Crops", Question: "yellow leaves", Answer: "Check the soil.", Diagnosis: true})
			if !test.valid {
				if !errors.Is(err, ErrExampleUnavailable) || result != nil {
					t.Fatalf("invalid upstream accepted: %v", err)
				}
				return
			}
			if err != nil || result == nil || result.DataURL != "data:image/png;base64,"+encoded || !strings.Contains(result.Caption, "cannot confirm") {
				t.Fatalf("result=%v error=%v", result, err)
			}
		})
	}
	provider := NewExampleImageProvider(config.Config{GeminiAPIKey: "test-key"})
	if _, err := provider.Generate(context.Background(), ExampleImageRequest{Topic: "Crops"}); !errors.Is(err, ErrInvalidExample) {
		t.Fatal("empty answer allowed")
	}
	if NewExampleImageProvider(config.Config{}) != nil {
		t.Fatal("unconfigured images reported available")
	}
}

func TestGroqVoiceTranscriptionContract(t *testing.T) {
	for _, test := range []struct {
		name, mime, extension, reply string
		status                       int
		valid                        bool
	}{
		{"webm", "audio/webm;codecs=opus", "webm", `{"text":" How do I plant maize? "}`, 200, true},
		{"mp4", "audio/mp4", "mp4", `{"text":"How do I plant maize?"}`, 200, true},
		{"ogg", "audio/ogg", "ogg", `{"text":"How do I plant maize?"}`, 200, true},
		{"empty", "audio/webm", "webm", `{"text":" "}`, 200, false},
		{"broken json", "audio/webm", "webm", "{", 200, false},
		{"too long", "audio/webm", "webm", `{"text":"` + strings.Repeat("a", 1201) + `"}`, 200, false},
		{"upstream failure", "audio/webm", "webm", `{"error":"private details"}`, 401, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/audio/transcriptions" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("wrong transcription endpoint/credentials")
				}
				if err := r.ParseMultipartForm(3 << 20); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				if r.FormValue("model") != "whisper-large-v3-turbo" || r.FormValue("language") != "en" || r.FormValue("response_format") != "json" {
					t.Error("wrong transcription fields")
				}
				f, h, err := r.FormFile("file")
				if err != nil {
					t.Error(err)
					return
				}
				defer f.Close()
				contents, _ := io.ReadAll(f)
				if h.Filename != "question."+test.extension || h.Header.Get("Content-Type") != strings.Split(test.mime, ";")[0] || string(contents) != "recorded question" {
					t.Error("recorded audio lost")
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.reply)
			}))
			defer server.Close()
			provider := NewVoiceTranscriber(config.Config{GroqAPIKey: "test-key", GroqBaseURL: server.URL, AIRequestTimeout: time.Second})
			result, err := provider.Transcribe(context.Background(), []byte("recorded question"), test.mime)
			if test.valid {
				if err != nil || result != "How do I plant maize?" {
					t.Fatalf("%q %v", result, err)
				}
			} else if !errors.Is(err, ErrVoiceUnavailable) || result != "" {
				t.Fatal("invalid transcript accepted")
			}
		})
	}
	provider := NewVoiceTranscriber(config.Config{GroqAPIKey: "test-key"})
	for _, audio := range [][]byte{nil, make([]byte, (2<<20)+1)} {
		if _, err := provider.Transcribe(context.Background(), audio, "audio/webm"); !errors.Is(err, ErrVoiceUnavailable) {
			t.Fatal("invalid audio accepted")
		}
	}
	if NewVoiceTranscriber(config.Config{}) != nil {
		t.Fatal("unconfigured transcription reported available")
	}
}
