package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Negotiation struct {
	service *services.NegotiationService
}

func NewNegotiationHandler(service *services.NegotiationService) *Negotiation {
	return &Negotiation{service: service}
}

type NegotiationListPageData struct {
	Negotiations []models.Negotiation
	UserID       int
}

func (h *Negotiation) ListHandler(w http.ResponseWriter, r *http.Request) {
	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	negotiations, err := h.service.MyNegotiations(farmer.ID)
	if err != nil {
		log.Println("failed to load negotiations:", err)
	}

	data := NegotiationListPageData{Negotiations: negotiations, UserID: farmer.ID}
	if err := render.RenderTemplates(w, "negotiations.html", data); err != nil {
		log.Println("render error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// StartHandler is posted to from the Marketplace page's "Negotiate" form.
func (h *Negotiation) StartHandler(w http.ResponseWriter, r *http.Request) {
	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cropID, _ := strconv.Atoi(r.FormValue("crop_id"))
	quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	price, _ := strconv.ParseFloat(r.FormValue("offer_price"), 64)
	message := r.FormValue("message")

	negotiation, err := h.service.StartNegotiation(farmer.ID, cropID, quantity, price, message)
	if err != nil {
		log.Println("failed to start negotiation:", err)
		http.Redirect(w, r, "/marketplace", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/negotiation?id="+strconv.Itoa(negotiation.ID), http.StatusSeeOther)
}

type NegotiationThreadPageData struct {
	Negotiation *models.Negotiation
	Messages    []models.NegotiationMessage
	UserID      int
	Error       string
	TimeLeft    string
}

// ThreadHandler shows the chat-style negotiation thread and handles
// sending offers, accepting, and rejecting.
func (h *Negotiation) ThreadHandler(w http.ResponseWriter, r *http.Request) {
	farmer, ok := middleware.FarmerFromContext(r)
	if !ok || farmer == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	switch r.Method {
	case http.MethodGet:
		h.render(w, id, farmer.ID, "")

	case http.MethodPost:
		action := r.FormValue("action")

		var err error
		switch action {
		case "offer":
			price, _ := strconv.ParseFloat(r.FormValue("offer_price"), 64)
			message := r.FormValue("message")
			err = h.service.SendOffer(id, farmer.ID, price, message)
		case "accept":
			err = h.service.Accept(id, farmer.ID)
		case "reject":
			err = h.service.Reject(id, farmer.ID)
		}

		if err != nil {
			h.render(w, id, farmer.ID, err.Error())
			return
		}

		http.Redirect(w, r, "/negotiation?id="+strconv.Itoa(id), http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Negotiation) render(w http.ResponseWriter, negotiationID, userID int, errMsg string) {
	negotiation, messages, err := h.service.Thread(negotiationID)
	if err != nil || negotiation == nil {
		http.Error(w, "Negotiation not found", http.StatusNotFound)
		return
	}

	timeLeft := negotiation.TimeLeft()
	timeLeftStr := "Expired"
	if timeLeft > 0 {
		hours := int(timeLeft.Hours())
		minutes := int(timeLeft.Minutes()) % 60
		timeLeftStr = strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m left"
	}

	data := NegotiationThreadPageData{
		Negotiation: negotiation,
		Messages:    messages,
		UserID:      userID,
		Error:       errMsg,
		TimeLeft:    timeLeftStr,
	}

	if err := render.RenderTemplates(w, "negotiation.html", data); err != nil {
		log.Println("render error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
