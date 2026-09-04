package handlers

import (
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

type Marketplace struct {
	product *services.ProductService
	order   *services.OrderService
	events  *repository.MarketEventRepository
}

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

type MarketplacePageData struct {
	Products      []models.Product
	IsBuyer       bool
	CurrentUserID int
	Message       string
	Error         string
}

func (h *Marketplace) MarketplaceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.FarmerFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {

	case http.MethodGet:
		log.Println("User Visited Marketplace")

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
					r.FormValue("session_id"),
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
			// Backward compatibility with existing marketplace forms
			// that may still submit crop_id.
			productID, err = strconv.Atoi(
				r.FormValue("crop_id"),
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

		// The event schema still calls the product identifier crop_id.
		h.record(
			"checkout_completed",
			&productID,
			user.ID,
			r,
			fmt.Sprintf(`{"quantity":%g}`, quantity),
		)

		h.render(
			w,
			user.IsBuyer(),
			user.ID,
			"Order placed successfully!",
			"",
		)

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

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

	sessionID, _ := r.Context().
		Value("session_id").
		(string)

	meta := "{}"

	if len(metadata) > 0 {
		meta = metadata[0]
	}

	_ = h.events.Record(
		&models.MarketEvent{
			EventType: eventType,
			CropID:    productID,
			UserID:    &uid,
			SessionID: sessionID,
			Metadata:  meta,
		},
	)
}

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
			"render error",
			err,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
	}
}
