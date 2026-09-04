package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

// AIAssistant handles HTTP requests for the Agro-Shield AI Assistant.
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
	History []models.Diagnosis
	Result  string
	Error   string
}

const (
	maxImageSize = 10 << 20
	maxAudioSize = 10 << 20
	maxVideoSize = 30 << 20
)

// validAICategories contains the agricultural categories supported
// by the Agro-Shield AI Assistant.
var validAICategories = map[string]bool{
	"Crops":                     true,
	"Livestock":                 true,
	"Poultry":                   true,
	"Goats":                     true,
	"Fish":                      true,
	"Pests/Insects":             true,
	"Plant Problems":            true,
	"General Farming Questions": true,
}

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
func (h *AIAssistant) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	switch r.Method {

	case http.MethodGet:
		log.Println("User visited AI Assistant page")

		h.render(
			w,
			user.ID,
			"",
			"",
		)

	case http.MethodPost:

		// Enforce a real request limit before multipart parsing.
		r.Body = http.MaxBytesReader(
			w,
			r.Body,
			50<<20,
		)

		if err := r.ParseMultipartForm(50 << 20); err != nil {
			h.render(
				w,
				user.ID,
				"",
				"Couldn't read the submitted information",
			)
			return
		}

		// Clean up temporary multipart files when the request finishes.
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()

		// Create the request that will be sent to the AI service.
		request := models.AIRequest{
			Category:    strings.TrimSpace(r.FormValue("category")),
			Description: strings.TrimSpace(r.FormValue("description")),
			Thinking:    r.FormValue("thinking") == "true",
		}

		var err error

		// Validate the selected agricultural category.
		if !validAICategories[request.Category] {
			h.render(
				w,
				user.ID,
				"",
				"Please select a valid farming category",
			)
			return
		}

		// Require either a description or at least one media attachment.
		hasImage := len(r.MultipartForm.File["image"]) > 0
		hasAudio := len(r.MultipartForm.File["audio"]) > 0
		hasVideo := len(r.MultipartForm.File["video"]) > 0

		if request.Description == "" &&
			!hasImage &&
			!hasAudio &&
			!hasVideo {

			h.render(
				w,
				user.ID,
				"",
				"Describe the issue or attach media before sending",
			)
			return
		}

		// --------------------------------------------------
		// OPTIONAL IMAGE
		// --------------------------------------------------

		request.Image,
			request.ImageType,
			err = readAIUpload(
			r,
			"image",
			maxImageSize,
			"image",
		)

		if err != nil {
			h.render(
				w,
				user.ID,
				"",
				err.Error(),
			)
			return
		}

		// --------------------------------------------------
		// OPTIONAL AUDIO
		// --------------------------------------------------

		request.Audio,
			request.AudioType,
			err = readAIUpload(
			r,
			"audio",
			maxAudioSize,
			"audio",
		)

		if err != nil {
			h.render(
				w,
				user.ID,
				"",
				err.Error(),
			)
			return
		}

		// --------------------------------------------------
		// OPTIONAL VIDEO
		// --------------------------------------------------

		request.Video,
			request.VideoType,
			err = readAIUpload(
			r,
			"video",
			maxVideoSize,
			"video",
		)

		if err != nil {
			h.render(
				w,
				user.ID,
				"",
				err.Error(),
			)
			return
		}

		// --------------------------------------------------
		// SEND REQUEST TO AI SERVICE
		// --------------------------------------------------

		diagnosis, err := h.ai.Diagnose(
			user.ID,
			request,
		)

		if err != nil {
			h.render(
				w,
				user.ID,
				"",
				err.Error(),
			)
			return
		}

		if diagnosis == nil {
			h.render(
				w,
				user.ID,
				"",
				"AI returned no diagnosis",
			)
			return
		}

		// --------------------------------------------------
		// BUILD GREETING
		// --------------------------------------------------

		greeting := "Good evening"

		hour := time.Now().Hour()

		if hour < 12 {
			greeting = "Good morning"
		} else if hour < 17 {
			greeting = "Good afternoon"
		}

		result :=
			greeting +
				", " +
				user.FullName +
				". I’m Agro-Shield AI, and I’m ready to help you.\n\n" +
				diagnosis.Result

		h.render(
			w,
			user.ID,
			result,
			"",
		)

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

// render loads the authenticated user's diagnosis history and renders
// the AI Assistant page.
func (h *AIAssistant) render(
	w http.ResponseWriter,
	userID int,
	result string,
	errMsg string,
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
		History: history,
		Result:  result,
		Error:   errMsg,
	}

	if err := render.RenderTemplates(
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
