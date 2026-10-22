package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

const maxPaymentWebhookBytes = 1 << 20

type PaymentHandler struct{ service *services.PaymentService }

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type PaymentPageData struct {
	User     *models.User
	Payments []models.Payment
	Orders   []models.Order
	Currency string
	Error    string
	Notice   string
}

// Handler renders payment history and starts hosted checkout for a pending
// order owned by the authenticated buyer.
func (h *PaymentHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.render(w, user, "", "")
	case http.MethodPost:
		if !user.CanBuy() {
			h.render(w, user, "Only buyers can start purchase payments.", "")
			return
		}

		orderID, err := strconv.Atoi(r.FormValue("order_id"))
		if err != nil || orderID <= 0 {
			h.render(w, user, "Select a valid order.", "")
			return
		}

		result, err := h.service.Initiate(r.Context(), user.ID, orderID)
		if err != nil {
			log.Printf("start payment for order %d: %v", orderID, err)
			h.render(w, user, paymentErrorMessage(err), "")
			return
		}

		http.Redirect(w, r, result.CheckoutURL, http.StatusSeeOther)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// CallbackHandler verifies Flutterwave's browser redirect before displaying
// the updated payment state. Query-string status values are never trusted.
func (h *PaymentHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	transactionID := r.URL.Query().Get("transaction_id")
	reference := r.URL.Query().Get("tx_ref")
	if transactionID == "" || reference == "" {
		http.Error(w, "Invalid payment callback", http.StatusBadRequest)
		return
	}

	status, err := h.service.ConfirmCallback(r.Context(), transactionID, reference)
	if err != nil {
		log.Printf("payment callback verification failed: %v", err)
		if errors.Is(err, services.ErrReferenceMismatch) {
			http.Error(w, "Payment verification failed", http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not verify payment", http.StatusBadGateway)
		return
	}

	log.Printf("verified payment callback for reference %s with status %s", reference, status)
	http.Redirect(w, r, "/payments", http.StatusSeeOther)
}

// WebhookHandler receives the provider's authoritative payment event.
func (h *PaymentHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPaymentWebhookBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	err = h.service.ProcessWebhook(
		r.Context(),
		body,
		r.Header.Get("verif-hash"),
	)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, services.ErrUnauthorizedWebhook):
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	case errors.Is(err, services.ErrReferenceMismatch):
		http.Error(w, "Payment verification failed", http.StatusBadRequest)
	case errors.Is(err, services.ErrInvalidWebhook):
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
	default:
		log.Printf("payment webhook processing failed: %v", err)
		http.Error(w, "Webhook processing failed", http.StatusInternalServerError)
	}
}

func (h *PaymentHandler) render(
	w http.ResponseWriter,
	user *models.User,
	errorText,
	notice string,
) {
	w.Header().Set("Cache-Control", "no-store")

	payments, err := h.service.MyPayments(user.ID)
	if err != nil && errorText == "" {
		errorText = "Could not load payments."
	}

	orders, err := h.service.PayableOrders(user.ID)
	if err != nil && errorText == "" {
		errorText = "Could not load orders awaiting payment."
	}

	data := PaymentPageData{
		User:     user,
		Payments: payments,
		Orders:   orders,
		Currency: h.service.Currency(),
		Error:    errorText,
		Notice:   notice,
	}
	if err := render.RenderTemplates(w, "payments.html", data); err != nil {
		log.Printf("payment page render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func paymentErrorMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrOrderNotFound):
		return "Order not found."
	case errors.Is(err, services.ErrOrderNotAwaitingPayment):
		return "This order is not awaiting payment."
	case errors.Is(err, services.ErrPaymentEmailRequired):
		return "Add an email address to your profile before paying."
	case errors.Is(err, services.ErrPaymentProvider):
		return "The payment provider is unavailable. Please try again."
	default:
		return "Could not start payment."
	}
}
