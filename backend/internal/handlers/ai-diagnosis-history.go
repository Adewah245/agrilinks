package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

// AIDiagnosisHistory handles requests for a user's AI diagnosis history.
type AIDiagnosisHistory struct {
	ai *services.AIService
}

// NewAIDiagnosisHistoryHandler creates a new AI diagnosis history handler.
func NewAIDiagnosisHistoryHandler(
	ai *services.AIService,
) *AIDiagnosisHistory {
	return &AIDiagnosisHistory{
		ai: ai,
	}
}

// AIDiagnosisHistoryPageData contains the data rendered on the
// AI diagnosis history page.
type AIDiagnosisHistoryPageData struct {
	History []models.Diagnosis
}

// Handler displays the authenticated user's diagnosis history and
// handles requests to delete individual diagnosis records.
func (h *AIDiagnosisHistory) Handler(
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

	// --------------------------------------------------
	// DELETE DIAGNOSIS
	// --------------------------------------------------

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Invalid request",
				http.StatusBadRequest,
			)
			return
		}

		id, err := strconv.Atoi(
			r.FormValue("id"),
		)

		if err != nil || id <= 0 {
			http.Error(
				w,
				"Invalid diagnosis",
				http.StatusBadRequest,
			)
			return
		}

		if err := h.ai.DeleteDiagnosis(
			user.ID,
			id,
		); err != nil {

			if errors.Is(err, sql.ErrNoRows) {
				http.Error(
					w,
					"Diagnosis not found",
					http.StatusNotFound,
				)
				return
			}

			log.Println(
				"failed to delete diagnosis:",
				err,
			)

			http.Error(
				w,
				"Unable to delete diagnosis",
				http.StatusInternalServerError,
			)
			return
		}

		// Allow the AI Assistant page to send the user back to
		// itself after deleting a diagnosis from its history.
		redirectTo := "/ai-diagnosis-history"

		if r.FormValue("return_to") == "/ai-assistant" {
			redirectTo = "/ai-assistant"
		}

		http.Redirect(
			w,
			r,
			redirectTo,
			http.StatusSeeOther,
		)

		return
	}

	// --------------------------------------------------
	// ONLY GET IS ALLOWED FOR HISTORY DISPLAY
	// --------------------------------------------------

	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	log.Println(
		"User visited AI diagnosis history",
	)

	// --------------------------------------------------
	// LOAD USER HISTORY
	// --------------------------------------------------

	history, err := h.ai.History(
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to load diagnosis history:",
			err,
		)

		http.Error(
			w,
			"Unable to load diagnosis history",
			http.StatusInternalServerError,
		)

		return
	}

	// --------------------------------------------------
	// RENDER PAGE
	// --------------------------------------------------

	data := AIDiagnosisHistoryPageData{
		History: history,
	}

	if err := render.RenderTemplates(
		w,
		"ai-diagnosis-history.html",
		data,
	); err != nil {

		log.Println(
			"render error:",
			err,
		)

		// Do not call http.Error here because the template renderer
		// may already have written the response headers/body.
		return
	}
}
