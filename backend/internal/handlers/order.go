package handlers

import (
	"log"
	"net/http"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

// OrderHandler exposes buyer checkout and purchase history.
type OrderHandler struct {
	service *services.OrderService
}

func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

type OrderPageData struct {
	User   *models.User
	Orders []models.Order
	Error  string
}

// ListHandler displays the authenticated buyer's purchases.
func (h *OrderHandler) ListHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !user.IsBuyer() {
		http.Redirect(w, r, "/sales", http.StatusSeeOther)
		return
	}

	orders, err := h.service.MyPurchases(user.ID)

	data := OrderPageData{
		User:   user,
		Orders: orders,
	}

	switch r.URL.Query().Get("review_error") {
	case "not_delivered":
		data.Error = "You can review this product after the order has been delivered."
	case "already_reviewed":
		data.Error = "You have already reviewed this purchase."
	case "failed":
		data.Error = "We could not save your review. Please try again."
	}

	if err != nil {
		log.Println("failed to load orders:", err)
		data.Error = "Could not load orders."
	}

	if err := render.RenderTemplates(w, "orders.html", data); err != nil {
		log.Println("order page render error:", err)
	}
}

// PlaceHandler converts the buyer's cart into an order.
func (h *OrderHandler) PlaceHandler(
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
		http.Error(w, "Only buyers can place orders", http.StatusForbidden)
		return
	}

	if _, err := h.service.Checkout(user.ID); err != nil {
		log.Println("checkout failed:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}
