package render

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
)

func TestNativePageDataPreservesProjectionAndPermissions(t *testing.T) {
	user := &models.User{ID: 7, FullName: "Ada Okafor", PasswordHash: "private-password", NIN: "private-nin", VerificationPhoto: []byte("private-selfie"), AccountNumber: "private-bank", StaffRole: "super_admin"}
	handler := PageDataRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := RenderPage(w, "admin.html", models.AdminPage{User: user, CSRF: "form-token", Section: models.AdminSections[0]}); err != nil {
			t.Error(err)
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Accept", PageDataContentType)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), PageDataContentType) {
		t.Fatalf("unexpected native response: %d %s", response.Code, response.Header())
	}
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Vary") != "Accept" {
		t.Fatal("page snapshots must not be cached or mixed with HTML")
	}
	for _, private := range []string{"private-password", "private-nin", "private-selfie", "private-bank"} {
		if strings.Contains(response.Body.String(), private) {
			t.Fatalf("native page exposed %q", private)
		}
	}
	var page struct {
		Page, Title, BodyClass string
		Styles                 []string
		Data                   struct {
			CSRF string
			User struct {
				FullName       string
				CanManageUsers bool
				CanAdmin       map[string]bool
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Page != "admin.html" || page.Title == "" || len(page.Styles) == 0 || page.Data.CSRF != "form-token" || page.Data.User.FullName != user.FullName || !page.Data.User.CanManageUsers || !page.Data.User.CanAdmin["revenue"] {
		t.Fatalf("native page lost metadata, form tokens, or server permissions: %+v", page)
	}
}

func TestPageDataNegotiationPreservesWebAndExistingEndpoints(t *testing.T) {
	for _, accept := range []string{"", "text/html", "application/json", PageDataContentType + ";q=0", PageDataContentType + ";q=invalid", PageDataContentType + ";q=NaN"} {
		t.Run(accept, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Accept", accept)
			PageDataRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := RenderPage(w, "index.html", nil); err != nil {
					t.Fatal(err)
				}
			})).ServeHTTP(response, request)
			if !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") || !strings.Contains(response.Body.String(), `id="page-data"`) {
				t.Fatal("web clients must keep receiving the HTML shell")
			}
		})
	}
	for _, redirect := range []bool{false, true} {
		request := httptest.NewRequest(http.MethodPost, "/login", nil)
		request.Header.Set("Accept", PageDataContentType)
		response := httptest.NewRecorder()
		PageDataRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redirect {
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"invalid input"}`))
		})).ServeHTTP(response, request)
		if redirect {
			if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard" {
				t.Fatal("native negotiation changed an authentication redirect")
			}
		} else if response.Code != http.StatusUnprocessableEntity || response.Header().Get("Content-Type") != "application/json" || response.Body.String() != `{"error":"invalid input"}` {
			t.Fatal("native negotiation changed an existing JSON endpoint")
		}
	}
}
