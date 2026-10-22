package handlers

import (
	"backend/internal/models"
	"fmt"
	"net/http"
	"strconv"
)

func (h *LearningHandler) Rooms(w http.ResponseWriter, r *http.Request) {
	userID, ok := getLearningUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	farmingID := 0
	if value := r.URL.Query().Get("farming_type_id"); value != "" {
		var err error
		farmingID, err = strconv.Atoi(value)
		if err != nil {
			http.Error(w, "Choose a farming activity", http.StatusBadRequest)
			return
		}
	}
	rooms, err := h.service.ListRooms(r.Context(), userID, farmingID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	types, err := h.service.GetFarmingTypes()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, "learning_rooms.html", struct {
		Rooms        []models.LearningRoom
		FarmingTypes []models.FarmingType
		SelectedType int
	}{rooms, types, farmingID})
}

func (h *LearningHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := learningPost(w, r)
	if !ok {
		return
	}
	farmingID, err := strconv.Atoi(r.FormValue("farming_type_id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	id, err := h.service.CreateRoom(r.Context(), userID, farmingID, r.FormValue("title"), r.FormValue("description"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/chat/view?id=%d", id), http.StatusSeeOther)
}

func (h *LearningHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := learningPost(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(r.FormValue("room_id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.service.JoinRoom(r.Context(), userID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/chat/view?id=%d", id), http.StatusSeeOther)
}
