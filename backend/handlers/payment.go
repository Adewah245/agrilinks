package handlers

import (
	"log"
	"net/http"
	"strconv"

	"backend/internal/models"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

type PaymentHandler struct{ service *services.PaymentService }

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type PaymentPageData struct {
	Transactions []models.Transaction
	Wallet       *models.Wallet
	Error        string
	Notice       string
}

func (h *PaymentHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !user.IsBuyer() {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.render(w, user.ID, "", "")
	case http.MethodPost:
		orderID, err := strconv.Atoi(r.FormValue("order_id"))
		if err != nil || orderID <= 0 {
			h.render(w, user.ID, "Enter a valid order ID.", "")
			return
		}
		t, err := h.service.StartBMoniPayment(user.ID, orderID)
		if err != nil {
			h.render(w, user.ID, err.Error(), "")
			return
		}
		log.Printf("created pending B-Moni payment %s for order %d", t.PaymentReference, t.OrderID)
		h.render(w, user.ID, "", "Payment request created. Complete it in B-Moni using reference "+t.PaymentReference+".")
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
func (h *PaymentHandler) render(w http.ResponseWriter, userID int, errorText, notice string) {
	transactions, err := h.service.MyPayments(userID)
	if err != nil {
		errorText = "Could not load payments."
	}
	wallet, err := h.service.MyWallet(userID)
	if err != nil {
		errorText = "Could not load wallet."
	}
	if err := render.RenderTemplates(w, "payments.html", PaymentPageData{Transactions: transactions, Wallet: wallet, Error: errorText, Notice: notice}); err != nil {
		log.Printf("payment page render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
