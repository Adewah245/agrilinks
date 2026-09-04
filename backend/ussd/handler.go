package ussd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
)

type userFinder interface {
	GetByPhone(string) (*models.User, error)
}

type productLister interface {
	AvailableProducts() ([]models.Product, error)
	MyProducts(string, int) ([]models.Product, error)
}

type weatherGetter interface {
	Current(context.Context, string) (services.Weather, error)
}

// Handler implements a provider-neutral USSD callback.
//
// It accepts the common phoneNumber/text fields used by
// Africa's Talking and similar gateways.
type Handler struct {
	users         userFinder
	products      productLister
	weather       weatherGetter
	orders        *services.OrderService
	notifications *services.NotificationService
}

func NewHandler(
	users *repository.UserRepository,
	products *services.ProductService,
	weather *services.WeatherService,
	orders *services.OrderService,
	notifications *services.NotificationService,
) *Handler {
	return &Handler{
		users:         users,
		products:      products,
		weather:       weather,
		orders:        orders,
		notifications: notifications,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = r.ParseForm()

	phone := strings.TrimSpace(
		first(
			r.FormValue("phoneNumber"),
			r.FormValue("phone"),
			r.FormValue("msisdn"),
		),
	)

	text := strings.TrimSpace(r.FormValue("text"))

	response := h.respond(r.Context(), phone, text)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(response))
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}

	return ""
}

func (h *Handler) respond(
	ctx context.Context,
	phone string,
	text string,
) string {
	// USSD is an authenticated access channel:
	// registration and profile capture happen online,
	// and the gateway phone number identifies the account.

	user, err := h.users.GetByPhone(phone)
	if err != nil {
		return "END Agro-Shield is temporarily unavailable. Please try again later."
	}

	if user == nil {
		return "END This number is not registered. Register online at Agro-Shield, then dial again."
	}

	parts := []string{}

	if text != "" {
		parts = strings.Split(text, "*")
	}

	if len(parts) == 0 || parts[0] == "" {
		return "CON Agro-Shield\n1. Storage\n2. Marketplace\n3. Sales\n4. Notifications\n5. Profile"
	}

	switch parts[0] {

	case "1":
		// Storage
		items, err := h.products.MyProducts("user", user.ID)
		if err != nil {
			return "END Storage is temporarily unavailable."
		}

		if len(items) == 0 {
			return "END Storage is empty."
		}

		lines := []string{"END Storage:"}

		for _, product := range items {
			lines = append(
				lines,
				fmt.Sprintf(
					"%s: %.0f %s",
					product.Name,
					product.Quantity,
					product.Unit,
				),
			)
		}

		return strings.Join(lines[:min(len(lines), 5)], "\n")

	case "2":
		// Marketplace
		items, err := h.products.AvailableProducts()
		if err != nil || len(items) == 0 {
			return "END Marketplace is empty."
		}

		lines := []string{"END Marketplace:"}

		for _, product := range items {
			lines = append(
				lines,
				fmt.Sprintf(
					"%d. %s ₦%s/%s",
					product.ID,
					product.Name,
					formatMoney(product.Price),
					product.Unit,
				),
			)
		}

		return strings.Join(lines[:min(len(lines), 5)], "\n")

	case "3":
		// Sales
		items, err := h.orders.MySales("user", user.ID)
		if err != nil || len(items) == 0 {
			return "END No sales recorded."
		}

		lines := []string{"END Sales:"}

		for _, order := range items {
			lines = append(
				lines,
				fmt.Sprintf(
					"%s: ₦%s (%s)",
					order.ProductName,
					formatMoney(order.TotalPrice),
					order.Status,
				),
			)
		}

		return strings.Join(lines[:min(len(lines), 5)], "\n")

	case "4":
		// Notifications
		count, err := h.notifications.GetUnreadCount(user.ID)
		if err != nil {
			return "END Notifications unavailable."
		}

		return fmt.Sprintf(
			"END You have %d unread notification(s).",
			count,
		)

	case "5":
		// Profile
		return fmt.Sprintf(
			"END %s\n%s\n%s",
			user.FullName,
			user.Phone,
			first(user.Location, "Location not set"),
		)

	default:
		return "CON Invalid option\n1. Storage\n2. Marketplace\n3. Sales\n4. Notifications\n5. Profile"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func formatMoney(value float64) string {
	s := strconv.FormatFloat(value, 'f', 2, 64)

	parts := strings.SplitN(s, ".", 2)

	whole := parts[0]

	sign := ""

	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}

	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}

	return sign + whole + "." + parts[1]
}

// ParseInt is kept small and exported for integrations that need
// to parse a menu selection consistently.
func ParseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}
