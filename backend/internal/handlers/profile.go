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

// Profile handles authenticated and public user profile requests.
type Profile struct {
	product      *services.ProductService
	order        *services.OrderService
	review       *services.ReviewService
	users        *repository.UserRepository
	notification *services.NotificationService
}

// NewProfileHandler creates a new Profile handler.
func NewProfileHandler(
	product *services.ProductService,
	order *services.OrderService,
	review *services.ReviewService,
	users *repository.UserRepository,
	notification *services.NotificationService,
) *Profile {
	return &Profile{
		product:      product,
		order:        order,
		review:       review,
		users:        users,
		notification: notification,
	}
}

// ProfilePageData contains the data passed to the profile template.
//
// The embedded User allows the template to access fields such as
// .FullName, .Phone, .Location, .Role, and .CreatedAt directly.
// Products, purchases, and sales contain role-specific profile data.
type ProfilePageData struct {
	*models.User
	IsOwner             bool
	Products            []models.Product
	Purchases           []models.Order
	Sales               []models.Order
	Badges              []models.VerificationBadge
	UnreadCount         int
	HasTransactionPIN   bool
	PINError, PINNotice string
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
	h.renderProfile(w, r, "")
}

func (h *Profile) PINHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not read the PIN form", http.StatusBadRequest)
		return
	}
	err := services.NewTransactionPINService(h.users).Setup(r.Context(), user, r.PostFormValue("current_password"), r.PostFormValue("transaction_pin"), r.PostFormValue("transaction_pin_confirmation"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.renderProfile(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/profile?notice=pin_created#profile-pin", http.StatusSeeOther)
}

func (h *Profile) renderProfile(w http.ResponseWriter, r *http.Request, pinError string) {
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
	viewerID := user.ID

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
		User:     user,
		IsOwner:  idValue == "",
		PINError: pinError,
	}
	if data.IsOwner {
		var err error
		data.HasTransactionPIN, err = h.users.HasTransactionPIN(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "Could not load wallet security", http.StatusInternalServerError)
			return
		}
		if r.URL.Query().Get("notice") == "pin_created" {
			data.PINNotice = "Your transaction PIN is ready. Keep it private."
		}
	}
	if h.notification != nil {
		if count, err := h.notification.GetUnreadCount(viewerID); err == nil {
			data.UnreadCount = count
		}
	}

	if h.review != nil {
		badges, badgeErr := h.review.ActiveBadges("user", user.ID)
		if badgeErr != nil {
			log.Println("failed to load profile badges:", badgeErr)
		} else {
			data.Badges = badges
		}
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

	if err := render.RenderPage(
		w,
		"profile.html",
		data,
	); err != nil {
		log.Println("profile page render error:", err)
		return
	}
}
