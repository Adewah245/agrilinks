package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/render"
)

func TestAdminTemplateRendersEveryStaffWorkspace(t *testing.T) {
	for _, role := range []string{"super_admin", "operations_manager", "customer_care"} {
		user := &models.User{ID: 1, FullName: "Ada <script>alert(1)</script>", StaffRole: role}
		var navigation []models.AdminSection
		for _, section := range models.AdminSections {
			if user.CanAdmin(section.Key) {
				navigation = append(navigation, section)
			}
		}
		for _, section := range navigation {
			t.Run(role+"/"+section.Key, func(t *testing.T) {
				page := models.AdminPage{
					User: user, Section: section, Navigation: navigation, CSRF: "test-session-token", Page: 2, PreviousPage: 1, NextPage: 3, HasNext: true,
					Tables:   []models.AdminTable{{Key: "users", Title: "Member accounts", Columns: []string{"ID", "Name", "Status"}, Rows: []models.AdminRow{{ID: "17", Status: "active", Cells: []string{"17", "Test Farmer", "active"}}}}},
					Settings: []models.AdminSetting{{Key: "support_default_priority", Label: "Default priority", Value: "normal"}},
					Policies: []models.AdminSetting{{Label: "Response target", Value: "48 hours"}},
				}
				if section.Key == "overview" || section.Key == "reports" {
					page.Metrics = []models.AdminMetric{
						{Label: "Total users", Value: "100", Path: "/admin/users"},
						{Label: "Active farmers", Value: "70", Path: "/admin/users"},
						{Label: "Buyers", Value: "30", Path: "/admin/users"},
						{Label: "Orders", Value: "14", Path: "/admin/orders"},
						{Label: "Pending tasks", Value: "2", Path: "/admin/support"},
					}
					if user.CanAdmin("finance") {
						page.Metrics = append(page.Metrics, models.AdminMetric{Label: "Revenue", Value: "NGN 100.00", Detail: "Recorded platform commissions", Path: "/admin/finance"})
					}
				}
				response := httptest.NewRecorder()
				if err := render.RenderTemplates(response, "admin.html", page); err != nil {
					t.Fatalf("admin workspace should render: %v", err)
				}
				body := response.Body.String()
				if !strings.Contains(body, "&lt;script&gt;") || strings.Contains(body, "<script>alert(1)</script>") {
					t.Error("staff names must remain escaped")
				}
				if role == "customer_care" && (strings.Contains(body, `href="/admin/finance"`) || strings.Contains(body, `href="/admin/staff"`) || strings.Contains(body, "Recorded platform commissions") || strings.Contains(body, `name="action" value="user_status"`)) {
					t.Error("customer care must not receive finance, staff, or account-management controls")
				}
				if !strings.Contains(body, `class="pagination"`) || !strings.Contains(body, `name="q"`) {
					t.Error("server search and pagination must remain available")
				}
			})
		}
	}
}

func TestAdminTemplatePreservesReviewFormsAndTokens(t *testing.T) {
	page := models.AdminPage{User: &models.User{ID: 1, StaffRole: "super_admin", FullName: "Ada Farmer"}, Section: models.AdminSections[5], CSRF: "review-session-token", Page: 1}
	for _, key := range []string{"users", "products", "moderation", "tickets", "staff", "withdrawals", "transfers"} {
		page.Tables = append(page.Tables, models.AdminTable{Key: key, Title: key, Columns: []string{"ID", "Status"}, Rows: []models.AdminRow{{ID: "17", Status: "pending", Cells: []string{"17", "pending"}}}})
	}
	response := httptest.NewRecorder()
	if err := render.RenderTemplates(response, "admin.html", page); err != nil {
		t.Fatalf("admin review forms should render: %v", err)
	}
	body := response.Body.String()
	for _, action := range []string{"user_status", "user_verification", "product_status", "report_update", "ticket_create", "ticket_update", "staff_role", "staff_access", "wallet_withdrawal", "wallet_transfer"} {
		if !strings.Contains(body, `name="action" value="`+action+`"`) {
			t.Errorf("missing review action %s", action)
		}
	}
	forms := strings.Count(body, `action="/admin/action"`)
	if forms == 0 || strings.Count(body, `name="csrf" value="review-session-token"`) != forms || strings.Count(body, `name="return_to" value="/admin/support"`) != forms {
		t.Error("every administrative action must retain its session token and return path")
	}
}
