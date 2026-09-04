package handlers

import (
	"encoding/json"
	"net/http"

	"backend/internal/chat"
)

type PresenceHandler struct {
	tracker *chat.Tracker
}

func NewPresenceHandler(tracker *chat.Tracker) *PresenceHandler {
	return &PresenceHandler{
		tracker: tracker,
	}
}

// PresenceResponse matches the payload expected by static/js/chat.js
type PresenceResponse struct {
	OnlineUserIDs []int64 `json:"online_user_ids"`
}

// GetPresence serves the active user IDs as JSON.
func (h *PresenceHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch active user IDs from tracker
	onlineIDs := h.tracker.GetOnlineUserIDs()
	if onlineIDs == nil {
		onlineIDs = []int64{}
	}

	response := PresenceResponse{
		OnlineUserIDs: onlineIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}
