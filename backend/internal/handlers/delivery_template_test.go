package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/render"
)

func TestDeliveryWorkspaceRendersBuyerSellerAndPartnerViews(t *testing.T) {
	response := httptest.NewRecorder()

	data := DeliveryPageData{
		User: &models.User{
			ID:       7,
			FullName: "Ada Buyer",
			Phone:    "08012345678",
			Role:     "buyer",
		},
		EligibleOrders: []models.DeliveryOrderOption{
			{ID: 11, ProductNames: "Maize", SellerName: "Ada Farms", Status: "confirmed"},
		},
		Deliveries: []models.Delivery{
			{ID: 4, OrderID: 11, PickupAddress: "Kano market", DestinationAddress: "12 Market Road", Status: "pending", TrackingCode: "AGD-ABC123"},
		},
		SellerDeliveries: []models.Delivery{
			{ID: 5, OrderID: 12, BuyerName: "Musa", PickupAddress: "Farm", DestinationAddress: "Town", Status: "assigned", PartnerName: "Swift Logistics"},
		},
		PartnerDeliveries: []models.Delivery{
			{ID: 6, OrderID: 13, BuyerName: "Ngozi", PickupAddress: "Farm", DestinationAddress: "Shop", Status: "in_transit", TrackingCode: "AGD-DEF456"},
		},
		AvailablePartners: []models.DeliveryPartner{{ID: 9, Name: "Swift Logistics", VehicleType: "Truck"}},
		Tracking: []models.DeliveryTracking{
			{ID: 1, DeliveryID: 6, LocationName: "Kano central market", Note: "On the way", CreatedAt: time.Now()},
		},
		SelectedID: 6,
	}

	if err := render.RenderTemplates(response, "deliveries.html", data); err != nil {
		t.Fatalf("deliveries template should render: %v", err)
	}

	body := response.Body.String()
	for _, expected := range []string{
		"Arrange delivery",
		"Request delivery",
		"Deliveries for your sales",
		"Deliveries assigned to you",
		"On the way",
		"Kano central market",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("deliveries template missing %q", expected)
		}
	}
}

func TestDeliveryPartnerTemplateRendersRegistrationAndReview(t *testing.T) {
	response := httptest.NewRecorder()
	data := DeliveryPageData{
		User: &models.User{ID: 1, FullName: "Admin", Phone: "08012345678", IsAdmin: true},
		PartnerAccounts: []models.DeliveryPartner{
			{ID: 9, Name: "Swift Logistics", Phone: "08087654321", Status: "active", MemberRole: "owner"},
		},
		PartnersForReview: []models.DeliveryPartner{
			{ID: 10, Name: "New Movers", Phone: "08011112222", Status: "active"},
		},
	}

	if err := render.RenderTemplates(response, "delivery_partner.html", data); err != nil {
		t.Fatalf("delivery partner template should render: %v", err)
	}

	body := response.Body.String()
	for _, expected := range []string{
		"Register a delivery business",
		"Local Government Area",
		"Swift Logistics",
		"New Movers",
		"Waiting for approval",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("delivery partner template missing %q", expected)
		}
	}
}

func TestOptionalPositiveID(t *testing.T) {
	value, err := optionalPositiveID(" 12 ")
	if err != nil {
		t.Fatal(err)
	}
	if value == nil || *value != 12 {
		t.Fatalf("value = %v, want 12", value)
	}

	for _, raw := range []string{"", "abc", "0", "-3"} {
		value, err := optionalPositiveID(raw)
		if raw == "" {
			if err != nil || value != nil {
				t.Fatalf("empty value = %v, error = %v", value, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("optionalPositiveID(%q) should fail", raw)
		}
	}
}
