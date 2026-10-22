package handlers

import (
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type LearningHandler struct{ service *services.LearningService }

func NewLearningHandler(service *services.LearningService) *LearningHandler {
	return &LearningHandler{service: service}
}

type LearningHomeData struct {
	FarmingTypes     []models.FarmingType
	UserFarmingTypes []models.UserFarmingType
	Journeys         []models.LearningJourney
	ActiveJourney    *models.LearningJourney
	AIAvailable      bool
}
type LearningJourneyData struct {
	Journey     *models.LearningJourney
	Checkpoints []models.LearningCheckpoint
	Messages    []models.LearningMessage
	AIAvailable bool
}

func (h *LearningHandler) LearningHome(w http.ResponseWriter, r *http.Request) {
	id, ok := getLearningUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	types, err := h.service.GetFarmingTypes()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	journeys, err := h.service.ListJourneys(id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, "learn.html", LearningHomeData{FarmingTypes: types, Journeys: journeys, AIAvailable: h.service.AIAvailable()})
}

func (h *LearningHandler) StartLearning(w http.ResponseWriter, r *http.Request) {
	id, ok := learningPost(w, r)
	if !ok {
		return
	}
	farmingID, err := strconv.Atoi(r.FormValue("farming_type_id"))
	if err != nil {
		h.fail(w, r, errors.New("choose a farming activity"))
		return
	}
	journey, err := h.service.StartLearningJourney(id, farmingID, r.FormValue("experience_level"), r.FormValue("goal"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.saved(w, r, journey.ID, journey)
}

func (h *LearningHandler) LearningJourney(w http.ResponseWriter, r *http.Request) {
	id, ok := getLearningUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	journeyID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	journey, err := h.service.GetJourneyForUser(id, journeyID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	checkpoints, err := h.service.GetLearningCheckpoints(id, journeyID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	messages, err := h.service.GetLearningMessages(id, journeyID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, "learning_journey.html", LearningJourneyData{Journey: journey, Checkpoints: checkpoints, Messages: messages, AIAvailable: h.service.AIAvailable()})
}

func (h *LearningHandler) UpdateJourneyProgress(w http.ResponseWriter, r *http.Request) {
	id, ok := learningPost(w, r)
	if !ok {
		return
	}
	journeyID, err := strconv.Atoi(r.FormValue("journey_id"))
	if err != nil {
		h.fail(w, r, repository.ErrLearningNotFound)
		return
	}
	// Percentages come from the saved checklist, never a submitted score.
	if r.FormValue("readiness_score") != "" || r.FormValue("stage") != "" {
		h.fail(w, r, errors.New("progress is calculated from your checklist"))
		return
	}
	if err := h.service.ChangeJourneyStatus(id, journeyID, r.FormValue("status")); err != nil {
		h.fail(w, r, err)
		return
	}
	h.saved(w, r, journeyID, nil)
}

func (h *LearningHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := learningPost(w, r)
	if !ok {
		return
	}
	journeyID, err := strconv.Atoi(r.FormValue("journey_id"))
	if err != nil {
		h.fail(w, r, repository.ErrLearningNotFound)
		return
	}
	messages, err := h.service.Ask(r.Context(), id, journeyID, r.FormValue("content"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.saved(w, r, journeyID, messages)
}

func (h *LearningHandler) CreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	id, ok := learningPost(w, r)
	if !ok {
		return
	}
	journeyID, err := strconv.Atoi(r.FormValue("journey_id"))
	if err != nil {
		h.fail(w, r, repository.ErrLearningNotFound)
		return
	}
	checkpoint, err := h.service.CreateLearningCheckpoint(id, journeyID, "", r.FormValue("title"), r.FormValue("description"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.saved(w, r, journeyID, checkpoint)
}

func (h *LearningHandler) UpdateCheckpoint(w http.ResponseWriter, r *http.Request) {
	id, ok := learningPost(w, r)
	if !ok {
		return
	}
	checkpointID, err := strconv.Atoi(r.FormValue("checkpoint_id"))
	if err != nil {
		h.fail(w, r, repository.ErrLearningNotFound)
		return
	}
	journeyID, err := strconv.Atoi(r.FormValue("journey_id"))
	if err != nil {
		h.fail(w, r, repository.ErrLearningNotFound)
		return
	}
	progress, err := strconv.Atoi(r.FormValue("progress"))
	if err != nil {
		h.fail(w, r, errors.New("enter progress from 0 to 100"))
		return
	}
	checkpoint := &models.LearningCheckpoint{ID: checkpointID, JourneyID: journeyID, Progress: progress, Status: r.FormValue("status"), Notes: r.FormValue("notes")}
	if err := h.service.UpdateLearningCheckpoint(id, checkpoint); err != nil {
		h.fail(w, r, err)
		return
	}
	h.saved(w, r, journeyID, checkpoint)
}

func getLearningUserID(r *http.Request) (int, bool) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		return 0, false
	}
	return user.ID, true
}
func learningPost(w http.ResponseWriter, r *http.Request) (int, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return 0, false
	}
	id, ok := getLearningUserID(r)
	if !ok {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return 0, false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "We could not read that form", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}
func (h *LearningHandler) saved(w http.ResponseWriter, r *http.Request, journeyID int, data any) {
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/learn/journey?id=%d", journeyID), http.StatusSeeOther)
}
func (h *LearningHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("learning request: %v", err)
	status, message := http.StatusBadRequest, "We could not save that change. Check your details and try again."
	switch {
	case errors.Is(err, repository.ErrLearningNotFound):
		status, message = http.StatusNotFound, "That learning journey was not found."
	case errors.Is(err, repository.ErrLearningConflict):
		status, message = http.StatusConflict, "This page changed. Please refresh it and try again."
	case errors.Is(err, services.ErrLearningUnavailable):
		status, message = http.StatusServiceUnavailable, services.ErrLearningUnavailable.Error()
	default:
		switch err.Error() {
		case "finish your checklist before completing this journey", "resume this learning journey before changing it", "resume this journey before asking a question", "progress is calculated from your checklist", "choose a farming activity", "choose your experience level", "another journey for this farming type is already active":
			message = err.Error()
		}
	}
	if wantsJSON(r) {
		writeError(w, status, message)
		return
	}
	http.Error(w, message, status)
}
func (h *LearningHandler) render(w http.ResponseWriter, name string, data any) {
	if err := render.RenderTemplates(w, name, data); err != nil {
		log.Printf("render learning: %v", err)
		http.Error(w, "We could not open this page", http.StatusInternalServerError)
	}
}
