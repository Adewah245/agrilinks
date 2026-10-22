package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	SearchTerm    string
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
			searchTerm := strings.TrimSpace(
				r.FormValue("search_term"),
			)

			if h.events != nil {
				_ = h.events.RecordSearch(
					searchTerm,
					r.FormValue("normalized_produce"),
					user.Location,
					&uid,
					"",
				)
			}

			h.renderWithSearch(
				w,
				user.IsBuyer(),
				user.ID,
				"",
				"",
				searchTerm,
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
				"Choose a valid product.",
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
				"Enter a valid amount.",
			)
			return
		}

		if err := h.order.PlaceOrder(
			user.ID,
			productID,
			quantity,
			r.FormValue("checkout_key"),
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
			"checkout_started",
			&productID,
			user.ID,
			r,
			fmt.Sprintf(
				`{"quantity":%g}`,
				quantity,
			),
		)

		http.Redirect(w, r, "/payments", http.StatusSeeOther)
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
	h.renderWithSearch(
		w,
		isBuyer,
		currentUserID,
		message,
		errMsg,
		"",
	)
}

// renderWithSearch loads either the complete marketplace or the products that
// match the buyer's search term, then renders the shared marketplace page.
func (h *Marketplace) renderWithSearch(
	w http.ResponseWriter,
	isBuyer bool,
	currentUserID int,
	message string,
	errMsg string,
	searchTerm string,
) {
	var products []models.Product
	searchTerm = strings.TrimSpace(searchTerm)

	if h.product != nil {
		var err error

		if searchTerm == "" {
			products, err = h.product.AvailableProducts()
		} else {
			products, err = h.product.SearchAvailable(searchTerm)
		}

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
		SearchTerm:    searchTerm,
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
