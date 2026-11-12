package render

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentReturnUsesFixedDestinationsWithoutPaymentClaims(t *testing.T) {
	for _, screen := range []string{"wallet", "payments"} {
		response := httptest.NewRecorder()
		if err := RenderPaymentReturn(response, screen); err != nil {
			t.Fatal(err)
		}
		body := response.Body.String()
		if !strings.Contains(body, `href="agrilinks://`+screen+`"`) || !strings.Contains(body, `href="/`+screen+`"`) {
			t.Fatal("missing app and web return links")
		}
		if strings.Contains(body, "#ZgotmplZ") || strings.Contains(body, "successful") || strings.Contains(body, "tx_ref") {
			t.Fatal("return page must not claim settlement or pass payment details to an app")
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("payment returns must not be cached")
		}
	}
	for _, destination := range []string{"", "https://other.example", "wallet?token=secret", "wallet\" onclick=alert(1)"} {
		response := httptest.NewRecorder()
		if err := RenderPaymentReturn(response, destination); err == nil || response.Body.Len() != 0 {
			t.Fatal("unrecognized app return destination was rendered")
		}
	}
}
