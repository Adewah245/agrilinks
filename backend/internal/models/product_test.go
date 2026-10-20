package models

import "testing"

func TestProductPrimaryImageURL(t *testing.T) {
	product := Product{
		Images: []ProductImage{
			{ImageURL: "/images/first.jpg"},
			{ImageURL: "/images/primary.jpg", IsPrimary: true},
		},
	}

	if got := product.PrimaryImageURL(); got != "/images/primary.jpg" {
		t.Fatalf("expected primary image, got %q", got)
	}

	product.Images = product.Images[:1]
	if got := product.PrimaryImageURL(); got != "/images/first.jpg" {
		t.Fatalf("expected fallback image, got %q", got)
	}

	product.Images = nil
	if got := product.PrimaryImageURL(); got != "" {
		t.Fatalf("expected empty image URL, got %q", got)
	}
}
