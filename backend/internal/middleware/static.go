package middleware

import (
	"net/http"
	"path"
	"strings"
)

// The frontend source tree and quarantined verification uploads are not public
// assets. Inspect the cleaned, decoded path so alternate spellings cannot bypass
// the quarantine. New profile images have their own explicit public directory.
func PublicStaticFiles(directory string) http.Handler {
	files := http.FileServer(http.Dir(directory))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		allowed := false
		for _, prefix := range []string{"assets/", "css/", "dist/", "uploads/products/", "uploads/profiles/"} {
			if strings.HasPrefix(name, prefix) {
				allowed = true
			}
		}
		if !allowed || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
