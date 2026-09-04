package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type Profile struct {
	product *services.ProductService
	order   *services.OrderService
	farmers *repository.UserRepository
}

func NewProfileHandler(
	product *services.ProductService,
	order *services.OrderService,
	farmers *repository.UserRepository,
) *Profile {
	return &Profile{
		product: product,
		order:   order,
		farmers: farmers,
	}
}

// ProfilePageData embeds the user so the template can use
// .FullName, .Phone, .Location, .Role, .CreatedAt directly,
// plus role-specific data.
type ProfilePageData struct {
	*models.User
	IsOwner   bool
	Products  []models.Product
	Purchases []models.Order
	Sales     []models.Order
}

func (h *Profile) ProfileHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	log.Println("User Visited Profile")

	user, ok := middleware.FarmerFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if idValue := r.URL.Query().Get("id"); idValue != "" {
		id, err := strconv.Atoi(idValue)
		if err != nil || id <= 0 {
			http.Error(
				w,
				"Invalid profile ID",
				http.StatusBadRequest,
			)
			return
		}

		profile, err := h.farmers.GetByID(id)
		if err != nil || profile == nil {
			http.NotFound(w, r)
			return
		}

		user = profile
	}

	data := ProfilePageData{
		User:    user,
		IsOwner: r.URL.Query().Get("id") == "",
	}

	if !data.IsOwner {
		// Public profile: contact details only.
	} else if user.IsBuyer() {
		if h.order != nil {
			if purchases, err := h.order.MyPurchases(user.ID); err == nil {
				data.Purchases = purchases
			} else {
				log.Println("failed to load purchases:", err)
			}
		}
	} else {
		if h.product != nil {
			if products, err := h.product.MyProducts("user", user.ID); err == nil {
				data.Products = products
			} else {
				log.Println("failed to load products:", err)
			}
		}

		if h.order != nil {
			if sales, err := h.order.MySales(user.ID); err == nil {
				data.Sales = sales
			} else {
				log.Println("failed to load sales:", err)
			}
		}
	}

	if err := render.RenderTemplates(
		w,
		"profile.html",
		data,
	); err != nil {
		log.Println("err Render Problem", err)
		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}
}
