package handlers

import (
	"log"
	"net/http"

	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

// Sales handles requests for the authenticated user's sales page.
type Sales struct {
	order *services.OrderService
}

// SalesPageData contains the data displayed on the sales page.
type SalesPageData struct {
	OwnerName string
	Sales     interface{}
	Total     float64
	Count     int
}

// NewSalesHandler creates a new Sales handler.
func NewSalesHandler(order *services.OrderService) *Sales {
	return &Sales{
		order: order,
	}
}

// Handler loads the authenticated user's sales and renders
// the sales page.
//
// Sales are loaded using the "user" owner type because this
// handler represents sales made from products owned directly
// by the authenticated user.
func (h *Sales) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// ========================================================
	// 1. METHOD CHECK
	// ========================================================

	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// ========================================================
	// 2. GET AUTHENTICATED USER
	// ========================================================

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

	// ========================================================
	// 3. BUYERS DO NOT HAVE SELLER SALES
	// ========================================================

	if user.IsBuyer() {
		http.Redirect(
			w,
			r,
			"/profile",
			http.StatusSeeOther,
		)
		return
	}

	// ========================================================
	// 4. LOAD USER SALES
	// ========================================================

	sales, err := h.order.MySales(
		"user",
		user.ID,
	)
	if err != nil {
		log.Println("failed to load sales:", err)

		http.Error(
			w,
			"Unable to load sales",
			http.StatusInternalServerError,
		)
		return
	}

	// ========================================================
	// 5. CALCULATE SALES SUMMARY
	// ========================================================

	data := SalesPageData{
		OwnerName: user.FullName,
		Sales:     sales,
		Count:     len(sales),
	}

	for _, sale := range sales {
		data.Total += sale.TotalPrice
	}

	// ========================================================
	// 6. RENDER SALES PAGE
	// ========================================================

	if err := render.RenderTemplates(
		w,
		"sales.html",
		data,
	); err != nil {
		log.Println(
			"failed to render sales page:",
			err,
		)
		return
	}
}
