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
	h.render(w, "learning.html", struct {
		FarmingTypes []string
		AIAvailable  bool
	}{services.LearningCategories(), h.service.AIAvailable()})
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
