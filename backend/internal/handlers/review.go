package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/services"
)

// ReviewHandler accepts product reviews from buyers.
type ReviewHandler struct {
	service *services.ReviewService
}

func NewReviewHandler(service *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

// CreateHandler records a rating for a product the buyer purchased.
func (h *ReviewHandler) CreateHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !user.IsBuyer() {
		http.Error(w, "Only buyers can review products", http.StatusForbidden)
		return
	}

	productID, productErr := strconv.Atoi(r.FormValue("product_id"))
	orderID, orderErr := strconv.Atoi(r.FormValue("order_id"))
	rating, ratingErr := strconv.Atoi(r.FormValue("rating"))

	if productErr != nil || orderErr != nil || ratingErr != nil {
		http.Error(w, "Invalid review", http.StatusBadRequest)
		return
	}

	if _, err := h.service.ReviewProduct(
		user.ID,
		productID,
		orderID,
		rating,
		r.FormValue("comment"),
	); err != nil {
		log.Println("product review failed:", err)
		http.Error(w, "Could not save review", http.StatusBadRequest)
		return
	}

	http.Redirect(
		w,
		r,
		"/product?id="+strconv.Itoa(productID)+"#reviews",
		http.StatusSeeOther,
	)
}
