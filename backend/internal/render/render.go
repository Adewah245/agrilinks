package render

import (
	"bytes"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func commaNumber(value float64, decimals int) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	s := strconv.FormatFloat(value, 'f', decimals, 64)
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	sign := ""
	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if decimals == 0 {
		return sign + whole
	}
	return sign + whole + "." + parts[1]
}

func money(value float64) string  { return commaNumber(value, 2) }
func money0(value float64) string { return commaNumber(value, 0) }
func hasCoordinates(latitude, longitude float64) bool {
	return latitude != 0 && longitude != 0
}

func readableStatus(status string) string {
	status = strings.TrimSpace(strings.ReplaceAll(status, "_", " "))
	if status == "" {
		return ""
	}
	runes := []rune(strings.ToLower(status))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func humanOrderStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "Waiting for payment"
	case "confirmed":
		return "Order confirmed"
	case "processing":
		return "Being prepared"
	case "ready_for_delivery":
		return "Ready for delivery"
	case "shipped":
		return "On the way"
	case "delivered":
		return "Delivered"
	case "cancelled", "canceled":
		return "Cancelled"
	default:
		return readableStatus(status)
	}
}

func humanPaymentStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "Waiting"
	case "successful":
		return "Paid"
	case "failed":
		return "Payment failed"
	case "cancelled", "canceled":
		return "Cancelled"
	case "refunded":
		return "Refunded"
	default:
		return readableStatus(status)
	}
}

func humanNegotiationStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open":
		return "Open"
	case "accepted":
		return "Accepted"
	case "rejected":
		return "Declined"
	case "expired":
		return "Expired"
	case "closed":
		return "Closed"
	default:
		return readableStatus(status)
	}
}

func humanOfferStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "Waiting for reply"
	case "accepted":
		return "Accepted"
	case "rejected":
		return "Declined"
	default:
		return readableStatus(status)
	}
}

func humanStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "Waiting"
	case "assigned":
		return "Driver assigned"
	case "picked_up":
		return "Picked up"
	case "in_transit":
		return "On the way"
	case "completed":
		return "Completed"
	case "failed":
		return "Not successful"
	case "active":
		return "Active"
	case "paused":
		return "Paused"
	case "in_progress":
		return "In progress"
	default:
		return readableStatus(status)
	}
}

func loadTemplates() *template.Template {
	funcs := template.FuncMap{
		"money":                  money,
		"money0":                 money0,
		"comma":                  func(v int) string { return commaNumber(float64(v), 0) },
		"hasCoordinates":         hasCoordinates,
		"humanOrderStatus":       humanOrderStatus,
		"humanPaymentStatus":     humanPaymentStatus,
		"humanNegotiationStatus": humanNegotiationStatus,
		"humanOfferStatus":       humanOfferStatus,
		"humanStatus":            humanStatus,
	}
	for _, pattern := range []string{"../frontend/pages/*.html", "../../frontend/pages/*.html", "../../../frontend/pages/*.html", "frontend/pages/*.html"} {
		if matches, err := os.ReadDir("."); err == nil && matches != nil {
			if templates, parseErr := template.New("base").Funcs(funcs).ParseGlob(pattern); parseErr == nil {
				return templates
			}
		}
	}
	return template.Must(template.New("base").Funcs(funcs).ParseGlob("../../../frontend/pages/*.html"))
}

var TMPL = loadTemplates()

func RenderTemplates(w http.ResponseWriter, name string, data any) error {
	// Render before writing to the HTTP response. A template error after a
	// partial write prevents callers from returning a correct error response
	// and produces a "superfluous response.WriteHeader" server warning.
	var output bytes.Buffer
	if err := TMPL.ExecuteTemplate(&output, name, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(output.Bytes())
	return err
}
