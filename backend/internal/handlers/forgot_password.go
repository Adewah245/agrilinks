package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"backend/internal/render"
	"backend/internal/services"
)

type passwordRecovery interface {
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string, string) error
}

type ForgotPassword struct{ auth passwordRecovery }

func NewForgotPasswordHandler(auth passwordRecovery) *ForgotPassword {
	return &ForgotPassword{auth: auth}
}

type ForgotPasswordPageData struct {
	Error    string
	Success  string
	CodeSent bool
	Phone    string
}

func (h *ForgotPassword) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		h.render(w, ForgotPasswordPageData{})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	data := ForgotPasswordPageData{Phone: strings.TrimSpace(r.FormValue("phone")), CodeSent: true}
	if data.Phone == "" {
		data.Error = "Enter your phone number."
		h.render(w, data)
		return
	}
	if r.FormValue("action") == "send_code" {
		err := h.auth.RequestPasswordReset(r.Context(), data.Phone)
		if errors.Is(err, services.ErrRecoveryUnavailable) {
			data.Error = "Password recovery is unavailable right now. Please contact support."
		} else {
			if err != nil {
				log.Printf("request password reset: %v", err)
			}
			data.Success = "Request received. If an account uses that number, check your phone or email for a code. If it does not arrive, try again later."
		}
		h.render(w, data)
		return
	}
	password := r.FormValue("new_password")
	code := strings.TrimSpace(r.FormValue("code"))
	switch {
	case password != r.FormValue("confirm_password"):
		data.Error = "The passwords do not match."
	case len(password) < 8 || len(password) > 72:
		data.Error = "Use a password with at least 8 characters and no more than 72 bytes."
	case code == "":
		data.Error = "Enter your verification code."
	default:
		if err := h.auth.ResetPassword(r.Context(), data.Phone, password, code); err != nil {
			log.Printf("reset password: %v", err)
			data.Error = "We could not change your password. Check the code or request a new one."
		} else {
			data = ForgotPasswordPageData{Success: "Your password has been changed. You can sign in now."}
		}
	}
	h.render(w, data)
}

func (h *ForgotPassword) render(w http.ResponseWriter, data ForgotPasswordPageData) {
	if err := render.RenderTemplates(w, "forgot-password.html", data); err != nil {
		log.Printf("render password recovery: %v", err)
	}
}
