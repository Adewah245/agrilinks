package handlers

import (
	"io"
	"log"
	"net/http"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type AIAssistant struct {
	ai *services.AIService
}

func NewAIAssistantHandler(ai *services.AIService) *AIAssistant {
	return &AIAssistant{ai: ai}
}

type AIAssistantPageData struct {
	History []models.Diagnosis
	Result  string
	Error   string
}

func (h *AIAssistant) Handler(w http.ResponseWriter, r *http.Request) {
	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {

	case http.MethodGet:
		log.Println("User Visited AI Assistant page")
		h.render(w, farmer.ID, "", "")

	case http.MethodPost:
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			h.render(
				w,
				farmer.ID,
				"",
				"Couldn't read the submitted information",
			)
			return
		}

		// Create the request that will be sent to the AI service.
		request := services.AIRequest{
			Category:    r.FormValue("category"),
			Description: r.FormValue("description"),
		}

		// --------------------------------------------------
		// OPTIONAL IMAGE
		// --------------------------------------------------

		if file, header, err := r.FormFile("image"); err == nil {
			defer file.Close()

			request.Image, err = io.ReadAll(file)
			if err != nil {
				h.render(
					w,
					farmer.ID,
					"",
					"Couldn't read the image",
				)
				return
			}

			request.ImageType = header.Header.Get("Content-Type")
		}

		// --------------------------------------------------
		// OPTIONAL AUDIO
		// --------------------------------------------------

		if file, header, err := r.FormFile("audio"); err == nil {
			defer file.Close()

			request.Audio, err = io.ReadAll(file)
			if err != nil {
				h.render(
					w,
					farmer.ID,
					"",
					"Couldn't read the audio",
				)
				return
			}

			request.AudioType = header.Header.Get("Content-Type")
		}

		// --------------------------------------------------
		// OPTIONAL VIDEO
		// --------------------------------------------------

		if file, header, err := r.FormFile("video"); err == nil {
			defer file.Close()

			request.Video, err = io.ReadAll(file)
			if err != nil {
				h.render(
					w,
					farmer.ID,
					"",
					"Couldn't read the video",
				)
				return
			}

			request.VideoType = header.Header.Get("Content-Type")
		}

		// --------------------------------------------------
		// SEND REQUEST TO AI SERVICE
		// --------------------------------------------------

		diagnosis, err := h.ai.Diagnose(
			farmer.ID,
			request,
		)

		if err != nil {
			h.render(
				w,
				farmer.ID,
				"",
				err.Error(),
			)
			return
		}

		h.render(
			w,
			farmer.ID,
			diagnosis.Result,
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

func (h *AIAssistant) render(
	w http.ResponseWriter,
	farmerID int,
	result string,
	errMsg string,
) {
	history, err := h.ai.History(farmerID)
	if err != nil {
		log.Println(
			"failed to load diagnosis history:",
			err,
		)
	}

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
		log.Println("render error", err)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
	}
}
