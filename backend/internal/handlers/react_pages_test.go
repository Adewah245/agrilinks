package handlers

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
	"backend/internal/testui"
)

// Exercise every route's real DTO with populated nested fields. This catches
// misspelled projections, missing presentation methods and invalid React markup.
func TestReactPages(t *testing.T) {
	pages := map[string]any{
		"admin-security.html": struct {
			Enabled, Verified, Available bool
			CSRF, Destination            string
		}{Available: true},
		"withdrawal-review.html": models.WithdrawalReviewPage{},
		"index.html":             nil,
		"login.html":             LoginPageData{},
		"register.html":          struct{ FarmingTypes []models.FarmingType }{},
		"forgot-password.html":   ForgotPasswordPageData{},
		"dashboard.html":         DashboardData{},
		"marketplace.html":       MarketplacePageData{},
		"product.html":           ProductPageData{},
		"cart.html":              CartPageData{},
		"orders.html":            OrderPageData{},
		"payments.html":          PaymentPageData{},
		"sales.html":             SalesPageData{},
		"storage.html":           StoragePageData{},
		"profile.html":           ProfilePageData{},
		"chat.html":              ChatListPageData{User: &models.User{}},
		"chat_thread.html":       ChatThreadPageData{User: &models.User{}},
		"negotiations.html":      NegotiationListPageData{},
		"negotiation.html":       NegotiationThreadPageData{},
		"notification.html": struct {
			User          *models.User
			Notifications []models.Notification
			UnreadCount   int
		}{},
		"deliveries.html":           DeliveryPageData{},
		"delivery_partner.html":     DeliveryPageData{},
		"ai-assistant.html":         AIAssistantPageData{},
		"ai-diagnosis-history.html": AIDiagnosisHistoryPageData{},
		"learning.html":             PublicLearningData{},
		"learn.html":                LearningHomeData{},
		"learning_journey.html":     LearningJourneyData{},
		"learning_rooms.html": struct {
			Rooms        []models.LearningRoom
			FarmingTypes []models.FarmingType
			SelectedType int
		}{},
		"trust.html":               TrustPageData{},
		"wallet.html":              services.WalletPage{},
		"wallet_admin.html":        services.WalletAdminPage{},
		"admin.html":               models.AdminPage{},
		"platform_revenue.html":    models.FinancePage{},
		"purchase_protection.html": models.EscrowPage{},
	}
	// Market insights is constructed as an anonymous handler view model.
	pages["market-insights.html"] = struct {
		IsBuyer   bool
		FullName  string
		Summaries []models.ProduceDemandSummary
	}{}
	for name, data := range pages {
		t.Run(name, func(t *testing.T) {
			if data != nil {
				value := reflect.New(reflect.TypeOf(data)).Elem()
				value.Set(reflect.ValueOf(data))
				populatePage(value, "", 0)
				data = value.Interface()
			}
			response := httptest.NewRecorder()
			if err := render.RenderPage(response, name, data); err != nil {
				t.Fatal(err)
			}
			body := testui.HTML(t, response.Body.String())
			if strings.Contains(body, "[object Object]") || strings.Contains(body, "NaN") || strings.Contains(body, "<no value>") {
				t.Fatal("invalid page data rendered")
			}
			if len(body) < 100 {
				t.Fatal("empty React page")
			}
		})
	}
}

func populatePage(value reflect.Value, name string, depth int) {
	if !value.CanSet() || depth > 7 {
		return
	}
	if value.Type() == reflect.TypeOf(time.Time{}) {
		value.Set(reflect.ValueOf(time.Date(2026, 11, 4, 12, 30, 0, 0, time.UTC)))
		return
	}
	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		populatePage(value.Elem(), name, depth+1)
	case reflect.Interface:
		if name == "Sales" {
			value.Set(reflect.ValueOf([]models.Order{{ID: 1, ProductName: "Maize", Quantity: 1, TotalAmount: 100}}))
		}
		if name == "User" {
			value.Set(reflect.ValueOf(&models.User{ID: 1, FullName: "Ada Farmer", Role: "farmer"}))
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			populatePage(value.Field(i), value.Type().Field(i).Name, depth+1)
		}
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		populatePage(value.Index(0), name, depth+1)
	case reflect.Map:
		if value.Type().Key().Kind() == reflect.String {
			value.Set(reflect.MakeMap(value.Type()))
			item := reflect.New(value.Type().Elem()).Elem()
			populatePage(item, name, depth+1)
			value.SetMapIndex(reflect.ValueOf("reference"), item)
		}
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(100)
	case reflect.String:
		text := "Example"
		switch name {
		case "Error", "Notice", "Success", "Query", "Search", "SearchTerm":
			text = ""
		case "PhotoURL", "ImageURL", "SellerPhotoURL", "OtherPhotoURL", "BuyerPhotoURL":
			text = "/static/assets/images/un.jpeg"
		case "FullName":
			text = "Ada Farmer"
		case "Key":
			text = "overview"
		case "StaffRole":
			text = "super_admin"
		case "Role":
			text = "farmer"
		case "Status":
			text = "active"
		case "Name", "ProductName":
			text = "Maize"
		case "Reference":
			text = "reference"
		case "Kind":
			text = "withdrawal"
		case "State":
			text = "Benue"
		case "ConversationType":
			text = "private"
		case "Currency":
			text = "NGN"
		case "Balance", "Amount", "ServiceFee", "ProviderFee", "Total", "WalletCredit", "WalletDebit", "BankCredit", "CheckoutTotal":
			text = "100.00"
		case "Href", "Path":
			text = "/dashboard"
		}
		value.SetString(text)
	}
}
