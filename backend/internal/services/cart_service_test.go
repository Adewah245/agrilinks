package services

import (
	"testing"

	"backend/internal/models"
)

func TestAddFromNegotiationRequiresAcceptedStatus(t *testing.T) {
	service := &CartService{}
	negotiation := &models.Negotiation{
		ID:        1,
		BuyerID:   2,
		ProductID: 3,
		Quantity:  1,
		Status:    "open",
	}

	if err := service.AddFromNegotiation(negotiation, 100); err == nil {
		t.Fatal("expected an open negotiation to be rejected")
	}
}

func TestAddFromNegotiationRejectsNilNegotiation(t *testing.T) {
	service := &CartService{}

	if err := service.AddFromNegotiation(nil, 100); err == nil {
		t.Fatal("expected a nil negotiation to be rejected")
	}
}
