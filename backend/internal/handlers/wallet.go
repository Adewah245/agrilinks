package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"backend/internal/middleware"
	"backend/internal/render"
	"backend/internal/services"
)

type WalletHandler struct{ service *services.WalletService }

func NewWalletHandler(service *services.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

func (h *WalletHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var err error
	if r.Method == http.MethodPost {
		if err = r.ParseForm(); err != nil {
			http.Error(w, "Could not read the form", 400)
			return
		}
		notice := "saved"
		switch r.FormValue("action") {
		case "deposit":
			var result *services.InitiatePaymentResult
			result, err = h.service.StartDeposit(r.Context(), user, r.FormValue("amount"), r.FormValue("request_key"))
			if err == nil {
				http.Redirect(w, r, result.CheckoutURL, http.StatusSeeOther)
				return
			}
		case "pay_order":
			id, _ := strconv.Atoi(r.FormValue("order_id"))
			err = h.service.PayOrder(r.Context(), user.ID, id)
			if err == nil {
				http.Redirect(w, r, "/orders", http.StatusSeeOther)
				return
			}
		case "send_code":
			err = h.service.SendCode(r.Context(), user, r.FormValue("purpose"))
			notice = "code_sent"
		case "transfer":
			err = h.service.RequestTransfer(r.Context(), user, r.FormValue("recipient"), r.FormValue("amount"), r.FormValue("note"), r.FormValue("code"))
		case "withdraw":
			err = h.service.RequestWithdrawal(r.Context(), user, r.FormValue("amount"), r.FormValue("code"))
		case "save_bank":
			err = h.service.SaveBank(r.Context(), user.ID, r.FormValue("bank_code"), r.FormValue("account_number"))
		case "cancel_transfer":
			id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
			err = h.service.DecideTransfer(r.Context(), user, id, "cancelled")
		case "cancel_withdrawal":
			id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
			err = h.service.DecideWithdrawal(r.Context(), user, id, "cancelled")
		default:
			http.Error(w, "Choose a wallet action", 400)
			return
		}
		if err == nil {
			http.Redirect(w, r, "/wallet?notice="+notice, http.StatusSeeOther)
			return
		}
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	page, pageErr := h.service.Page(r.Context(), user)
	if pageErr != nil {
		log.Printf("wallet page: %v", pageErr)
		http.Error(w, "Could not load your wallet", 500)
		return
	}
	if err != nil {
		log.Printf("wallet action: %v", err)
		page.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
	}
	switch r.URL.Query().Get("notice") {
	case "saved":
		page.Notice = "Your request was saved. Check its status below."
	case "code_sent":
		page.Notice = "Your code was sent. Enter it below to continue."
	}
	if err = render.RenderTemplates(w, "wallet.html", page); err != nil {
		log.Printf("wallet template: %v", err)
	}
}

func (h *WalletHandler) Banks(w http.ResponseWriter, r *http.Request) {
	banks, err := h.service.Banks(r.Context())
	if err != nil {
		http.Error(w, "Could not load the bank list. Please try again.", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(banks)
}

func (h *WalletHandler) Admin(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || !user.IsAdministrator() {
		http.Error(w, "Administrator access is required", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var err error
	if r.Method == http.MethodPost {
		id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
		switch r.FormValue("kind") {
		case "transfer":
			err = h.service.DecideTransfer(r.Context(), user, id, r.FormValue("decision"))
		case "withdrawal":
			err = h.service.DecideWithdrawal(r.Context(), user, id, r.FormValue("decision"))
		default:
			http.Error(w, "Choose a request", 400)
			return
		}
		if err == nil {
			http.Redirect(w, r, "/wallet/admin", http.StatusSeeOther)
			return
		}
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	page, pageErr := h.service.AdminPage(r.Context(), user)
	if pageErr != nil {
		http.Error(w, "Could not load wallet requests", 500)
		return
	}
	if err != nil {
		page.Error = err.Error()
		w.WriteHeader(400)
	}
	if err = render.RenderTemplates(w, "wallet_admin.html", page); err != nil {
		log.Printf("wallet admin template: %v", err)
	}
}
