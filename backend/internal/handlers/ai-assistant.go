package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

// AIAssistant handles HTTP requests for the Agrilinks Nigeria AI Assistant.
type AIAssistant struct {
	ai *services.AIService
}

// NewAIAssistantHandler creates a new AI Assistant handler.
func NewAIAssistantHandler(ai *services.AIService) *AIAssistant {
	return &AIAssistant{
		ai: ai,
	}
}

// AIAssistantPageData contains the data rendered on the AI Assistant page.
type AIAssistantPageData struct {
	History           []models.Diagnosis
	Result            string
	Error             string
	Available         bool
	ExamplesAvailable bool
	ResultID          int
}

const (
	maxImageSize = 10 << 20
	maxAudioSize = 10 << 20
	maxVideoSize = 30 << 20
)

// validAICategories contains the agricultural categories supported
// by the Agrilinks Nigeria AI Assistant.
var validAICategories = func() map[string]bool {
	values := make(map[string]bool)
	for _, category := range services.DiagnosisCategories() {
		values[category] = true
	}
	return values
}()

// readAIUpload reads and validates an uploaded AI media file.
//
// The file is limited to the supplied maximum size and its detected
// content type is checked before the data is returned.
func readAIUpload(
	r *http.Request,
	field string,
	maxSize int64,
	allowedPrefix string,
) ([]byte, string, error) {
	file, header, err := r.FormFile(field)

	if errors.Is(err, http.ErrMissingFile) {
		return nil, "", nil
	}

	if err != nil {
		return nil, "", err
	}

	defer file.Close()

	data, err := io.ReadAll(
		io.LimitReader(file, maxSize+1),
	)
	if err != nil {
		return nil, "", err
	}

	if int64(len(data)) > maxSize {
		return nil, "", errors.New(field + " is too large")
	}

	detectedMIME := http.DetectContentType(data)

	declaredMIME := strings.ToLower(
		strings.TrimSpace(
			header.Header.Get("Content-Type"),
		),
	)

	if separator := strings.IndexByte(declaredMIME, ';'); separator >= 0 {
		declaredMIME = strings.TrimSpace(
			declaredMIME[:separator],
		)
	}

	mime := detectedMIME

	valid := strings.HasPrefix(
		detectedMIME,
		allowedPrefix+"/",
	)

	// Browser-recorded audio often uses WebM or MP4 containers.
	// Go may detect those containers as video or generic binary even
	// when they contain only an audio track, so the browser's declared
	// audio MIME type is retained after validating the container.
	if field == "audio" &&
		strings.HasPrefix(declaredMIME, "audio/") {

		validContainer :=
			strings.HasPrefix(detectedMIME, "audio/") ||
				detectedMIME == "video/webm" ||
				detectedMIME == "video/mp4" ||
				detectedMIME == "application/octet-stream"

		if validContainer {
			valid = true
			mime = declaredMIME
		}
	}

	if !valid {
		return nil, "", errors.New(
			"unsupported " + field + " format",
		)
	}

	return data, mime, nil
}

// Handler processes GET and POST requests for the AI Assistant.
//
// GET displays the AI Assistant page and the user's diagnosis history.
// POST validates the submitted agricultural information and optional
// media, sends the request to the AI service, and displays the result.
func (h *AIAssistant) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		h.render(w, user.ID, "", "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	fail := func(message string, status int) {
		if wantsJSON(r) {
			writeError(w, status, message)
		} else {
			h.render(w, user.ID, "", message)
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		fail("We could not read that information. Please try again.", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()
	request := models.AIRequest{UserID: user.ID, UserName: user.FullName, Category: strings.TrimSpace(r.FormValue("category")), Description: strings.TrimSpace(r.FormValue("description")), Thinking: r.FormValue("thinking") == "true"}
	if !validAICategories[request.Category] {
		fail("Please choose a farming topic.", http.StatusBadRequest)
		return
	}
	var err error
	request.Image, request.ImageType, err = readAIUpload(r, "image", maxImageSize, "image")
	if err != nil {
		fail(err.Error(), http.StatusBadRequest)
		return
	}
	request.Audio, request.AudioType, err = readAIUpload(r, "audio", maxAudioSize, "audio")
	if err != nil {
		fail(err.Error(), http.StatusBadRequest)
		return
	}
	request.Video, request.VideoType, err = readAIUpload(r, "video", maxVideoSize, "video")
	if err != nil {
		fail(err.Error(), http.StatusBadRequest)
		return
	}
	if request.Description == "" && len(request.Image)+len(request.Audio)+len(request.Video) == 0 {
		fail("Describe what happened or attach a photo or recording.", http.StatusBadRequest)
		return
	}
	diagnosis, err := h.ai.Diagnose(r.Context(), user.ID, request)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			fail("The answer took too long. Please try again.", http.StatusServiceUnavailable)
			return
		}
		fail(err.Error(), http.StatusServiceUnavailable)
		return
	}
	if diagnosis == nil {
		fail("The farming assistant returned no answer. Please try again.", http.StatusServiceUnavailable)
		return
	}
	result := diagnosisText(diagnosis.Result)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"result": result, "diagnosis_id": diagnosis.ID, "examples_available": h.ai.ExamplesAvailable()})
		return
	}
	h.render(w, user.ID, result, "", diagnosis.ID)
}

func (h *AIAssistant) Example(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var input struct {
		DiagnosisID int `json:"diagnosis_id"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.DiagnosisID <= 0 {
		writeError(w, http.StatusBadRequest, "Choose a saved answer first.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Choose a saved answer first.")
		return
	}
	result, err := h.ai.Example(r.Context(), user.ID, input.DiagnosisID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "That answer was not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, services.ErrExampleUnavailable.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func diagnosisText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// render loads the authenticated user's diagnosis history and renders
// the AI Assistant page.
func (h *AIAssistant) render(
	w http.ResponseWriter,
	userID int,
	result string,
	errMsg string,
	diagnosisID ...int,
) {
	historyStart := time.Now()

	log.Println("⏱️ Loading diagnosis history")

	history, err := h.ai.History(userID)

	if err != nil {
		log.Println(
			"failed to load diagnosis history:",
			err,
		)
	}

	log.Printf(
		"⏱️ History loading took: %v",
		time.Since(historyStart),
	)

	data := AIAssistantPageData{
		History:           history,
		Result:            result,
		Error:             errMsg,
		Available:         h.ai.Available(),
		ExamplesAvailable: h.ai.ExamplesAvailable(),
	}

	if len(diagnosisID) > 0 {
		data.ResultID = diagnosisID[0]
	}
	if err := render.RenderPage(
		w,
		"ai-assistant.html",
		data,
	); err != nil {

		if clientDisconnected(err) {
			log.Printf(
				"AI assistant client disconnected while rendering: %v",
				err,
			)
			return
		}

		log.Println(
			"render error",
			err,
		)

		// A template can fail after writing headers/body.
		// Avoid attempting a second response, which would produce
		// a misleading superfluous response header warning.
	}
}
