package handlers

import (
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type DeliveryHandler struct {
	service *services.DeliveryService
}

func NewDeliveryHandler(service *services.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

type DeliveryPageData struct {
	User *models.User

	Deliveries        []models.Delivery
	EligibleOrders    []models.DeliveryOrderOption
	SellerDeliveries  []models.Delivery
	PartnerDeliveries []models.Delivery
	AllDeliveries     []models.Delivery

	PartnerAccounts   []models.DeliveryPartner
	AvailablePartners []models.DeliveryPartner
	PartnersForReview []models.DeliveryPartner

	Tracking   []models.DeliveryTracking
	SelectedID int
	Error      string
	Notice     string
}

// Handler renders the delivery workspace appropriate to the signed-in account.
// Buyers request and follow deliveries, sellers arrange them, partners update
// them, and administrators can see the full queue.
func (h *DeliveryHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}

	data := DeliveryPageData{
		User:   user,
		Error:  deliveryErrorMessage(r.URL.Query().Get("error")),
		Notice: deliveryNoticeMessage(r.URL.Query().Get("notice")),
	}

	if user.IsBuyer() {
		data.Deliveries, _ = h.service.MyDeliveries(user.ID)
		data.EligibleOrders, _ = h.service.EligibleOrders(user.ID)
	} else {
		data.SellerDeliveries, _ = h.service.SellerDeliveries(user.ID)
		data.AvailablePartners, _ = h.service.AvailablePartners()
	}

	data.PartnerAccounts, _ = h.service.PartnerAccounts(user.ID)
	if len(data.PartnerAccounts) > 0 {
		data.PartnerDeliveries, _ = h.service.PartnerDeliveries(user.ID)
	}

	if user.IsAdministrator() {
		data.AllDeliveries, _ = h.service.AllDeliveries(user)
		data.PartnersForReview, _ = h.service.PartnersForReview(user)
	}

	if raw := r.URL.Query().Get("id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			data.Error = "That delivery link is not valid."
		} else {
			data.SelectedID = id
			tracking, err := h.service.DeliveryTracking(user, id)
			if err != nil {
				data.Error = deliveryErrorMessage(deliveryErrorCode(err))
			} else {
				data.Tracking = tracking
			}
		}
	}

	if err := render.RenderTemplates(w, "deliveries.html", data); err != nil {
		log.Printf("delivery page render: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RequestDeliveryHandler creates a delivery request for one paid order.
func (h *DeliveryHandler) RequestDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectDelivery(w, r, "/deliveries", "invalid", "")
		return
	}

	orderID, _ := strconv.Atoi(r.FormValue("order_id"))
	_, err := h.service.RequestDelivery(user, orderID, r.FormValue("destination_address"))
	if err != nil {
		redirectDelivery(w, r, "/deliveries", deliveryErrorCode(err), "")
		return
	}
	redirectDelivery(w, r, "/deliveries", "", "request_created")
}

// AssignPartnerHandler lets a seller or administrator assign an approved
// logistics partner.
func (h *DeliveryHandler) AssignPartnerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectDelivery(w, r, "/deliveries", "invalid", "")
		return
	}

	deliveryID, _ := strconv.Atoi(r.FormValue("delivery_id"))
	partnerID, _ := strconv.Atoi(r.FormValue("partner_id"))
	if err := h.service.AssignPartner(user, deliveryID, partnerID); err != nil {
		redirectDelivery(w, r, "/deliveries", deliveryErrorCode(err), "")
		return
	}
	redirectDelivery(w, r, "/deliveries", "", "partner_assigned")
}

// UpdateStatusHandler moves a delivery to its next step.
func (h *DeliveryHandler) UpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectDelivery(w, r, "/deliveries", "invalid", "")
		return
	}

	deliveryID, _ := strconv.Atoi(r.FormValue("delivery_id"))
	if err := h.service.UpdateStatus(user, deliveryID, r.FormValue("status")); err != nil {
		redirectDelivery(w, r, "/deliveries", deliveryErrorCode(err), "")
		return
	}
	redirectDelivery(w, r, "/deliveries", "", "status_updated")
}

// TrackingHandler appends a partner location update.
func (h *DeliveryHandler) TrackingHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectDelivery(w, r, "/deliveries", "invalid", "")
		return
	}

	deliveryID, _ := strconv.Atoi(r.FormValue("delivery_id"))
	latitude, latitudeErr := strconv.ParseFloat(strings.TrimSpace(r.FormValue("latitude")), 64)
	longitude, longitudeErr := strconv.ParseFloat(strings.TrimSpace(r.FormValue("longitude")), 64)
	if latitudeErr != nil || longitudeErr != nil {
		redirectDelivery(w, r, "/deliveries", "invalid_location", "")
		return
	}

	err := h.service.AddTracking(
		user,
		deliveryID,
		latitude,
		longitude,
		r.FormValue("location_name"),
		r.FormValue("note"),
	)
	if err != nil {
		redirectDelivery(w, r, "/deliveries", deliveryErrorCode(err), "")
		return
	}
	redirectDelivery(w, r, "/deliveries", "", "tracking_added")
}

