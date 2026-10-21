package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/render"
)

// TestMarketplaceTemplateRendersProductModel keeps the marketplace template
// aligned with the product-based model used by the repository and service.
func TestMarketplaceTemplateRendersProductModel(t *testing.T) {
	response := httptest.NewRecorder()

	data := MarketplacePageData{
		IsBuyer:       true,
		CurrentUserID: 1,
		SearchTerm:    "maize",
		Products: []models.Product{
			{
				ID:         2,
				OwnerID:    3,
				Name:       "Maize",
				Category:   "Grains",
				Quantity:   10,
				Unit:       "kg",
				Price:      1250,
				Status:     "active",
				SellerName: "Test Seller",
				Images: []models.ProductImage{
					{
						ImageURL:  "/uploads/products/maize.jpg",
						IsPrimary: true,
					},
				},
			},
		},
	}

	if err := render.RenderTemplates(response, "marketplace.html", data); err != nil {
		t.Fatalf("marketplace template should render: %v", err)
	}

	body := response.Body.String()

	for _, expected := range []string{
		"Maize",
		"₦1,250.00",
		"/uploads/products/maize.jpg",
		"per kg",
		"Amount in kg",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("marketplace template missing %q", expected)
		}
	}

	for _, unwanted := range []string{"Quantity of Maize", "Price per unit"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("marketplace template still uses technical wording %q", unwanted)
		}
	}
}
