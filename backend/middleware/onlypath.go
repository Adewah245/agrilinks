package middleware

import (
	"log"
	"net/http"
)

// OnlyPath allows a request to continue only when its URL path
// exactly matches the expected path.
//
// Requests to any other path receive a 404 Not Found response.
func OnlyPath(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			log.Printf("%s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}

		next(w, r)
	}
}
