package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"backend/internal/app"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testui"
)

type publicLearningProvider struct{ input services.LearningContext }

type examplePictureProvider struct {
	calls int
	input services.ExampleImageRequest
}

func (p *examplePictureProvider) Generate(_ context.Context, input services.ExampleImageRequest) (*services.ExampleImageResult, error) {
	p.calls++
	p.input = input
	return &services.ExampleImageResult{DataURL: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jV8sAAAAASUVORK5CYII=", Alt: "Example farming picture", Caption: "Example drawing. It cannot confirm what is wrong on your farm."}, nil
}

type questionTranscriber struct{ calls int }

func (p *questionTranscriber) Transcribe(_ context.Context, audio []byte, mime string) (string, error) {
	p.calls++
	return "How do I plant maize?", nil
}

func (p *publicLearningProvider) Learn(_ context.Context, input services.LearningContext) (string, error) {
	p.input = input
	return "Choose a dry, well-ventilated shelter.", nil
}

func TestPublicLearningRoutesNeedNoAccountOrDatabase(t *testing.T) {
	service := services.NewLearningService(nil)
	provider := &publicLearningProvider{}
	service.SetProvider(provider)
	service.SetImageProvider(&examplePictureProvider{})
	service.SetTranscriber(&questionTranscriber{})
	handler := routes.NewHandler(&app.Container{Config: testConfig(t), Learning: service})
	home := request(handler, http.MethodGet, "/", nil, nil)
	requireStatus(t, home, http.StatusOK)
	if !strings.Contains(testui.HTML(t, home.Body.String()), `href="/learning"`) {
		t.Fatal("public learning is not reachable from the landing page")
	}
	page := request(handler, http.MethodGet, "/learning", nil, nil)
	requireStatus(t, page, http.StatusOK)
	html := testui.HTML(t, page.Body.String())
	if !strings.Contains(html, "No account is needed") || !strings.Contains(html, "Mixed farming") || !strings.Contains(html, "Explore farming topics") {
		t.Fatal("public legacy learning page incomplete")
	}
	if len(page.Result().Cookies()) != 0 {
		t.Fatal("learning unexpectedly created an account session")
	}
	post := httptest.NewRequest(http.MethodPost, "/learning/chat", strings.NewReader(`{"farmingType":"Goat farming","messages":[{"role":"user","content":"What shelter do goats need?"}]}`))
	post.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, post)
	requireStatus(t, response, http.StatusOK)
	var result map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["answer"] == "" || provider.input.FarmingType != "Goat farming" {
		t.Fatal("public question did not reach learning provider")
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("chat unexpectedly required an account session")
	}
	service.SetProvider(nil)
	response = httptest.NewRecorder()
	post = httptest.NewRequest(http.MethodPost, "/learning/chat", strings.NewReader(`{"farmingType":"Goat farming","messages":[{"role":"user","content":"What shelter do goats need?"}]}`))
	handler.ServeHTTP(response, post)
	requireStatus(t, response, http.StatusServiceUnavailable)
}

func TestPublicVoiceAndPicturesNeedNoAccountOrDatabase(t *testing.T) {
	service := services.NewLearningService(nil)
	images := &examplePictureProvider{}
	voice := &questionTranscriber{}
	service.SetImageProvider(images)
	service.SetTranscriber(voice)
	handler := routes.NewHandler(&app.Container{Config: testConfig(t), Learning: service})
	post := httptest.NewRequest("POST", "/learning/example", strings.NewReader(`{"topic":"Crops","question":"How do I plant maize?","answer":"Give each plant enough space."}`))
	post.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, post)
	requireStatus(t, response, http.StatusOK)
	if images.calls != 1 || images.input.Topic != "Crops" || images.input.Diagnosis || len(response.Result().Cookies()) != 0 {
		t.Fatal("public pictures required an account or lost context")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="audio"; filename="question.webm"`)
	header.Set("Content-Type", "audio/webm")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte{0x1a, 0x45, 0xdf, 0xa3, 0x00, 0x00})
	_ = writer.Close()
	audioBody := append([]byte(nil), body.Bytes()...)
	post = httptest.NewRequest("POST", "/learning/transcribe", bytes.NewReader(audioBody))
	post.Header.Set("Content-Type", writer.FormDataContentType())
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, post)
	requireStatus(t, response, http.StatusOK)
	if voice.calls != 1 || !strings.Contains(response.Body.String(), "How do I plant maize?") || len(response.Result().Cookies()) != 0 {
		t.Fatal("public voice question did not work")
	}
	service.SetImageProvider(nil)
	service.SetTranscriber(nil)
	for _, path := range []string{"/learning/example", "/learning/transcribe"} {
		var r *http.Request
		if path == "/learning/example" {
			r = httptest.NewRequest("POST", path, strings.NewReader(`{"topic":"Crops","answer":"Give each plant enough space."}`))
		} else {
			r = httptest.NewRequest("POST", path, bytes.NewReader(audioBody))
			r.Header.Set("Content-Type", writer.FormDataContentType())
		}
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, r)
		requireStatus(t, response, http.StatusServiceUnavailable)
	}
}
