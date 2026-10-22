package handlers

import (
	"backend/internal/middleware"
	"backend/internal/services"
	"log"
	"net/http"
)

func NewLogoutHandler(sessions *services.SessionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token, ok := middleware.SessionIDFromRequest(r); ok {
			if err := sessions.Delete(r.Context(), token); err != nil {
				log.Printf("delete session: %v", err)
				http.Error(w, "We could not sign you out. Please try again.", http.StatusServiceUnavailable)
				return
			}
		}
		middleware.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
