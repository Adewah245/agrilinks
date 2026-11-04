package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

type WalletHandler struct {
	service  *services.WalletService
	security *services.SecurityService
}

func NewWalletHandler(service *services.WalletService, security ...*services.SecurityService) *WalletHandler {
	h := &WalletHandler{service: service}
	if len(security) > 0 {
		h.security = security[0]
	}
	return h
}

func (h *WalletHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var err error
	fundingAction := false
	if r.Method == http.MethodPost {
		if h.security != nil {
			if err = h.security.Limit(r.Context(), user.ID, "wallet", 30, time.Minute); err != nil {
				sensitiveActionError(w, err)
				return
			}
		}
		if err = r.ParseForm(); err != nil {
			http.Error(w, "Could not read the form", 400)
			return
		}
		notice := "saved"
		switch r.PostFormValue("action") {
		case "setup_funding", "check_funding":
			fundingAction = true
			var account *models.WalletVirtualAccount
			if r.PostFormValue("action") == "setup_funding" {
				account, err = h.service.SetupVirtualAccount(r.Context(), user, services.VirtualAccountSetup{IdentityKind: r.PostFormValue("identity_kind"), IdentityNumber: r.PostFormValue("identity_number"), Consent: r.PostFormValue("consent") == "yes"})
			} else {
				account, err = h.service.RefreshVirtualAccount(r.Context(), user.ID, user.FullName)
			}
			// Identity input is never echoed into the error page or retained in a URL.
			r.PostForm.Del("identity_number")
			r.Form.Del("identity_number")
			notice = "funding_pending"
			if account != nil && account.Ready() {
				notice = "funding_ready"
			}
			if account != nil && account.Status == models.VirtualAccountFailed && err == nil {
				err = services.ErrVirtualAccountRejected
			}
		case "setup_pin":
			err = h.service.SetupPIN(r.Context(), user, r.PostFormValue("current_password"), r.PostFormValue("transaction_pin"), r.PostFormValue("transaction_pin_confirmation"))
			notice = "pin_created"
		case "quote_deposit", "quote_withdrawal", "quote_funding":
			kind := "deposit"
			if r.PostFormValue("action") == "quote_withdrawal" {
				kind = "withdrawal"
			}
			if r.PostFormValue("action") == "quote_funding" {
				kind = "funding_preview"
			}
			var quote *models.FeeQuote
			quote, err = h.service.Quote(r.Context(), user, kind, r.PostFormValue("amount"), r.PostFormValue("payment_method"))
			if err == nil {
				http.Redirect(w, r, "/wallet?quote="+quote.ID+"&kind="+kind+"#wallet-quote", http.StatusSeeOther)
				return
			}
		case "quote_allocation":
			var q *models.FeeQuote
			q, err = h.service.QuoteUnallocatedFunding(r.Context(), user, r.PostFormValue("reference"))
			if err == nil {
				http.Redirect(w, r, "/wallet?quote="+q.ID+"&kind=funding_allocation#wallet-quote", 303)
				return
			}
		case "allocate_funding":
			err = h.service.AllocateFunding(r.Context(), user.ID, r.PostFormValue("request_key"))
		case "accept_funding":
			err = h.service.AcceptFundingIntent(r.Context(), user.ID, r.PostFormValue("request_key"))
			notice = "funding_instruction"
		case "deposit":
			var result *services.InitiatePaymentResult
			result, err = h.service.StartDeposit(r.Context(), user, r.FormValue("amount"), r.FormValue("request_key"))
			if err == nil {
				http.Redirect(w, r, result.CheckoutURL, http.StatusSeeOther)
				return
			}
		case "pay_order":
			id, _ := strconv.Atoi(r.FormValue("order_id"))
			err = h.service.PayOrder(r.Context(), user.ID, id, r.PostFormValue("transaction_pin"))
			if err == nil {
				http.Redirect(w, r, "/orders", http.StatusSeeOther)
				return
			}
		case "send_code":
			err = h.service.SendCode(r.Context(), user, r.FormValue("purpose"))
			notice = "code_sent"
		case "transfer":
			err = h.service.RequestTransfer(r.Context(), user, r.FormValue("recipient"), r.FormValue("amount"), r.FormValue("note"), r.FormValue("code"), r.PostFormValue("transaction_pin"), r.PostFormValue("request_key"))
		case "withdraw":
			session, _ := middleware.SessionIDFromRequest(r)
			err = h.service.RequestWithdrawal(r.Context(), user, r.FormValue("amount"), r.FormValue("code"), r.PostFormValue("transaction_pin"), r.PostFormValue("request_key"), r.PostFormValue("evidence_token"), session)
			notice = "withdrawal_review"
		case "save_bank":
			err = h.service.SaveBank(r.Context(), user.ID, r.FormValue("bank_code"), r.FormValue("account_number"), r.PostFormValue("transaction_pin"))
			if err == nil && h.security != nil {
				h.security.Event(r.Context(), user.ID, "payout_bank_changed", "warning")
			}
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
			destination := "/wallet?notice=" + notice
			if r.PostFormValue("action") == "send_code" && r.PostFormValue("quote") != "" {
				destination += "&kind=withdrawal&quote=" + url.QueryEscape(r.PostFormValue("quote")) + "#wallet-quote"
			}
			http.Redirect(w, r, destination, http.StatusSeeOther)
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
	if key := r.URL.Query().Get("quote"); key != "" {
		page.Quote, pageErr = h.service.ReviewQuote(r.Context(), user.ID, key, r.URL.Query().Get("kind"))
		if pageErr != nil {
			page.Error = pageErr.Error()
		}
	}
	if err != nil {
		if h.security != nil {
			h.security.Event(r.Context(), user.ID, "wallet_action_failed", "warning")
		}
		if action := r.PostFormValue("action"); action == "withdraw" || action == "deposit" {
			kind := "deposit"
			if action == "withdraw" {
				kind = "withdrawal"
			}
			page.Quote, _ = h.service.ReviewQuote(r.Context(), user.ID, r.PostFormValue("request_key"), kind)
		}
		log.Printf("wallet action: %v", err)
		page.Error = err.Error()
		page.FailedAction = r.PostFormValue("action")
		page.RequestKey = r.PostFormValue("request_key")
		if fundingAction {
			page.Error = fundingErrorMessage(err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
	}
	switch r.URL.Query().Get("notice") {
	case "withdrawal_review":
		page.Notice = "Your withdrawal and private evidence are awaiting administrator review. No payout has been submitted."
	case "funding_instruction":
		page.Notice = "Funding instruction confirmed. Transfer the exact total shown below to receive your full requested wallet credit."
	case "pin_created":
		page.Notice = "Your transaction PIN is ready. Keep it private; Agrilinks Nigeria staff will never ask for it."
	case "saved":
		page.Notice = "Your request was saved. Check its status below."
	case "code_sent":
		page.Notice = "Your code was sent. Enter it below to continue."
	case "funding_ready":
		page.Notice = "Your funding account is ready. Use the bank details below to add money."
	case "funding_pending":
		page.Notice = "Your funding account is awaiting confirmation. You can check its status below."
	}
	if err = render.RenderPage(w, "wallet.html", page); err != nil {
		log.Printf("wallet template: %v", err)
	}
}

func fundingErrorMessage(err error) string {
	for _, known := range []error{services.ErrVirtualAccountConsent, services.ErrVirtualAccountIdentity, services.ErrVirtualAccountProfile, services.ErrVirtualAccountRejected, services.ErrVirtualAccountUnavailable} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "We could not confirm your funding account. Check its status before trying again."
}

func (h *WalletHandler) Status(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	revision, err := h.service.Revision(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Could not check your wallet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"revision": revision})
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
	if err = render.RenderPage(w, "wallet_admin.html", page); err != nil {
		log.Printf("wallet admin template: %v", err)
	}
}
