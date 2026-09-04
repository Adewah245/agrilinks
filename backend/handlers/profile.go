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

// Profile handles authenticated and public user profile requests.
type Profile struct {
	product *services.ProductService
	order   *services.OrderService
	users   *repository.UserRepository
}

// NewProfileHandler creates a new Profile handler.
func NewProfileHandler(
	product *services.ProductService,
	order *services.OrderService,
	users *repository.UserRepository,
) *Profile {
	return &Profile{
		product: product,
		order:   order,
		users:   users,
	}
}

// ProfilePageData contains the data passed to the profile template.
//
// The embedded User allows the template to access fields such as
// .FullName, .Phone, .Location, .Role, and .CreatedAt directly.
// Products, purchases, and sales contain role-specific profile data.
type ProfilePageData struct {
	*models.User
	IsOwner   bool
	Products  []models.Product
	Purchases []models.Order
	Sales     []models.Order
}

// ProfileHandler displays the authenticated user's profile or
// another user's public profile.
//
// When an id query parameter is supplied, the handler loads that
// user's profile from UserRepository. Otherwise, it displays the
// currently authenticated user's profile with their relevant
// products, purchases, or sales.
func (h *Profile) ProfileHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	// ========================================================
	// 1. GET AUTHENTICATED USER
	// ========================================================

	log.Println("User Visited Profile")

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
	// 2. CHECK FOR PUBLIC PROFILE
	// ========================================================

	idValue := r.URL.Query().Get("id")

	if idValue != "" {
		id, err := strconv.Atoi(idValue)
		if err != nil || id <= 0 {
			http.Error(
				w,
				"Invalid profile ID",
				http.StatusBadRequest,
			)
			return
		}

		profile, err := h.users.GetByID(id)
		if err != nil {
			log.Println("failed to load profile:", err)
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}

		if profile == nil {
			http.NotFound(w, r)
			return
		}

		user = profile
	}

	// ========================================================
	// 3. PREPARE PROFILE DATA
	// ========================================================

	data := ProfilePageData{
		User:    user,
		IsOwner: idValue == "",
	}

	// ========================================================
	// 4. LOAD ROLE-SPECIFIC DATA
	// ========================================================

	if !data.IsOwner {
		// Public profiles currently expose only the basic user
		// profile information. Private activity remains protected.
	} else if user.IsBuyer() {
		// ----------------------------------------------------
		// BUYER PROFILE
		// ----------------------------------------------------

		if h.order != nil {
			purchases, err := h.order.MyPurchases(user.ID)
			if err != nil {
				log.Println("failed to load purchases:", err)
			} else {
				data.Purchases = purchases
			}
		}
	} else {
		// ----------------------------------------------------
		// SELLER / FARMER / OTHER NON-BUYER PROFILE
		// ----------------------------------------------------

		if h.product != nil {
			products, err := h.product.MyProducts(
				"user",
				user.ID,
			)
			if err != nil {
				log.Println("failed to load products:", err)
			} else {
				data.Products = products
			}
		}

		if h.order != nil {
			sales, err := h.order.MySales("user", user.ID)
			if err != nil {
				log.Println("failed to load sales:", err)
			} else {
				data.Sales = sales
			}
		}
	}

	// ========================================================
	// 5. RENDER PROFILE PAGE
	// ========================================================

	if err := render.RenderTemplates(
		w,
		"profile.html",
		data,
	); err != nil {
		log.Println("profile page render error:", err)
		return
	}
}
