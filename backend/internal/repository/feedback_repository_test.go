package repository

import (
	"testing"

	"backend/internal/models"
)

func TestValidateFeedback(t *testing.T) {
	valid := &models.Feedback{
		UserID:    1,
		FarmerID:  2,
		ProductID: 3,
		Rating:    5,
		Comment:   "  Good produce  ",
	}

	if err := validateFeedback(valid); err != nil {
		t.Fatalf("valid feedback rejected: %v", err)
	}
	if valid.Comment != "Good produce" {
		t.Fatalf("comment = %q, want trimmed value", valid.Comment)
	}

	tests := []struct {
		name   string
		mutate func(*models.Feedback)
	}{
		{name: "missing user", mutate: func(f *models.Feedback) { f.UserID = 0 }},
		{name: "missing seller", mutate: func(f *models.Feedback) { f.FarmerID = 0 }},
		{name: "missing product", mutate: func(f *models.Feedback) { f.ProductID = 0 }},
		{name: "negative order", mutate: func(f *models.Feedback) { f.OrderID = -1 }},
		{name: "rating too low", mutate: func(f *models.Feedback) { f.Rating = 0 }},
		{name: "rating too high", mutate: func(f *models.Feedback) { f.Rating = 6 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			feedback := *valid
			test.mutate(&feedback)
			if err := validateFeedback(&feedback); err == nil {
				t.Fatal("expected invalid feedback to be rejected")
			}
		})
	}

	if err := validateFeedback(nil); err == nil {
		t.Fatal("expected nil feedback to be rejected")
	}
}
