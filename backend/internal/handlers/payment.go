package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

type PaymentHandler struct{ service *services.PaymentService }

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type PaymentPageData struct {
	Payments []models.Payment
	Wallet   *models.Wallet
	Profile  *models.User
	Error    string
	Notice   string
}

func (h *PaymentHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.render(w, user.ID, "", "")
	case http.MethodPost:
		if r.FormValue("action") == "save_bank_account" {
			err := h.service.UpdateBankAccount(
				user.ID,
				r.FormValue("bank_name"),
				r.FormValue("bank_code"),
				r.FormValue("account_name"),
				r.FormValue("account_number"),
			)
			if err != nil {
				h.render(w, user.ID, err.Error(), "")
				return
			}

			h.render(w, user.ID, "", "Bank account saved.")
			return
		}

		if !user.IsBuyer() {
			h.render(w, user.ID, "Only buyers can start purchase payments.", "")
			return
		}

		orderID, err := strconv.Atoi(r.FormValue("order_id"))
		if err != nil || orderID <= 0 {
			h.render(w, user.ID, "Enter a valid order ID.", "")
			return
		}
		payment, err := h.service.StartFlutterwavePayment(user.ID, orderID)
		if err != nil {
			h.render(w, user.ID, err.Error(), "")
			return
		}
		log.Printf("created pending Flutterwave payment %s for order %d", payment.Reference, payment.OrderID)
		h.render(w, user.ID, "", "Payment request created. Complete it with Flutterwave using reference "+payment.Reference+".")
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
func (h *PaymentHandler) render(w http.ResponseWriter, userID int, errorText, notice string) {
	payments, err := h.service.MyPayments(userID)
	if err != nil && errorText == "" {
		errorText = "Could not load payments."
	}
	wallet, err := h.service.MyWallet(userID)
	if err != nil && errorText == "" {
		errorText = "Could not load wallet."
	}
	profile, err := h.service.PaymentProfile(userID)
	if err != nil && errorText == "" {
		errorText = "Could not load payment profile."
	}
	if err := render.RenderTemplates(w, "payments.html", PaymentPageData{Payments: payments, Wallet: wallet, Profile: profile, Error: errorText, Notice: notice}); err != nil {
		log.Printf("payment page render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
