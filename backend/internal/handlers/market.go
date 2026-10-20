package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

// Marketplace handles marketplace product browsing, searching,
// and product purchase requests.
type Marketplace struct {
	product *services.ProductService
	order   *services.OrderService
	events  *repository.MarketEventRepository
}

// NewMarketplaceHandler creates a new Marketplace handler.
func NewMarketplaceHandler(
	product *services.ProductService,
	order *services.OrderService,
	events *repository.MarketEventRepository,
) *Marketplace {
	return &Marketplace{
		product: product,
		order:   order,
		events:  events,
	}
}

// MarketplacePageData contains the data passed to marketplace.html.
type MarketplacePageData struct {
	Products      []models.Product
	IsBuyer       bool
	CurrentUserID int
	Message       string
	Error         string
}

// MarketplaceHandler renders the marketplace and processes marketplace
// search and purchase requests.
func (h *Marketplace) MarketplaceHandler(
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

	switch r.Method {
	case http.MethodGet:
		log.Println("User visited marketplace")

		h.record(
			"marketplace_view",
			nil,
			user.ID,
			r,
		)

		h.render(
			w,
			user.IsBuyer(),
			user.ID,
			"",
			"",
		)
		return

	case http.MethodPost:
		// Marketplace search.
		if r.FormValue("action") == "search" {
			uid := user.ID

			if h.events != nil {
				_ = h.events.RecordSearch(
					r.FormValue("search_term"),
					r.FormValue("normalized_produce"),
					user.Location,
					&uid,
					"",
				)
			}

			h.render(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				"",
			)
			return
		}

		// Only buyers can purchase products.
		if !user.IsBuyer() {
			h.render(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				"Only buyer accounts can place orders",
			)
			return
		}

		productID, err := strconv.Atoi(
			r.FormValue("product_id"),
		)

		if err != nil || productID <= 0 {
			h.render(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				"Invalid product",
			)
			return
		}

		quantity, err := strconv.ParseFloat(
			r.FormValue("quantity"),
			64,
		)

		if err != nil || quantity <= 0 {
			h.render(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				"Invalid quantity",
			)
			return
		}

		if err := h.order.PlaceOrder(
			user.ID,
			productID,
			quantity,
		); err != nil {
			h.render(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				err.Error(),
			)
			return
		}

		// Record the completed checkout using the new product-based
		// marketplace architecture.
		h.record(
			"checkout_completed",
			&productID,
			user.ID,
			r,
			fmt.Sprintf(
				`{"quantity":%g}`,
				quantity,
			),
		)

		h.render(
			w,
			user.IsBuyer(),
			user.ID,
			"Order placed successfully!",
			"",
		)
		return

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}
}

// record records a marketplace event when the event repository is available.
func (h *Marketplace) record(
	eventType string,
	productID *int,
	userID int,
	r *http.Request,
	metadata ...string,
) {
	if h.events == nil {
		return
	}

	uid := userID
	meta := "{}"

	if len(metadata) > 0 && metadata[0] != "" {
		meta = metadata[0]
	}

	_ = h.events.Record(
		&models.MarketEvent{
			EventType: eventType,
			ProductID: productID,
			UserID:    &uid,
			Metadata:  json.RawMessage(meta),
		},
	)
}

// render loads available products and renders the marketplace page.
func (h *Marketplace) render(
	w http.ResponseWriter,
	isBuyer bool,
	currentUserID int,
	message string,
	errMsg string,
) {
	var products []models.Product

	if h.product != nil {
		var err error

		products, err = h.product.AvailableProducts()
		if err != nil {
			log.Println(
				"failed to load products:",
				err,
			)
		}
	}

	data := MarketplacePageData{
		Products:      products,
		IsBuyer:       isBuyer,
		CurrentUserID: currentUserID,
		Message:       message,
		Error:         errMsg,
	}

	if err := render.RenderTemplates(
		w,
		"marketplace.html",
		data,
	); err != nil {
		log.Println(
			"marketplace render error:",
			err,
		)
		return
	}
}
