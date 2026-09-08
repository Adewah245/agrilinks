package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/services"
)

type LearningHandler struct {
	service *services.LearningService
}

func NewLearningHandler(service *services.LearningService) *LearningHandler {
	return &LearningHandler{
		service: service,
	}
}

// ============================================================
// LEARNING HOME
// ============================================================

// LearningHome renders the main Learning page.
//
// GET /learn
func (h *LearningHandler) LearningHome(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	farmingTypes, err := h.service.GetFarmingTypes()
	if err != nil {
		http.Error(
			w,
			"failed to load farming types",
			http.StatusInternalServerError,
		)
		return
	}

	userFarmingTypes, err := h.service.GetUserFarmingTypes(userID)
	if err != nil {
		http.Error(
			w,
			"failed to load learning interests",
			http.StatusInternalServerError,
		)
		return
	}

	activeJourney, err := h.service.GetActiveLearningJourney(userID)
	if err != nil {
		http.Error(
			w,
			"failed to load learning journey",
			http.StatusInternalServerError,
		)
		return
	}

	data := LearningHomeData{
		FarmingTypes:     farmingTypes,
		UserFarmingTypes: userFarmingTypes,
		ActiveJourney:    activeJourney,
	}

	h.renderTemplate(w, "learn.html", data)
}

// ============================================================
// START LEARNING
// ============================================================

// StartLearning creates a learning journey.
//
// POST /learn/start
func (h *LearningHandler) StartLearning(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		h.writeJSONError(
			w,
			http.StatusUnauthorized,
			"you must be logged in",
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid form data",
		)
		return
	}

	farmingTypeID, err := strconv.Atoi(
		strings.TrimSpace(r.FormValue("farming_type_id")),
	)
	if err != nil || farmingTypeID <= 0 {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid farming type",
		)
		return
	}

	experienceLevel := strings.TrimSpace(
		r.FormValue("experience_level"),
	)

	goal := strings.TrimSpace(
		r.FormValue("goal"),
	)

	journey, err := h.service.StartLearningJourney(
		userID,
		farmingTypeID,
		experienceLevel,
		goal,
	)
	if err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	// JSON response for AJAX/frontend requests.
	if wantsJSON(r) {
		h.writeJSON(
			w,
			http.StatusCreated,
			map[string]interface{}{
				"success": true,
				"message": "learning journey started",
				"journey": journey,
			},
		)
		return
	}

	http.Redirect(
		w,
		r,
		fmt.Sprintf("/learn/journey?id=%d", journey.ID),
		http.StatusSeeOther,
	)
}

// ============================================================
// LEARNING JOURNEY
// ============================================================

// LearningJourney renders a user's learning journey.
//
// GET /learn/journey?id=1
func (h *LearningHandler) LearningJourney(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	journeyID, err := strconv.Atoi(
		strings.TrimSpace(r.URL.Query().Get("id")),
	)
	if err != nil || journeyID <= 0 {
		http.Error(
			w,
			"invalid learning journey",
			http.StatusBadRequest,
		)
		return
	}

	journey, err := h.getJourneyForUser(
		userID,
		journeyID,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)
		return
	}

	checkpoints, err := h.service.GetLearningCheckpoints(
		userID,
		journeyID,
	)
	if err != nil {
		http.Error(
			w,
			"failed to load checkpoints",
			http.StatusInternalServerError,
		)
		return
	}

	messages, err := h.service.GetLearningMessages(
		userID,
		journeyID,
	)
	if err != nil {
		http.Error(
			w,
			"failed to load learning messages",
			http.StatusInternalServerError,
		)
		return
	}

	data := LearningJourneyData{
		Journey:     journey,
		Checkpoints: checkpoints,
		Messages:    messages,
	}

	h.renderTemplate(
		w,
		"learning_journey.html",
		data,
	)
}

// ============================================================
// UPDATE JOURNEY
// ============================================================

// UpdateJourneyProgress updates the learner's progress.
//
// POST /learn/journey/update
func (h *LearningHandler) UpdateJourneyProgress(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		h.writeJSONError(
			w,
			http.StatusUnauthorized,
			"you must be logged in",
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid form data",
		)
		return
	}

	journeyID, err := strconv.Atoi(
		r.FormValue("journey_id"),
	)
	if err != nil || journeyID <= 0 {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid journey",
		)
		return
	}

	stage := strings.TrimSpace(
		r.FormValue("stage"),
	)

	status := strings.TrimSpace(
		r.FormValue("status"),
	)

	readinessScore, err := strconv.Atoi(
		r.FormValue("readiness_score"),
	)
	if err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid readiness score",
		)
		return
	}

	if err := h.service.UpdateJourneyProgress(
		userID,
		journeyID,
		stage,
		readinessScore,
		status,
	); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if wantsJSON(r) {
		h.writeJSON(
			w,
			http.StatusOK,
			map[string]interface{}{
				"success": true,
				"message": "learning journey updated",
			},
		)
		return
	}

	http.Redirect(
		w,
		r,
		fmt.Sprintf("/learn/journey?id=%d", journeyID),
		http.StatusSeeOther,
	)
}

