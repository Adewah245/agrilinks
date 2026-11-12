package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"backend/internal/middleware"
	"backend/internal/services"
)

func bankAccountErrorMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrBankAccountUnconfirmed):
		return "We couldn't confirm this account. Check the bank and account number, then try again."
	case errors.Is(err, services.ErrBankVerificationUnavailable):
		return "Bank verification is temporarily unavailable. Please try again shortly."
	default:
		return err.Error()
	}
}

func (h *WalletHandler) ResolveBankAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	reject := func(status int, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		reject(http.StatusUnauthorized, "Please sign in to verify your bank account.")
		return
	}
	if h.security != nil {
		if err := h.security.Limit(r.Context(), user.ID, "bank-account-lookup", 10, time.Minute); err != nil {
			if errors.Is(err, services.ErrSecurityLimit) {
				w.Header().Set("Retry-After", "60")
				reject(http.StatusTooManyRequests, "Please wait a minute before checking another bank account.")
			} else {
				reject(http.StatusServiceUnavailable, "Bank verification is temporarily unavailable. Please try again shortly.")
			}
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		reject(http.StatusBadRequest, "Could not read the bank details.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	account, err := h.service.ResolveBankAccount(ctx, user, r.PostFormValue("bank_code"), r.PostFormValue("account_number"))
	switch {
	case errors.Is(err, services.ErrBankAccountNumber), errors.Is(err, services.ErrBankSelection):
		reject(http.StatusBadRequest, bankAccountErrorMessage(err))
	case errors.Is(err, services.ErrBankAccountUnconfirmed):
		reject(http.StatusUnprocessableEntity, bankAccountErrorMessage(err))
	case err != nil:
		reject(http.StatusServiceUnavailable, "Bank verification is temporarily unavailable. Please try again shortly.")
	default:
		_ = json.NewEncoder(w).Encode(account)
	}
}

func (h *WalletHandler) ResolveBankAccountByNumber(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	reject := func(status int, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
	user, ok := middleware.UserFromContext(r)
	if !ok || user == nil {
		reject(http.StatusUnauthorized, "Please sign in to verify your bank account.")
		return
	}
	if h.security != nil {
		if err := h.security.Limit(r.Context(), user.ID, "bank-account-lookup", 10, time.Minute); err != nil {
			if errors.Is(err, services.ErrSecurityLimit) {
				w.Header().Set("Retry-After", "60")
				reject(http.StatusTooManyRequests, "Please wait a minute before checking another bank account.")
			} else {
				reject(http.StatusServiceUnavailable, "Bank verification is temporarily unavailable. Please try again shortly.")
			}
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		reject(http.StatusBadRequest, "Could not read the account number.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	account, err := h.service.ResolveBankAccountByNumber(ctx, user, r.PostFormValue("account_number"))
	switch {
	case errors.Is(err, services.ErrBankAccountNumber):
		reject(http.StatusBadRequest, bankAccountErrorMessage(err))
	case errors.Is(err, services.ErrBankAccountUnconfirmed):
		reject(http.StatusUnprocessableEntity, "We couldn't find this account number at any bank. Please check the number and try again.")
	case err != nil:
		reject(http.StatusServiceUnavailable, "Bank verification is temporarily unavailable. Please try again shortly.")
	default:
		_ = json.NewEncoder(w).Encode(account)
	}
}
