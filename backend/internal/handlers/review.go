package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

// ReviewHandler accepts product reviews from buyers.
type ReviewHandler struct {
	service *services.ReviewService
}

func NewReviewHandler(service *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

// CreateHandler records a rating for a product the buyer purchased.
func (h *ReviewHandler) CreateHandler(
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
		http.Error(w, "Only buyers can review products", http.StatusForbidden)
		return
	}

	productID, productErr := strconv.Atoi(r.FormValue("product_id"))
	orderID, orderErr := strconv.Atoi(r.FormValue("order_id"))
	rating, ratingErr := strconv.Atoi(r.FormValue("rating"))

	if productErr != nil || orderErr != nil || ratingErr != nil {
		http.Error(w, "Invalid review", http.StatusBadRequest)
		return
	}

	if _, err := h.service.ReviewProduct(
		user.ID,
		productID,
		orderID,
		rating,
		r.FormValue("comment"),
	); err != nil {
		log.Println("product review failed:", err)
		code := "failed"
		switch {
		case errors.Is(err, services.ErrReviewRequiresDeliveredOrder):
			code = "not_delivered"
		case errors.Is(err, services.ErrReviewAlreadyExists):
			code = "already_reviewed"
		}
		http.Redirect(w, r, "/orders?review_error="+code, http.StatusSeeOther)
		return
	}

	http.Redirect(
		w,
		r,
		"/product?id="+strconv.Itoa(productID)+"#reviews",
		http.StatusSeeOther,
	)
}

type TrustPageData struct {
	User      *models.User
	OwnerType string
	OwnerID   int
	Badges    []models.VerificationBadge
	Error     string
	Notice    string
}

// TrustHandler lets administrators issue and revoke public trust badges.
// Identity documents and NIN data are deliberately absent from this page.
func (h *ReviewHandler) TrustHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !user.IsAdministrator() {
		http.Error(w, "Only administrators can manage trust badges", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			redirectTrust(w, r, "invalid", "")
			return
		}

		ownerType := r.FormValue("owner_type")
		ownerID, _ := strconv.Atoi(r.FormValue("owner_id"))
		switch strings.ToLower(strings.TrimSpace(r.FormValue("action"))) {
		case "issue":
			expiresAt, err := parseBadgeExpiry(r.FormValue("expires_at"))
			if err != nil {
				redirectTrust(w, r, "invalid_expiry", "")
				return
			}
			_, err = h.service.IssueBadge(
				user,
				ownerType,
				ownerID,
				r.FormValue("badge_type"),
				r.FormValue("description"),
				expiresAt,
			)
			if err != nil {
				log.Println("issue trust badge failed:", err)
				if errors.Is(err, services.ErrBadgeOwnerNotFound) {
					redirectTrust(w, r, "owner_not_found", "")
					return
				}
				redirectTrust(w, r, "badge_failed", "")
				return
			}
			http.Redirect(w, r, trustOwnerURL(ownerType, ownerID, "issued"), http.StatusSeeOther)
			return
		case "revoke":
			badgeID, _ := strconv.Atoi(r.FormValue("badge_id"))
			if err := h.service.RevokeBadge(user, badgeID); err != nil {
				log.Println("revoke trust badge failed:", err)
				redirectTrust(w, r, "badge_failed", "")
				return
			}
			http.Redirect(w, r, trustOwnerURL(ownerType, ownerID, "revoked"), http.StatusSeeOther)
			return
		default:
			redirectTrust(w, r, "invalid", "")
			return
		}
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ownerType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("owner_type")))
	if ownerType == "" {
		ownerType = "user"
	}
	ownerID, _ := strconv.Atoi(r.URL.Query().Get("owner_id"))

	data := TrustPageData{
		User:      user,
		OwnerType: ownerType,
		OwnerID:   ownerID,
	}
	switch r.URL.Query().Get("error") {
	case "invalid":
		data.Error = "That badge request was not complete."
	case "invalid_expiry":
		data.Error = "Enter a valid expiry date."
	case "badge_failed":
		data.Error = "We could not update that trust badge."
	case "owner_not_found":
		data.Error = "That user or organization could not be found."
	}
	switch r.URL.Query().Get("notice") {
	case "issued":
		data.Notice = "Trust badge issued."
	case "revoked":
		data.Notice = "Trust badge removed from public display."
	}

	if ownerID > 0 {
		badges, err := h.service.ActiveBadges(ownerType, ownerID)
		if err != nil {
			log.Println("load trust badges failed:", err)
			data.Error = "We could not load those trust badges."
		} else {
			data.Badges = badges
		}
	}

	if err := render.RenderTemplates(w, "trust.html", data); err != nil {
		log.Println("trust page render error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// parseBadgeExpiry treats an HTML date as valid through the end of the chosen
// day. Storing midnight would make a badge chosen for "today" already expired
// before the administrator submits the form.
func parseBadgeExpiry(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	day, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return nil, err
	}
	expiresAt := day.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return &expiresAt, nil
}

func redirectTrust(w http.ResponseWriter, r *http.Request, errorCode, noticeCode string) {
	switch {
	case errorCode != "":
		http.Redirect(w, r, "/trust?error="+errorCode, http.StatusSeeOther)
	case noticeCode != "":
		http.Redirect(w, r, "/trust?notice="+noticeCode, http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/trust", http.StatusSeeOther)
	}
}

func trustOwnerURL(ownerType string, ownerID int, notice string) string {
	values := url.Values{}
	values.Set("owner_type", ownerType)
	values.Set("owner_id", strconv.Itoa(ownerID))
	values.Set("notice", notice)
	return "/trust?" + values.Encode()
}
