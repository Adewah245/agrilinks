package render

import (
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"backend/internal/models"
)

func TestReactBootstrapEscapesAndProjectsPrivateData(t *testing.T) {
	response := httptest.NewRecorder()
	user := &models.User{ID: 7, FullName: `Ada </script><script>alert(1)</script>`, PasswordHash: "secret-password-hash", NIN: "private-identity-number", AccountNumber: "private-bank-number", StaffRole: "super_admin"}
	err := RenderPage(response, "admin.html", models.AdminPage{User: user, CSRF: "form-token", Section: models.AdminSections[0]})
	if err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	for _, forbidden := range []string{"secret-password-hash", "private-identity-number", "private-bank-number", "<script>alert(1)</script>"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("bootstrap exposes %q", forbidden)
		}
	}
	match := regexp.MustCompile(`<script id="page-data" type="application/json">([\s\S]*?)</script>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatal("missing JSON bootstrap")
	}
	var bootstrap struct {
		Page string
		Data struct {
			User struct {
				FullName       string
				CanManageUsers bool
				CanAdmin       map[string]bool
			}
			CSRF string
		}
	}
	if err := json.Unmarshal([]byte(match[1]), &bootstrap); err != nil {
		t.Fatal(err)
	}
	if bootstrap.Page != "admin.html" || bootstrap.Data.User.FullName != user.FullName || !bootstrap.Data.User.CanManageUsers || !bootstrap.Data.User.CanAdmin["revenue"] || bootstrap.Data.CSRF != "form-token" {
		t.Fatalf("page data or server permissions were lost: %+v", bootstrap)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("personal page data must not be cached")
	}
}

func TestReactUnknownPageReturnsErrorBeforeWriting(t *testing.T) {
	response := httptest.NewRecorder()
	if RenderPage(response, "missing.html", nil) == nil || response.Code != 500 {
		t.Fatal("unknown pages must fail with HTTP 500")
	}
	if strings.Contains(response.Body.String(), "<html") {
		t.Fatal("failed render wrote a partial page")
	}
}
