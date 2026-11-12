package render

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed payment_return.html
var paymentReturnSource string

var paymentReturn = template.Must(template.New("payment-return").Parse(paymentReturnSource))

// RenderPaymentReturn reconnects the payment browser with the app without
// sharing a session, reference, or payment status through the custom scheme.
func RenderPaymentReturn(w http.ResponseWriter, screen string) error {
	if screen != "wallet" && screen != "payments" {
		return fmt.Errorf("unknown payment return screen")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	return paymentReturn.Execute(w, struct {
		AppURL template.URL
		WebURL string
	}{
		// Only the two fixed destinations above can reach this trusted URL.
		AppURL: template.URL("agrilinks://" + screen),
		WebURL: "/" + screen,
	})
}
