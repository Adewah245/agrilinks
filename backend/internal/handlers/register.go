package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/render"
)

// UserReg contains registration information submitted by a new user.
//
// The structure represents the fields collected by the registration
// form before they are passed to AuthService for validation and
// account creation.
type UserReg struct {
	First_Name       string
	Last_Name        string
	Phone            string
	Email            string
	Password         string
	Confirm_Password string

	AccountType string

	OrganisationName string

	Purposes     []string
	FarmingTypes []string

	Country   string
	State     string
	LGA       string
	Community string
}

// RegisterHandler renders the registration page and processes
// new user registration requests.
//
// The handler collects registration information, validates the
// required form fields, saves the required profile picture, determines
// the appropriate user role, and delegates account creation to
// AuthService.
func (h *Register) RegisterHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {

	case http.MethodGet:
		types, err := h.learning.GetFarmingTypes()
		if err != nil {
			http.Error(w, "Could not load farming activities", 500)
			return
		}
		// ====================================================
		// DISPLAY REGISTRATION PAGE
		// ====================================================

		log.Println("[registration] page.opened")

		if err := render.RenderTemplates(
			w,
			"register.html",
			map[string]any{"FarmingTypes": types},
		); err != nil {
			log.Println("[registration] page.render_failed:", err)
			return
		}

	case http.MethodPost:
		log.Println("[registration] submission.received")
		// ====================================================
		// PARSE REGISTRATION FORM
		// ====================================================

		// Parse multipart form because registration includes
		// the selfie captured with the user's camera.
		r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Println("[registration] submission.invalid_form:", err)
			http.Error(
				w,
				"Invalid registration form",
				http.StatusBadRequest,
			)
			return
		}

		defer r.MultipartForm.RemoveAll()
		confirmation, cookieErr := r.Cookie(registrationCookieName)
		if cookieErr != nil || len(confirmation.Value) != 64 || strings.TrimSpace(r.FormValue("registration_code")) == "" {
			log.Println("[registration] submission.phone_confirmation_missing")
			http.Error(w, "Confirm your phone number with the SMS code before completing registration.", http.StatusBadRequest)
			return
		}

		// ====================================================
		// SAVE PROFILE PICTURE
		// ====================================================

		// Every user must have a profile picture.
		//
		// The upload directory is now "users" because the
		// application no longer stores users as farmers.
		photoURL, err := saveUploadedFile(
			r,
			"profile_picture",
			"users",
		)

		if err != nil || photoURL == "" {
			log.Printf("[registration] selfie.save_failed: missing=%t error=%v", photoURL == "", err)
			http.Error(
				w,
				"Take a selfie before completing registration.",
				http.StatusBadRequest,
			)
			return
		}
		log.Printf("[registration] selfie.saved: path=%s", photoURL)
		registered := false
		defer func() {
			if !registered {
				if directory, err := frontendDirectory(); err == nil {
					if err := os.Remove(filepath.Join(directory, "uploads", "users", filepath.Base(photoURL))); err != nil {
						log.Printf("[registration] selfie.cleanup_failed: path=%s error=%v", photoURL, err)
					} else {
						log.Printf("[registration] selfie.removed_after_failure: path=%s", photoURL)
					}
				}
			}
		}()

		// ====================================================
		// COLLECT REGISTRATION DATA
		// ====================================================

		user := UserReg{
			First_Name: strings.TrimSpace(
				r.FormValue("first-name"),
			),
			Last_Name: strings.TrimSpace(
				r.FormValue("last-name"),
			),
			Phone: strings.TrimSpace(
				r.FormValue("phone"),
			),
			Email: strings.TrimSpace(
				r.FormValue("email"),
			),

			Password: r.FormValue("password"),
			Confirm_Password: r.FormValue(
				"confirm-password",
			),

			AccountType: strings.TrimSpace(
				r.FormValue("account_type"),
			),

			OrganisationName: strings.TrimSpace(
				r.FormValue("organisation_name"),
			),

			Purposes:     r.Form["purpose"],
			FarmingTypes: r.Form["farm_type"],

			Country: strings.TrimSpace(
				r.FormValue("country"),
			),
			State: strings.TrimSpace(
				r.FormValue("state"),
			),
			LGA: strings.TrimSpace(
				r.FormValue("lga"),
			),
			Community: strings.TrimSpace(
				r.FormValue("community"),
			),
		}

		// ====================================================
		// BASIC VALIDATION
		// ====================================================

		if user.First_Name == "" ||
			user.Last_Name == "" ||
			user.Phone == "" ||
			user.Password == "" ||
			user.Confirm_Password == "" {

			log.Println("[registration] submission.required_details_missing")

			http.Error(
				w,
				"Required fields must not be empty",
				http.StatusBadRequest,
			)
			return
		}

		if user.Password != user.Confirm_Password {
			log.Println("[registration] submission.password_mismatch")

			http.Error(
				w,
				"Passwords do not match",
				http.StatusBadRequest,
			)
			return
		}

		// ====================================================
		// PURPOSE VALIDATION
		// ====================================================

		if len(user.Purposes) == 0 {
			http.Error(
				w,
				"Please select your purpose",
				http.StatusBadRequest,
			)
			return
		}

		// ====================================================
		// LOCATION VALIDATION
		// ====================================================

		if user.Country == "" ||
			user.State == "" ||
			user.LGA == "" ||
			user.Community == "" {

			http.Error(
				w,
				"Complete your location information",
				http.StatusBadRequest,
			)
			return
		}

		// ====================================================
		// DETERMINE USER ROLE
		// ====================================================

		role := "farmer"

		wantsToSell := false
		wantsToBuy := false
		wantsToLearn := false
		wantsToManageMinistry := false

		for _, purpose := range user.Purposes {
			switch strings.ToLower(strings.TrimSpace(purpose)) {
			case "sell":
				wantsToSell = true

			case "buy":
				wantsToBuy = true

			case "learn":
				wantsToLearn = true

			case "ministry":
				wantsToManageMinistry = true
			}
		}

		// ----------------------------------------------------
		// ORGANISATION ACCOUNT
		// ----------------------------------------------------
		//
		// An organization account represents an organization
		// rather than an individual user.
		//
		// AuthService requires organization accounts to use
		// the "organisation" role.

		if strings.EqualFold(
			user.AccountType,
			"organisation",
		) {
			role = "organisation"

			if user.OrganisationName == "" {
				http.Error(
					w,
					"Organisation name is required",
					http.StatusBadRequest,
				)
				return
			}
		} else if wantsToManageMinistry {
			// ------------------------------------------------
			// MINISTRY ACCOUNT
			// ------------------------------------------------
			role = "ministry"
		} else if wantsToBuy && !wantsToSell {
			// ------------------------------------------------
			// BUYER ONLY
			// ------------------------------------------------
			role = "buyer"
		} else if wantsToLearn && !wantsToBuy && !wantsToSell {
			// ------------------------------------------------
			// LEARNER ONLY
			// ------------------------------------------------
			role = "learner"
		} else {
			// ------------------------------------------------
			// FARMER / SELLER
			// ------------------------------------------------
			//
			// Users who sell, or who both buy and sell,
			// remain normal individual agricultural users.
			role = "farmer"
		}

		// ====================================================
		// LOCATION STORAGE
		// ====================================================
		//
		// The current AuthService accepts:
		//
		// Location = Community
		// LGA      = LGA
		// State    = State
		// Country  = Country
		//
		// This keeps the current registration flow compatible
		// with the users table.

		location := user.Community

		// ====================================================
		// REGISTER USER
		// ====================================================

		err = h.service.Register(
			r.Context(), confirmation.Value, r.FormValue("registration_code"),
			user.First_Name,
			user.Last_Name,
			user.Phone,
			user.Email,
			user.Password,

			location,
			user.LGA,
			user.State,
			user.Country,

			role,
			photoURL,

			user.AccountType,
			user.OrganisationName,
			user.FarmingTypes,
		)

		if err != nil {
			log.Println("[registration] submission.failed:", err)

			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		// ====================================================
		// REGISTRATION SUCCESS
		// ====================================================

		log.Printf("[registration] completed: role=%s selfie=%s", role, photoURL)
		registered = true
		clearRegistrationCookie(w, r)

		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}
}
