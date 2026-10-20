package middleware

import (
	"log"
	"net/http"
)

// OnlyGet allows a request to continue only when its HTTP method is GET.
//
// Requests using any other HTTP method receive a 405 Method Not Allowed
// response.
func OnlyGet(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			log.Println("Method Not Allowed")
			http.Error(
				w,
				"Method Not Allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		next(w, r)
	}
}
