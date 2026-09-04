package handlers

import (
	"log"
	"net/http"

	"backend/internal/services"
	"backend/middleware"
)

// ProfileEdit handles profile update requests for authenticated users.
type ProfileEdit struct {
	auth *services.AuthService
}

// NewProfileEditHandler creates a new ProfileEdit handler.
func NewProfileEditHandler(auth *services.AuthService) *ProfileEdit {
	return &ProfileEdit{
		auth: auth,
	}
}

// Handler processes profile updates and optional profile photo uploads.
//
// Only authenticated users can update their profile. The handler
// delegates profile and photo updates to AuthService and redirects
// the user back to the profile page when processing is complete.
func (h *ProfileEdit) Handler(w http.ResponseWriter, r *http.Request) {
	// ========================================================
	// 1. GET AUTHENTICATED USER
	// ========================================================

	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// ========================================================
	// 2. METHOD CHECK
	// ========================================================

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// ========================================================
	// 3. PARSE FORM
	// ========================================================

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Println("form parse error:", err)

		http.Redirect(
			w,
			r,
			"/profile",
			http.StatusSeeOther,
		)
		return
	}

	// ========================================================
	// 4. GET PROFILE INFORMATION
	// ========================================================

	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	email := r.FormValue("email")
	location := r.FormValue("location")

	// ========================================================
	// 5. UPDATE PROFILE
	// ========================================================

	if err := h.auth.UpdateProfile(
		user.ID,
		fullName,
		phone,
		email,
		location,
	); err != nil {
		log.Println("profile update failed:", err)
	}

	// ========================================================
	// 6. SAVE NEW PROFILE PHOTO
	// ========================================================

	photoURL, err := saveUploadedFile(
		r,
		"passport",
		"users",
	)

	if err != nil {
		log.Println("profile photo upload failed:", err)
	} else if photoURL != "" {
		if err := h.auth.UpdatePhoto(
			user.ID,
			photoURL,
		); err != nil {
			log.Println("photo update failed:", err)
		}
	}

	// ========================================================
	// 7. RETURN TO PROFILE
	// ========================================================

	http.Redirect(
		w,
		r,
		"/profile",
		http.StatusSeeOther,
	)
}
