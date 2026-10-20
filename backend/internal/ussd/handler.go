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

// userFinder defines the user lookup operation required by the USSD handler.
type userFinder interface {
	GetByPhone(string) (*models.User, error)
}

// productLister defines the product operations required by the USSD handler.
type productLister interface {
	AvailableProducts() ([]models.Product, error)
	MyProducts(string, int) ([]models.Product, error)
}

// weatherGetter defines the weather operation required by the USSD handler.
//
// Weather remains available as a dependency because Agro-Shield's USSD
// experience can later provide weather-aware storage and farming alerts.
type weatherGetter interface {
	Current(context.Context, string) (models.Weather, error)
}

// Handler implements a provider-neutral USSD callback.
//
// It accepts the common phoneNumber/text fields used by
// Africa's Talking and similar USSD gateways.
//
// The user's registered phone number is used to identify the
// Agro-Shield account.
type Handler struct {
	users         userFinder
	products      productLister
	weather       weatherGetter
	orders        *services.OrderService
	notifications *services.NotificationService
}

// NewHandler creates a new USSD handler with its required dependencies.
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

// ServeHTTP handles an incoming USSD callback request.
//
// Both GET and POST requests are accepted because different USSD
// gateway providers may use different HTTP methods.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
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

	text := strings.TrimSpace(
		r.FormValue("text"),
	)

	response := h.respond(
		r.Context(),
		phone,
		text,
	)

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	_, _ = w.Write([]byte(response))
}

// first returns the first non-empty string from the supplied values.
//
// It is used for USSD gateway compatibility because providers may
// use different parameter names for the same value.
func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

// respond processes a USSD request and returns the appropriate
// CON or END response.
//
// USSD is treated as an authenticated access channel. Registration
// and profile management happen online, while the registered phone
// number identifies the user's Agro-Shield account.
func (h *Handler) respond(
	ctx context.Context,
	phone string,
	text string,
) string {
	if strings.TrimSpace(phone) == "" {
		return "END Unable to identify your Agro-Shield account."
	}

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
		return mainMenu()
	}

	switch parts[0] {

	case "1":
		// ----------------------------------------------------
		// Storage
		// ----------------------------------------------------

		items, err := h.products.MyProducts(
			"user",
			user.ID,
		)

		if err != nil {
			return "END Storage is temporarily unavailable."
		}

		if len(items) == 0 {
			return "END Storage is empty."
		}

		lines := []string{
			"END Storage:",
		}

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

		return strings.Join(
			lines[:min(len(lines), 5)],
			"\n",
		)

	case "2":
		// ----------------------------------------------------
		// Marketplace
		// ----------------------------------------------------

		items, err := h.products.AvailableProducts()

		if err != nil || len(items) == 0 {
			return "END Marketplace is empty."
		}

		lines := []string{
			"END Marketplace:",
		}

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

		return strings.Join(
			lines[:min(len(lines), 5)],
			"\n",
		)

	case "3":
		// ----------------------------------------------------
		// Sales
		// ----------------------------------------------------

		items, err := h.orders.MySales(
			"user",
			user.ID,
		)

		if err != nil || len(items) == 0 {
			return "END No sales recorded."
		}

		lines := []string{
			"END Sales:",
		}

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

		return strings.Join(
			lines[:min(len(lines), 5)],
			"\n",
		)

	case "4":
		// ----------------------------------------------------
		// Notifications
		// ----------------------------------------------------

		count, err := h.notifications.GetUnreadCount(
			user.ID,
		)

		if err != nil {
			return "END Notifications unavailable."
		}

		return fmt.Sprintf(
			"END You have %d unread notification(s).",
			count,
		)

	case "5":
		// ----------------------------------------------------
		// Profile
		// ----------------------------------------------------

		return fmt.Sprintf(
			"END %s\n%s\n%s",
			user.FullName,
			user.Phone,
			first(
				user.Location,
				"Location not set",
			),
		)

	default:
		return mainMenuInvalid()
	}
}

// mainMenu returns the initial Agro-Shield USSD menu.
func mainMenu() string {
	return "CON Agro-Shield\n" +
		"1. Storage\n" +
		"2. Marketplace\n" +
		"3. Sales\n" +
		"4. Notifications\n" +
		"5. Profile"
}

// mainMenuInvalid returns the USSD menu after an invalid selection.
func mainMenuInvalid() string {
	return "CON Invalid option\n" +
		"1. Storage\n" +
		"2. Marketplace\n" +
		"3. Sales\n" +
		"4. Notifications\n" +
		"5. Profile"
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// formatMoney formats a monetary value using two decimal places
// and comma-separated thousands.
func formatMoney(value float64) string {
	s := strconv.FormatFloat(
		value,
		'f',
		2,
		64,
	)

	parts := strings.SplitN(
		s,
		".",
		2,
	)

	whole := parts[0]

	sign := ""

	if strings.HasPrefix(whole, "-") {
		sign = "-"
		whole = whole[1:]
	}

	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] +
			"," +
			whole[i:]
	}

	return sign +
		whole +
		"." +
		parts[1]
}

// ParseInt parses a string into an integer after trimming
// surrounding whitespace.
//
// It is exported for integrations that need to parse USSD
// menu selections consistently.
func ParseInt(value string) (int, error) {
	return strconv.Atoi(
		strings.TrimSpace(value),
	)
}
