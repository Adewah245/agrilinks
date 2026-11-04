package middleware

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicStaticFilesQuarantinesLegacyVerificationAndSource(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"uploads/users/private.jpg", "uploads/profiles/public.jpg", "assets/logo.svg", "src/main.jsx", ".env"} {
		file := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	handler := PublicStaticFiles(dir)
	for _, path := range []string{"/uploads/users/private.jpg", "/uploads/profiles/../users/private.jpg", "/uploads/%75sers/private.jpg", "/src/main.jsx", "/.env", "/assets/"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"/assets/logo.svg", "/uploads/profiles/public.jpg"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Errorf("public asset %s: %d", path, w.Code)
		}
	}
}
