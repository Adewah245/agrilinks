package handlers

import (
	"log"
	"net/http"

	"backend/internal/middleware"
	"backend/internal/render"
	"backend/internal/services"
)

// LoginPageData contains the data passed to login.html.
type LoginPageData struct {
	Error string
}

// Login handles user authentication and login-page rendering.
type Login struct {
	service *services.AuthService
}

// NewLoginHandler creates a new Login handler.
func NewLoginHandler(service *services.AuthService) *Login {
	return &Login{
		service: service,
	}
}

// LoginHandler renders the login page and processes user login requests.
func (h *Login) LoginHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User visited login page")

		if err := render.RenderTemplates(
			w,
			"login.html",
			LoginPageData{},
		); err != nil {
			log.Println("login page render error:", err)
			return
		}

	case http.MethodPost:
		phone := r.FormValue("phone")
		password := r.FormValue("password")

		user, err := h.service.Login(phone, password)
		if err != nil {
			log.Println("login failed:", err)

			if renderErr := render.RenderTemplates(
				w,
				"login.html",
				LoginPageData{
					Error: "Invalid phone number or password",
				},
			); renderErr != nil {
				log.Println("login error page render error:", renderErr)
			}

			return
		}

		sessionID := middleware.CreateSession(user.ID)
		middleware.SetSessionCookie(w, sessionID)

		http.Redirect(
			w,
			r,
			"/dashboard",
			http.StatusSeeOther,
		)

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}
