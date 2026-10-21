package render

import "testing"

func TestHumanStatusLabels(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "order pending", got: humanOrderStatus("pending"), want: "Waiting for payment"},
		{name: "order shipped", got: humanOrderStatus("shipped"), want: "On the way"},
		{name: "payment successful", got: humanPaymentStatus("successful"), want: "Paid"},
		{name: "payment failed", got: humanPaymentStatus("failed"), want: "Payment failed"},
		{name: "negotiation rejected", got: humanNegotiationStatus("rejected"), want: "Declined"},
		{name: "offer pending", got: humanOfferStatus("pending"), want: "Waiting for reply"},
		{name: "delivery assigned", got: humanStatus("assigned"), want: "Driver assigned"},
		{name: "fallback wording", got: humanStatus("ready_for_delivery"), want: "Ready for delivery"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestHumanBadgeLabels(t *testing.T) {
	tests := map[string]string{
		"verified_farmer":       "Verified farmer",
		"verified_organization": "Verified organization",
		"government_partner":    "Government partner",
		"trusted_seller":        "Trusted seller",
	}

	for raw, want := range tests {
		if got := humanBadge(raw); got != want {
			t.Fatalf("humanBadge(%q) = %q, want %q", raw, got, want)
		}
	}
}
