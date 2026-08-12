package middleware

import (
	"log"
	"net/http"
)

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