// PartnerPageHandler shows and registers delivery partner accounts.
func (h *DeliveryHandler) PartnerPageHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			redirectDelivery(w, r, "/delivery-partner", "invalid", "")
			return
		}
		stateID, stateErr := optionalPositiveID(r.FormValue("state_id"))
		lgaID, lgaErr := optionalPositiveID(r.FormValue("lga_id"))
		if stateErr != nil || lgaErr != nil || (lgaID != nil && stateID == nil) {
			redirectDelivery(w, r, "/delivery-partner", "partner_invalid", "")
			return
		}
		_, err := h.service.RegisterPartner(user, models.DeliveryPartnerInput{
			Name:               r.FormValue("name"),
			Phone:              r.FormValue("phone"),
			Email:              r.FormValue("email"),
			VehicleType:        r.FormValue("vehicle_type"),
			RegistrationNumber: r.FormValue("registration_number"),
			StateID:            stateID,
			LGAID:              lgaID,
		})
		if err != nil {
			redirectDelivery(w, r, "/delivery-partner", "partner_invalid", "")
			return
		}
		redirectDelivery(w, r, "/delivery-partner", "", "partner_registered")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	data := DeliveryPageData{
		User:            user,
		Error:           deliveryErrorMessage(r.URL.Query().Get("error")),
		Notice:          deliveryNoticeMessage(r.URL.Query().Get("notice")),
		PartnerAccounts: []models.DeliveryPartner{},
	}
	data.PartnerAccounts, _ = h.service.PartnerAccounts(user.ID)
	if user.IsAdministrator() {
		data.PartnersForReview, _ = h.service.PartnersForReview(user)
	}

	if err := render.RenderTemplates(w, "delivery_partner.html", data); err != nil {
		log.Printf("delivery partner page render: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// PartnerReviewHandler lets an administrator approve, pause, or suspend a
// delivery partner.
func (h *DeliveryHandler) PartnerReviewHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentDeliveryUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectDelivery(w, r, "/delivery-partner", "invalid", "")
		return
	}

	partnerID, _ := strconv.Atoi(r.FormValue("partner_id"))
	var err error
	switch strings.ToLower(strings.TrimSpace(r.FormValue("action"))) {
	case "approve":
		err = h.service.SetPartnerVerified(user, partnerID, true)
	case "unapprove":
		err = h.service.SetPartnerVerified(user, partnerID, false)
	case "activate":
		err = h.service.SetPartnerStatus(user, partnerID, "active")
	case "pause":
		err = h.service.SetPartnerStatus(user, partnerID, "inactive")
	case "suspend":
		err = h.service.SetPartnerStatus(user, partnerID, "suspended")
	default:
		redirectDelivery(w, r, "/delivery-partner", "invalid", "")
		return
	}
	if err != nil {
		redirectDelivery(w, r, "/delivery-partner", deliveryErrorCode(err), "")
		return
	}
	redirectDelivery(w, r, "/delivery-partner", "", "partner_reviewed")
}

func currentDeliveryUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil, false
	}
	return user, true
}

func redirectDelivery(w http.ResponseWriter, r *http.Request, path, errorCode, noticeCode string) {
	values := url.Values{}
	if errorCode != "" {
		values.Set("error", errorCode)
	}
	if noticeCode != "" {
		values.Set("notice", noticeCode)
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

func optionalPositiveID(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return nil, errors.New("invalid location")
	}
	return &value, nil
}

func deliveryErrorCode(err error) string {
	switch {
	case errors.Is(err, services.ErrDeliveryNotFound):
		return "not_found"
	case errors.Is(err, services.ErrDeliveryForbidden):
		return "forbidden"
	case errors.Is(err, services.ErrDeliveryAlreadyExists):
		return "already_exists"
	case errors.Is(err, services.ErrDeliveryNotReady):
		return "not_ready"
	case errors.Is(err, services.ErrDeliveryStatusConflict):
		return "conflict"
	case errors.Is(err, services.ErrDeliveryAddressRequired):
		return "address_required"
	case errors.Is(err, services.ErrDeliveryPartnerNotApproved):
		return "partner_not_approved"
	case errors.Is(err, services.ErrDeliveryAssignmentClosed):
		return "assignment_closed"
	case errors.Is(err, services.ErrDeliveryTrackingClosed):
		return "tracking_closed"
	default:
		return "failed"
	}
}

func deliveryErrorMessage(code string) string {
	switch strings.TrimSpace(code) {
	case "not_found":
		return "We could not find that delivery or order."
	case "forbidden":
		return "You do not have access to make that delivery change."
	case "already_exists":
		return "Delivery has already been arranged for this order."
	case "not_ready":
		return "Please pay for this order before arranging delivery."
	case "conflict":
		return "Someone already updated this delivery. Refresh to see the latest step."
	case "address_required":
		return "Enter the address where the delivery should go."
	case "partner_not_approved":
		return "Choose a delivery partner that has been approved and is active."
	case "assignment_closed":
		return "A partner can no longer be changed after pickup."
	case "tracking_closed":
		return "Tracking has already closed for this delivery."
	case "invalid_location":
		return "Enter a valid latitude and longitude for the tracking update."
	case "partner_invalid":
		return "Enter the delivery partner name and phone number."
	case "invalid":
		return "That request was not complete. Please check the form and try again."
	case "failed":
		return "We could not update the delivery. Please try again."
	default:
		return ""
	}
}

func deliveryNoticeMessage(code string) string {
	switch strings.TrimSpace(code) {
	case "request_created":
		return "Delivery request created. The seller can now assign a partner."
	case "partner_registered":
		return "Delivery partner registered. An administrator must approve it before it can be assigned."
	case "partner_assigned":
		return "Delivery partner assigned."
	case "status_updated":
		return "Delivery step updated."
	case "tracking_added":
		return "Tracking update added."
	case "partner_reviewed":
		return "Delivery partner updated."
	default:
		return ""
	}
}
