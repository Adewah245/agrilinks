package services

import (
	"testing"

	"backend/internal/models"
)

func TestIsNegotiationParticipant(t *testing.T) {
	service := &NegotiationService{}

	tests := []struct {
		name        string
		negotiation *models.Negotiation
		userID      int
		want        bool
	}{
		{
			name: "buyer",
			negotiation: &models.Negotiation{
				BuyerID:   2,
				OwnerType: "user",
				OwnerID:   9,
			},
			userID: 2,
			want:   true,
		},
		{
			name: "individual product owner",
			negotiation: &models.Negotiation{
				BuyerID:   2,
				OwnerType: "user",
				OwnerID:   9,
			},
			userID: 9,
			want:   true,
		},
		{
			name: "unrelated user",
			negotiation: &models.Negotiation{
				BuyerID:   2,
				OwnerType: "user",
				OwnerID:   9,
			},
			userID: 3,
			want:   false,
		},
		{
			name: "buyer of organization-owned product",
			negotiation: &models.Negotiation{
				BuyerID:   2,
				OwnerType: "organization",
				OwnerID:   2,
			},
			userID: 2,
			want:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.isNegotiationParticipant(
				test.negotiation,
				test.userID,
			)
			if err != nil {
				t.Fatalf("isNegotiationParticipant() error = %v", err)
			}

			if got != test.want {
				t.Fatalf("isNegotiationParticipant() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRequireParticipantRejectsUnrelatedUser(t *testing.T) {
	service := &NegotiationService{}
	negotiation := &models.Negotiation{
		BuyerID:   2,
		OwnerType: "user",
		OwnerID:   9,
	}

	if err := service.requireParticipant(negotiation, 3); err != ErrNegotiationAccessDenied {
		t.Fatalf("requireParticipant() error = %v, want access denied", err)
	}
}

func TestOtherParticipantID(t *testing.T) {
	service := &NegotiationService{}
	negotiation := &models.Negotiation{
		BuyerID:   2,
		OwnerType: "user",
		OwnerID:   9,
	}

	if got := service.otherParticipantID(negotiation, 2); got != 9 {
		t.Fatalf("buyer recipient = %d, want 9", got)
	}

	if got := service.otherParticipantID(negotiation, 9); got != 2 {
		t.Fatalf("seller recipient = %d, want 2", got)
	}

	negotiation.OwnerType = "organization"
	if got := service.otherParticipantID(negotiation, 2); got != 0 {
		t.Fatalf("organization recipient = %d, want 0", got)
	}
}
