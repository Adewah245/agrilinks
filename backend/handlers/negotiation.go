package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

// Negotiation handles negotiation pages, messages, offers,
// offer responses, and negotiation closing.
type Negotiation struct {
	service *services.NegotiationService
	events  *repository.MarketEventRepository
}

// NewNegotiationHandler creates a new Negotiation handler.
func NewNegotiationHandler(
	service *services.NegotiationService,
	events *repository.MarketEventRepository,
) *Negotiation {
	return &Negotiation{
		service: service,
		events:  events,
	}
}

// NegotiationListPageData contains the data passed to the
// negotiations list page.
type NegotiationListPageData struct {
	Negotiations []models.Negotiation
	UserID       int
}

// ListHandler displays all negotiations involving the logged-in user.
func (h *Negotiation) ListHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

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

	negotiations, err := h.service.MyNegotiations(user.ID)
	if err != nil {
		log.Println(
			"failed to load negotiations:",
			err,
		)
	}

	data := NegotiationListPageData{
		Negotiations: negotiations,
		UserID:       user.ID,
	}

	if err := render.RenderTemplates(
		w,
		"negotiations.html",
		data,
	); err != nil {
		log.Println(
			"negotiation list render error:",
			err,
		)
		return
	}
}

// StartHandler starts a new negotiation from a marketplace product.
//
// POST /negotiation/start
func (h *Negotiation) StartHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

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

	// Get product ID.
	productID, err := strconv.Atoi(
		r.FormValue("product_id"),
	)

	if err != nil || productID <= 0 {
		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	// Get requested quantity.
	quantity, err := strconv.ParseFloat(
		r.FormValue("quantity"),
		64,
	)

	if err != nil || quantity <= 0 {
		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	// Get the first offer price.
	price, err := strconv.ParseFloat(
		r.FormValue("offer_price"),
		64,
	)

	if err != nil || price <= 0 {
		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	message := r.FormValue("message")

	// Create the negotiation.
	negotiation, err := h.service.StartNegotiation(
		user.ID,
		productID,
		quantity,
		price,
		message,
	)

	if err != nil {
		log.Println(
			"failed to start negotiation:",
			err,
		)

		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	// Record the negotiation event.
	if h.events != nil {
		uid := user.ID
		pid := productID

		_ = h.events.Record(
			&models.MarketEvent{
				EventType: "negotiation_started",
				ProductID: &pid,
				UserID:    &uid,
				Metadata: json.RawMessage(fmt.Sprintf(
					`{"quantity":%g,"offer_price":%g}`,
					quantity,
					price,
				)),
			},
		)
	}

	// Open the newly created negotiation.
	http.Redirect(
		w,
		r,
		"/negotiation?id="+strconv.Itoa(
			negotiation.ID,
		),
		http.StatusSeeOther,
	)
}

// NegotiationThreadPageData contains the data passed to the
// negotiation thread page.
type NegotiationThreadPageData struct {
	Negotiation *models.Negotiation
	Messages    []models.NegotiationMessage
	UserID      int
	Error       string
	TimeLeft    string
}

// ThreadHandler displays a negotiation thread and processes
// messages, offers, acceptances, rejections, and closing requests.
func (h *Negotiation) ThreadHandler(
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

	negotiationID, err := strconv.Atoi(
		r.URL.Query().Get("id"),
	)

	if err != nil || negotiationID <= 0 {
		http.Error(
			w,
			"Invalid negotiation ID",
			http.StatusBadRequest,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.render(
			w,
			negotiationID,
			user.ID,
			"",
		)
		return

	case http.MethodPost:
		action := r.FormValue("action")

		switch action {
		case "chat":
			h.handleChat(
				w,
				r,
				negotiationID,
				user.ID,
			)
			return

		case "offer":
			h.handleOffer(
				w,
				r,
				negotiationID,
				user.ID,
			)
			return

		case "accept":
			h.handleAccept(
				w,
				r,
				negotiationID,
				user.ID,
			)
			return

		case "reject":
			h.handleReject(
				w,
				r,
				negotiationID,
				user.ID,
			)
			return

		case "end":
			if err := h.service.End(
				negotiationID,
				user.ID,
			); err != nil {
				h.render(
					w,
					negotiationID,
					user.ID,
					err.Error(),
				)
				return
			}

		default:
			h.render(
				w,
				negotiationID,
				user.ID,
				"invalid negotiation action",
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/negotiation?id="+strconv.Itoa(
				negotiationID,
			),
			http.StatusSeeOther,
		)
		return

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

// handleChat sends a normal negotiation chat message.
func (h *Negotiation) handleChat(
	w http.ResponseWriter,
	r *http.Request,
	negotiationID int,
	userID int,
) {
	message := r.FormValue("message")

	if message == "" {
		h.render(
			w,
			negotiationID,
			userID,
			"message cannot be empty",
		)
		return
	}

	if err := h.service.SendMessage(
		negotiationID,
		userID,
		message,
	); err != nil {
		h.render(
			w,
			negotiationID,
			userID,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/negotiation?id="+strconv.Itoa(
			negotiationID,
		),
		http.StatusSeeOther,
	)
}

// handleOffer sends a new negotiation offer.
func (h *Negotiation) handleOffer(
	w http.ResponseWriter,
	r *http.Request,
	negotiationID int,
	userID int,
) {
	price, err := strconv.ParseFloat(
		r.FormValue("offer_price"),
		64,
	)

	if err != nil || price <= 0 {
		h.render(
			w,
			negotiationID,
			userID,
			"invalid offer price",
		)
		return
	}

	message := r.FormValue("message")

	if err := h.service.SendOffer(
		negotiationID,
		userID,
		price,
		message,
	); err != nil {
		h.render(
			w,
			negotiationID,
			userID,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/negotiation?id="+strconv.Itoa(
			negotiationID,
		),
		http.StatusSeeOther,
	)
}

// handleAccept accepts a specific negotiation offer.
func (h *Negotiation) handleAccept(
	w http.ResponseWriter,
	r *http.Request,
	negotiationID int,
	userID int,
) {
	offerID, err := strconv.Atoi(
		r.FormValue("offer_id"),
	)

	if err != nil || offerID <= 0 {
		h.render(
			w,
			negotiationID,
			userID,
			"invalid offer ID",
		)
		return
	}

	if err := h.service.Accept(
		negotiationID,
		offerID,
		userID,
	); err != nil {
		h.render(
			w,
			negotiationID,
			userID,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/negotiation?id="+strconv.Itoa(
			negotiationID,
		),
		http.StatusSeeOther,
	)
}

// handleReject rejects a specific negotiation offer.
func (h *Negotiation) handleReject(
	w http.ResponseWriter,
	r *http.Request,
	negotiationID int,
	userID int,
) {
	offerID, err := strconv.Atoi(
		r.FormValue("offer_id"),
	)

	if err != nil || offerID <= 0 {
		h.render(
			w,
			negotiationID,
			userID,
			"invalid offer ID",
		)
		return
	}

	if err := h.service.Reject(
		negotiationID,
		offerID,
		userID,
	); err != nil {
		h.render(
			w,
			negotiationID,
			userID,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/negotiation?id="+strconv.Itoa(
			negotiationID,
		),
		http.StatusSeeOther,
	)
}

// render loads a negotiation thread and renders the negotiation page.
//
// Authorization is enforced by the negotiation service when the
// thread is loaded, so this handler does not incorrectly treat an
// organization ID as a user ID.
func (h *Negotiation) render(
	w http.ResponseWriter,
	negotiationID int,
	userID int,
	errMsg string,
) {
	negotiation, messages, err := h.service.Thread(
		negotiationID,
	)

	if err != nil || negotiation == nil {
		http.Error(
			w,
			"Negotiation not found",
			http.StatusNotFound,
		)
		return
	}

	// Calculate the remaining offer-window time.
	timeLeft := negotiation.TimeLeft()
	timeLeftStr := "Expired"

	if timeLeft > 0 {
		hours := int(timeLeft.Hours())
		minutes := int(timeLeft.Minutes()) % 60

		timeLeftStr =
			strconv.Itoa(hours) +
				"h " +
				strconv.Itoa(minutes) +
				"m left"
	}

	data := NegotiationThreadPageData{
		Negotiation: negotiation,
		Messages:    messages,
		UserID:      userID,
		Error:       errMsg,
		TimeLeft:    timeLeftStr,
	}

	if err := render.RenderTemplates(
		w,
		"negotiation.html",
		data,
	); err != nil {
		log.Println(
			"negotiation render error:",
			err,
		)
	}
}
