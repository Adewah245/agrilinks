package services

import (
	"testing"

	"backend/internal/models"
)

func TestBuildOrderFromCartUsesStoredAgreedPrices(t *testing.T) {
	order, cartIDs, err := buildOrderFromCart(7, []models.CartItem{
		{
			ID:           11,
			ProductID:    21,
			Quantity:     2,
			PricePerUnit: 1500,
			SellerType:   "user",
			SellerID:     9,
			ProductName:  "Maize",
			SellerName:   "Ada Farms",
		},
		{
			ID:           12,
			ProductID:    22,
			Quantity:     1,
			PricePerUnit: 500,
			SellerType:   "user",
			SellerID:     9,
			ProductName:  "Beans",
			SellerName:   "Ada Farms",
		},
	})
	if err != nil {
		t.Fatalf("buildOrderFromCart() error = %v", err)
	}

	if order.TotalAmount != 3500 {
		t.Fatalf("total = %v, want 3500", order.TotalAmount)
	}

	if len(order.Items) != 2 || len(cartIDs) != 2 {
		t.Fatalf("items = %d, cart IDs = %d", len(order.Items), len(cartIDs))
	}

	if order.Items[0].UnitPrice != 1500 || order.Items[1].UnitPrice != 500 {
		t.Fatalf("agreed prices were not preserved: %#v", order.Items)
	}
}

func TestBuildOrderFromCartRejectsMixedSellers(t *testing.T) {
	_, _, err := buildOrderFromCart(7, []models.CartItem{
		{ID: 1, ProductID: 1, Quantity: 1, SellerType: "user", SellerID: 9},
		{ID: 2, ProductID: 2, Quantity: 1, SellerType: "user", SellerID: 10},
	})
	if err == nil {
		t.Fatal("expected mixed sellers to be rejected")
	}
}

func TestBuildOrderFromCartRejectsSelfPurchase(t *testing.T) {
	_, _, err := buildOrderFromCart(7, []models.CartItem{
		{ID: 1, ProductID: 1, Quantity: 1, SellerType: "user", SellerID: 7},
	})
	if err == nil {
		t.Fatal("expected self-purchase to be rejected")
	}
}
