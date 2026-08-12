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
		log.Println("User Visited Ai Assistant page")
		h.render(w, farmer.ID, "", "")

	case http.MethodPost:
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			h.render(w, farmer.ID, "", "Couldn't read the uploaded image")
			return
		}

		file, header, err := r.FormFile("crop_image")
		if err != nil {
			h.render(w, farmer.ID, "", "Please choose an image to analyze")
			return
		}
		defer file.Close()

		imageBytes, err := io.ReadAll(file)
		if err != nil {
			h.render(w, farmer.ID, "", "Couldn't read the uploaded image")
			return
		}

		diagnosis, err := h.ai.Diagnose(farmer.ID, header.Filename, imageBytes)
		if err != nil {
			h.render(w, farmer.ID, "", err.Error())
			return
		}

		h.render(w, farmer.ID, diagnosis.Result, "")

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AIAssistant) render(w http.ResponseWriter, farmerID int, result, errMsg string) {
	history, err := h.ai.History(farmerID)
	if err != nil {
		log.Println("failed to load diagnosis history:", err)
	}

	data := AIAssistantPageData{
		History: history,
		Result:  result,
		Error:   errMsg,
	}

	if err := render.RenderTemplates(w, "ai-assistant.html", data); err != nil {
		log.Println("render error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
