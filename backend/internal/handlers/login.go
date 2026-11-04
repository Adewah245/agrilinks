package handlers

import (
	"log"
	"net/http"
	"time"

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
	service  *services.AuthService
	sessions *services.SessionService
	security *services.SecurityService
}

// NewLoginHandler creates a new Login handler.
func NewLoginHandler(service *services.AuthService, sessions *services.SessionService, security ...*services.SecurityService) *Login {
	h := &Login{
		service:  service,
		sessions: sessions,
	}
	if len(security) > 0 {
		h.security = security[0]
	}
	return h
}

// LoginHandler renders the login page and processes user login requests.
func (h *Login) LoginHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User visited login page")

		if err := render.RenderPage(
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
		if h.security != nil {
			allowed, err := h.security.Repo.Allow(r.Context(), services.SecurityHash("login:"+services.NormalizeNigerianPhone(phone)), 10, 15*time.Minute)
			if err != nil {
				http.Error(w, "Authentication is unavailable", 503)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", "900")
				http.Error(w, "Too many sign-in attempts. Try again later.", 429)
				return
			}
		}

		user, err := h.service.Login(phone, password)
		if err != nil {
			log.Println("login failed:", err)
			if h.security != nil {
				h.security.Event(r.Context(), 0, "sign_in_failed", "warning")
			}

			if renderErr := render.RenderPage(
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

		sessionID, err := h.sessions.Create(r.Context(), user.ID)
		if err != nil {
			log.Printf("create login session: %v", err)
			http.Error(w, "Login is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if previous, ok := middleware.SessionIDFromRequest(r); ok {
			_ = h.sessions.Delete(r.Context(), previous)
		}
		middleware.SetSessionCookie(w, r, sessionID, h.sessions.TTL())

		destination := "/dashboard"
		if user.AdminRole() != "" {
			destination = "/admin/security"
		}
		http.Redirect(
			w,
			r,
			destination,
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
