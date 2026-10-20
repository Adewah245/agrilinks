package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

// ProductHandler handles requests related to viewing one product.
type ProductHandler struct {
	product *services.ProductService
	events  *repository.MarketEventRepository
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(
	product *services.ProductService,
	events *repository.MarketEventRepository,
) *ProductHandler {
	return &ProductHandler{
		product: product,
		events:  events,
	}
}

// ProductDetailsHandler gets one product by ID and displays its details page.
//
// The handler loads the product and its images through ProductService.
// If a user is authenticated, their user ID is attached to the
// marketplace view event. Anonymous visitors are still recorded
// without a user ID.
func (h *ProductHandler) ProductDetailsHandler(
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
	// 2. GET PRODUCT ID
	// ========================================================

	idString := r.URL.Query().Get("id")

	productID, err := strconv.Atoi(idString)
	if err != nil || productID <= 0 {
		http.Error(
			w,
			"Invalid product ID",
			http.StatusBadRequest,
		)
		return
	}

	// ========================================================
	// 3. GET PRODUCT
	// ========================================================

	product, err := h.product.GetProduct(productID)
	if err != nil {
		log.Println("failed to load product:", err)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if product == nil {
		http.Error(
			w,
			"Product not found",
			http.StatusNotFound,
		)
		return
	}

	// ========================================================
	// 4. GET PRODUCT IMAGES
	// ========================================================

	images, err := h.product.GetProductImages(productID)
	if err != nil {
		log.Println("failed to load product images:", err)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	// ========================================================
	// 5. RECORD MARKETPLACE VIEW
	// ========================================================

	if h.events != nil {
		var userID *int

		// A visitor may be anonymous or authenticated.
		// When authenticated, record the user's ID with the event.
		user, ok := middleware.UserFromContext(r)

		if ok && user != nil {
			id := user.ID
			userID = &id
		}

		// RecordListingView handles the marketplace view event.
		// An empty session ID is used because session tracking is
		// no longer handled through a raw request query parameter.
		_ = h.events.RecordListingView(
			productID,
			userID,
			"",
			time.Hour,
		)
	}

	// ========================================================
	// 6. PREPARE TEMPLATE DATA
	// ========================================================

	data := ProductPageData{
		Product: product,
		Images:  images,
	}

	// ========================================================
	// 7. RENDER PRODUCT PAGE
	// ========================================================

	if err := render.RenderTemplates(
		w,
		"product.html",
		data,
	); err != nil {
		log.Println("render product page error:", err)
		return
	}
}

// ProductPageData contains the data passed to the product page template.
type ProductPageData struct {
	Product *models.Product
	Images  []models.ProductImage
	Error   string
}
