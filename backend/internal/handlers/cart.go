package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

// ============================================================
// CART HANDLER
//
// Responsibility:
// - Display the buyer's cart.
// - Display products accepted through negotiation.
// - Remove products from the cart.
// - Protect the cart so users can only access their own items.
// ============================================================

// Cart handles HTTP requests related to the buyer's cart.
type Cart struct {
	service *services.CartService
	events  *repository.MarketEventRepository
}

// ============================================================
// CREATE CART HANDLER
//
// Connects the Cart Handler to the Cart Service.
// ============================================================

// NewCartHandler creates a new Cart handler.
func NewCartHandler(
	service *services.CartService,
	events *repository.MarketEventRepository,
) *Cart {
	return &Cart{
		service: service,
		events:  events,
	}
}

// ============================================================
// CART PAGE DATA
//
// This is the data that will be sent to cart.html.
// ============================================================

// CartPageData contains the data rendered on the cart page.
type CartPageData struct {
	Items  []models.CartItem
	UserID int
	Error  string
	Total  float64
}

// ============================================================
// CART HANDLER
//
// GET:
//     Displays the buyer's cart.
//
// POST:
//     Handles cart actions such as removing an item.
// ============================================================

// Handler displays the authenticated buyer's cart and handles
// cart actions such as removing an item.
func (h *Cart) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// ========================================================
	// 1. GET LOGGED-IN USER
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
	// 2. CART IS FOR BUYERS
	//
	// Only users whose account role allows them to act as buyers
	// should access the buyer cart.
	// ========================================================

	if !user.CanBuy() {
		http.Redirect(
			w,
			r,
			"/marketplace",
			http.StatusSeeOther,
		)
		return
	}

	// ========================================================
	// 3. HANDLE HTTP METHOD
	// ========================================================

	switch r.Method {

	case http.MethodGet:

		// ----------------------------------------------------
		// GET CART
		// ----------------------------------------------------

		log.Println("User visited Cart")

		// Record that the authenticated user opened their cart.
		// Opening the cart is not the same as starting checkout,
		// so checkout_started is intentionally not recorded here.
		if h.events != nil {
			userID := user.ID

			_ = h.events.Record(
				&models.MarketEvent{
					EventType: "cart_view",
					UserID:    &userID,
				},
			)
		}

		h.render(
			w,
			user.ID,
			"",
		)

	case http.MethodPost:

		// ----------------------------------------------------
		// POST CART ACTION
		// ----------------------------------------------------

		action := r.FormValue("action")

		switch action {
		case "add":
			productID, _ := strconv.Atoi(r.FormValue("product_id"))
			quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
			if err := h.service.AddProduct(user.ID, productID, quantity); err != nil {
				h.render(w, user.ID, err.Error())
				return
			}
			http.Redirect(w, r, "/cart", http.StatusSeeOther)

		// ====================================================
		// REMOVE ITEM
		// ====================================================

		case "remove":

			cartID, err := strconv.Atoi(
				r.FormValue("cart_id"),
			)

			if err != nil || cartID <= 0 {
				h.render(
					w,
					user.ID,
					"Invalid cart item",
				)
				return
			}

			// ------------------------------------------------
			// Remove item through the Cart Service.
			//
			// The service makes sure the item belongs
			// to the logged-in buyer.
			// ------------------------------------------------

			if err := h.service.RemoveFromCart(
				user.ID,
				cartID,
			); err != nil {

				log.Println(
					"failed to remove cart item:",
					err,
				)

				h.render(
					w,
					user.ID,
					err.Error(),
				)

				return
			}

			// Record the successful removal.
			if h.events != nil {
				userID := user.ID

				_ = h.events.Record(
					&models.MarketEvent{
						EventType: "cart_item_removed",
						UserID:    &userID,
					},
				)
			}

			// ------------------------------------------------
			// Return to cart after successful removal.
			// ------------------------------------------------

			http.Redirect(
				w,
				r,
				"/cart",
				http.StatusSeeOther,
			)

		default:

			h.render(
				w,
				user.ID,
				"Unknown cart action",
			)
		}

	default:

		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

// ============================================================
// RENDER CART
//
// Responsibility:
// - Load the current buyer's cart.
// - Prepare the data.
// - Render cart.html.
// ============================================================

// render loads the buyer's cart, prepares the page data,
// and renders the cart template.
func (h *Cart) render(
	w http.ResponseWriter,
	buyerID int,
	errMsg string,
) {
	// ========================================================
	// 1. LOAD CART
	// ========================================================

	items, err := h.service.MyCart(buyerID)

	if err != nil {

		log.Println(
			"failed to load cart:",
			err,
		)

		errMsg = err.Error()
	}

	// ========================================================
	// 2. PREPARE PAGE DATA
	// ========================================================

	data := CartPageData{
		Items:  items,
		UserID: buyerID,
		Error:  errMsg,
	}

	_, data.Total = cartSummary(items)

	// ========================================================
	// 3. RENDER CART HTML
	// ========================================================

	if err := render.RenderTemplates(
		w,
		"cart.html",
		data,
	); err != nil {

		log.Println(
			"cart render error:",
			err,
		)

		// Do not call http.Error here because the template
		// renderer may already have written the response.
		// Attempting another response can produce a
		// superfluous response.WriteHeader warning.
		return
	}
}
