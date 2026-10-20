package handlers

import (
	"net/http"

	"backend/internal/middleware"
)

// LogoutHandler ends the current user session and redirects the user
// to the login page.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if sessionID, ok := middleware.SessionIDFromRequest(r); ok {
		middleware.DeleteSession(sessionID)
	}

	middleware.ClearSessionCookie(w)

	http.Redirect(
		w,
		r,
		"/login",
		http.StatusSeeOther,
	)
}