// ============================================================
// LEARNING MESSAGES
// ============================================================

// SendMessage sends a message into a learning journey.
//
// POST /learn/message
func (h *LearningHandler) SendMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		h.writeJSONError(
			w,
			http.StatusUnauthorized,
			"you must be logged in",
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid form data",
		)
		return
	}

	journeyID, err := strconv.Atoi(
		r.FormValue("journey_id"),
	)
	if err != nil || journeyID <= 0 {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid journey",
		)
		return
	}

	content := strings.TrimSpace(
		r.FormValue("content"),
	)

	message, err := h.service.SendLearningMessage(
		userID,
		journeyID,
		"user",
		"text",
		content,
	)
	if err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	h.writeJSON(
		w,
		http.StatusCreated,
		map[string]interface{}{
			"success": true,
			"message": message,
		},
	)
}

// ============================================================
// CHECKPOINT
// ============================================================

// UpdateCheckpoint updates one learning checkpoint.
//
// POST /learn/checkpoint/update
func (h *LearningHandler) UpdateCheckpoint(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getLearningUserID(r)
	if !ok {
		h.writeJSONError(
			w,
			http.StatusUnauthorized,
			"you must be logged in",
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid form data",
		)
		return
	}

	checkpointID, err := strconv.Atoi(
		r.FormValue("checkpoint_id"),
	)
	if err != nil || checkpointID <= 0 {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid checkpoint",
		)
		return
	}

	journeyID, err := strconv.Atoi(
		r.FormValue("journey_id"),
	)
	if err != nil || journeyID <= 0 {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid journey",
		)
		return
	}

	progress, err := strconv.Atoi(
		r.FormValue("progress"),
	)
	if err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid progress",
		)
		return
	}

	status := strings.TrimSpace(
		r.FormValue("status"),
	)

	notes := strings.TrimSpace(
		r.FormValue("notes"),
	)

	checkpoint := &models.LearningCheckpoint{
		ID:        checkpointID,
		JourneyID: journeyID,
		Status:    status,
		Progress:  progress,
		Notes:     notes,
	}

	if err := h.service.UpdateLearningCheckpoint(
		userID,
		checkpoint,
	); err != nil {
		h.writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	h.writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"success":    true,
			"checkpoint": checkpoint,
		},
	)
}

// ============================================================
// DATA STRUCTURES FOR TEMPLATES
// ============================================================

type LearningHomeData struct {
	FarmingTypes     []models.FarmingType
	UserFarmingTypes []models.UserFarmingType
	ActiveJourney    *models.LearningJourney
}

type LearningJourneyData struct {
	Journey     *models.LearningJourney
	Checkpoints []models.LearningCheckpoint
	Messages    []models.LearningMessage
}

// ============================================================
// HELPERS
// ============================================================

// getLearningUserID gets the logged-in user ID.
//
// IMPORTANT:
// This function assumes your authentication middleware stores
// the user ID in the request context under "user_id".
//
// If Agro-Shield currently uses a different context key,
// we will replace this helper when we connect the routes.
func getLearningUserID(r *http.Request) (int, bool) {
	value := r.Context().Value("user_id")

	switch v := value.(type) {
	case int:
		if v > 0 {
			return v, true
		}

	case int64:
		if v > 0 {
			return int(v), true
		}

	case string:
		id, err := strconv.Atoi(v)
		if err == nil && id > 0 {
			return id, true
		}
	}

	return 0, false
}

// getJourneyForUser makes sure the requested journey belongs
// to the logged-in user.
func (h *LearningHandler) getJourneyForUser(
	userID int,
	journeyID int,
) (*models.LearningJourney, error) {
	// We currently obtain the user's active journey first.
	journey, err := h.service.GetActiveLearningJourney(userID)
	if err != nil {
		return nil, err
	}

	if journey == nil {
		return nil, fmt.Errorf("learning journey not found")
	}

	if journey.ID != journeyID {
		return nil, fmt.Errorf("learning journey not found")
	}

	return journey, nil
}

// renderTemplate renders an HTML template.
func (h *LearningHandler) renderTemplate(
	w http.ResponseWriter,
	name string,
	data interface{},
) {
	tmpl, err := template.ParseFiles(
		"frontend/pages/" + name,
	)
	if err != nil {
		http.Error(
			w,
			"failed to load learning page",
			http.StatusInternalServerError,
		)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(
			w,
			"failed to render learning page",
			http.StatusInternalServerError,
		)
	}
}

// writeJSON writes a JSON response.
func (h *LearningHandler) writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return
	}
}

// writeJSONError writes a JSON error response.
func (h *LearningHandler) writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	h.writeJSON(
		w,
		status,
		map[string]interface{}{
			"success": false,
			"error":   message,
		},
	)
}

// wantsJSON determines whether the request expects JSON.
func wantsJSON(r *http.Request) bool {
	if strings.Contains(
		r.Header.Get("Accept"),
		"application/json",
	) {
		return true
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}

	return false
}
