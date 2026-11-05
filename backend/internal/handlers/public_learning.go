package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"backend/internal/services"
)

func (h *LearningHandler) PublicPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	h.render(w, "learning.html", PublicLearningData{FarmingTypes: services.LearningCategories(), AIAvailable: h.service.AIAvailable(), ExamplesAvailable: h.service.ExamplesAvailable(), VoiceUploadAvailable: h.service.VoiceUploadAvailable()})
}

type PublicLearningData struct {
	FarmingTypes         []string
	AIAvailable          bool
	ExamplesAvailable    bool
	VoiceUploadAvailable bool
}

func (h *LearningHandler) PublicExample(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 96<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input services.ExampleImageRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, services.ErrInvalidExample.Error())
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, services.ErrInvalidExample.Error())
		return
	}
	result, err := h.service.Example(r.Context(), input)
	if errors.Is(err, services.ErrInvalidExample) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, services.ErrExampleUnavailable.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *LearningHandler) PublicTranscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Please record a shorter voice question.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	audio, mime, err := readAIUpload(r, "audio", 2<<20, "audio")
	if err != nil || len(audio) == 0 {
		writeError(w, http.StatusBadRequest, "Please record your voice question again.")
		return
	}
	text, err := h.service.Transcribe(r.Context(), audio, mime)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, services.ErrVoiceUnavailable.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}

func (h *LearningHandler) PublicChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request services.LearningChatRequest
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "Your question is too large.")
		} else {
			writeError(w, http.StatusBadRequest, "Invalid learning request.")
		}
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Invalid learning request.")
		return
	}
	answer, err := h.service.PublicChat(r.Context(), request)
	if err != nil {
		if errors.Is(err, services.ErrInvalidLearningChat) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusServiceUnavailable, services.ErrLearningUnavailable.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"answer": answer})
}
