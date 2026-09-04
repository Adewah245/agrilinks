package handlers

import (
	"log"
	"net/http"

	"backend/render"
)

// IndexHandler renders the Agro-Shield home page.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		log.Println("User visited home page")

		if err := render.RenderTemplates(
			w,
			"index.html",
			nil,
		); err != nil {
			log.Println("home page render error:", err)
			return
		}

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
	}
}
