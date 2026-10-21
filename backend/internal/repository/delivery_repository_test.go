package repository

import (
	"testing"
	"time"

	"backend/internal/models"
)

func TestNormalizeDeliveryStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", want: "pending"},
		{name: "normalizes", input: " IN_TRANSIT ", want: "in_transit"},
		{name: "rejects unknown", input: "failed", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeDeliveryStatus(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("status = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPrepareDeliveryTimestamps(t *testing.T) {
	delivery := &models.Delivery{Status: "delivered"}

	if err := prepareDeliveryTimestamps(delivery); err != nil {
		t.Fatal(err)
	}
	if delivery.PickedUpAt == nil || delivery.DeliveredAt == nil {
		t.Fatal("delivered delivery must set pickup and delivery timestamps")
	}
	if delivery.DeliveredAt.Before(*delivery.PickedUpAt) {
		t.Fatal("delivery time cannot be before pickup time")
	}

	delivery = &models.Delivery{
		Status:      "delivered",
		PickedUpAt:  timePtr(time.Now()),
		DeliveredAt: timePtr(time.Now().Add(-time.Hour)),
	}
	if err := prepareDeliveryTimestamps(delivery); err == nil {
		t.Fatal("expected invalid timestamp order to be rejected")
	}
}

func TestOrderStatusForDelivery(t *testing.T) {
	tests := map[string]string{
		"assigned":   "",
		"picked_up":  "shipped",
		"in_transit": "shipped",
		"delivered":  "delivered",
		"cancelled":  "",
	}

	for deliveryStatus, want := range tests {
		if got := orderStatusForDelivery(deliveryStatus); got != want {
			t.Fatalf("orderStatusForDelivery(%q) = %q, want %q", deliveryStatus, got, want)
		}
	}
}

func timePtr(value time.Time) *time.Time { return &value }
