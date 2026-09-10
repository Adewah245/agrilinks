package handlers

import (
	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
	"log"
	"net/http"
	"strconv"
)

type DeliveryHandler struct{ service *services.DeliveryService }

func NewDeliveryHandler(service *services.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

type DeliveryPageData struct {
	Deliveries []models.Delivery
	Tracking   []models.DeliveryTracking
	SelectedID int
	Error      string
}

func (h *DeliveryHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !user.IsBuyer() {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	ds, err := h.service.MyDeliveries(user.ID)
	data := DeliveryPageData{Deliveries: ds}
	if err != nil {
		data.Error = "Could not load deliveries"
	}
	if raw := r.URL.Query().Get("id"); raw != "" {
		id, e := strconv.Atoi(raw)
		if e != nil || id <= 0 {
			data.Error = "Invalid delivery"
		} else {
			data.SelectedID = id
			data.Tracking, err = h.service.MyTracking(user.ID, id)
			if err != nil {
				data.Error = "Could not load delivery tracking"
			}
		}
	}
	if err := render.RenderTemplates(w, "deliveries.html", data); err != nil {
		log.Printf("delivery page render: %v", err)
		http.Error(w, "Internal Server Error", 500)
	}
}
