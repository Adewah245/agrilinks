package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/repository"
)

// LocationHandler handles API requests for Nigerian states and LGAs.
type LocationHandler struct {
	locationRepo *repository.LocationRepository
}

// NewLocationHandler creates a new LocationHandler.
func NewLocationHandler(
	locationRepo *repository.LocationRepository,
) *LocationHandler {
	return &LocationHandler{
		locationRepo: locationRepo,
	}
}

// GetStatesHandler returns all Nigerian states and the Federal Capital Territory.
//
// GET /api/states
func (h *LocationHandler) GetStatesHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	states, err := h.locationRepo.GetStates()
	if err != nil {
		http.Error(
			w,
			"failed to load states",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(states); err != nil {
		// The response may already have been partially written at this
		// point, so there is no need to call http.Error again.
		return
	}
}

// GetLGAsHandler returns all LGAs belonging to a specified state.
//
// GET /api/states/{stateID}/lgas
func (h *LocationHandler) GetLGAsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Expected path:
	//
	// /api/states/7/lgas
	parts := strings.Split(
		strings.Trim(r.URL.Path, "/"),
		"/",
	)

	if len(parts) != 4 ||
		parts[0] != "api" ||
		parts[1] != "states" ||
		parts[3] != "lgas" {
		http.NotFound(w, r)
		return
	}

	stateID, err := strconv.Atoi(parts[2])
	if err != nil || stateID <= 0 {
		http.Error(
			w,
			"invalid state ID",
			http.StatusBadRequest,
		)
		return
	}

	lgas, err := h.locationRepo.GetLGAsByState(stateID)
	if err != nil {
		http.Error(
			w,
			"failed to load LGAs",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(lgas); err != nil {
		// The response may already have been partially written at this
		// point, so there is no need to call http.Error again.
		return
	}
}
