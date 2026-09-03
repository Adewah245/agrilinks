package handlers

import (
	"log"
	"net/http"
	"strings"

	"backend/render"
)

type UserReg struct {
	First_Name       string
	Last_Name        string
	Phone            string
	Email            string
	Password         string
	Confirm_Password string

	AccountType string
	NIN         string

	OrganisationName string

	Purposes     []string
	FarmingTypes []string

	Country   string
	State     string
	LGA       string
	Community string
}

func (h *Register) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		log.Println("User Visited Register page")

		if err := render.RenderTemplates(w, "register.html", nil); err != nil {
			log.Println("render error:", err)
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}

	case http.MethodPost:

		// Parse multipart form because the registration
		// includes a profile picture upload.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Invalid registration form", http.StatusBadRequest)
			return
		}

		// Profile picture is required.
		photoURL, err := saveUploadedFile(r, "profile_picture", "farmers")
		if err != nil || photoURL == "" {
			http.Error(w, "Profile picture is required", http.StatusBadRequest)
			return
		}

		user := UserReg{
			First_Name: strings.TrimSpace(r.FormValue("first-name")),
			Last_Name:  strings.TrimSpace(r.FormValue("last-name")),
			Phone:      strings.TrimSpace(r.FormValue("phone")),
			Email:      strings.TrimSpace(r.FormValue("email")),

			Password:         r.FormValue("password"),
			Confirm_Password: r.FormValue("confirm-password"),

			AccountType: strings.TrimSpace(r.FormValue("account_type")),
			NIN:         strings.TrimSpace(r.FormValue("nin")),

			OrganisationName: strings.TrimSpace(
				r.FormValue("organisation_name"),
			),

			Purposes:     r.Form["purpose"],
			FarmingTypes: r.Form["farm_type"],

			Country:   strings.TrimSpace(r.FormValue("country")),
			State:     strings.TrimSpace(r.FormValue("state")),
			LGA:       strings.TrimSpace(r.FormValue("lga")),
			Community: strings.TrimSpace(r.FormValue("community")),
		}

		// --------------------------------------------------
		// BASIC VALIDATION
		// --------------------------------------------------

		if user.First_Name == "" ||
			user.Last_Name == "" ||
			user.Phone == "" ||
			user.Password == "" ||
			user.Confirm_Password == "" {

			log.Println("Required user details are missing")
			http.Error(w, "Required fields must not be empty", http.StatusBadRequest)
			return
		}

		if user.Password != user.Confirm_Password {
			log.Println("Password mismatch")
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		// --------------------------------------------------
		// PURPOSE VALIDATION
		// --------------------------------------------------

		if len(user.Purposes) == 0 {
			http.Error(w, "Please select your purpose", http.StatusBadRequest)
			return
		}

		// --------------------------------------------------
		// LOCATION VALIDATION
		// --------------------------------------------------

		if user.Country == "" ||
			user.State == "" ||
			user.LGA == "" ||
			user.Community == "" {

			http.Error(w, "Complete your location information", http.StatusBadRequest)
			return
		}

		// --------------------------------------------------
		// DETERMINE ROLE
		// --------------------------------------------------

		role := "farmer"

		wantsToSell := false
		wantsToBuy := false

		for _, purpose := range user.Purposes {
			switch purpose {
			case "sell":
				wantsToSell = true

			case "buy":
				wantsToBuy = true
			}
		}

		// Sell only = farmer
		// Buy only  = buyer
		// Sell + Buy = farmer
		if wantsToBuy && !wantsToSell {
			role = "buyer"
		}

		// --------------------------------------------------
		// LOCATION STORAGE
		// --------------------------------------------------
		//
		// For now:
		// Location = Community
		// LGA      = LGA
		// State    = State
		// Country  = Country
		//
		// This matches the existing farmers table without
		// requiring a new community column.

		location := user.Community

		// --------------------------------------------------
		// REGISTER USER
		// --------------------------------------------------

		err = h.service.Register(
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
			user.NIN,
			user.OrganisationName,
			user.FarmingTypes,
		)

		if err != nil {
			log.Println("registration error:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Println("User registered successfully as", role)

		http.Redirect(w, r, "/login", http.StatusSeeOther)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}
