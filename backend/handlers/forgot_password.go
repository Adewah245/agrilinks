package handlers

import (
	"log"
	"net/http"

	"backend/internal/services"
	"backend/render"
)

// ForgotPassword handles password-reset requests from users.
type ForgotPassword struct {
	auth *services.AuthService
}

// NewForgotPasswordHandler creates a new ForgotPassword handler.
func NewForgotPasswordHandler(auth *services.AuthService) *ForgotPassword {
	return &ForgotPassword{
		auth: auth,
	}
}

// ForgotPasswordPageData contains the data displayed on the
// forgot-password page.
type ForgotPasswordPageData struct {
	Error   string
	Success string
}

// Handler renders the password-reset page and processes password-reset
// submissions.
func (h *ForgotPassword) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := render.RenderTemplates(
			w,
			"forgot-password.html",
			ForgotPasswordPageData{},
		); err != nil {
			log.Println("forgot password render error:", err)
		}
		return

	case http.MethodPost:
		phone := r.FormValue("phone")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		if newPassword != confirmPassword {
			h.render(w, "Passwords do not match", "")
			return
		}

		if err := h.auth.ResetPassword(phone, newPassword); err != nil {
			h.render(w, err.Error(), "")
			return
		}

		h.render(
			w,
			"",
			"Password updated — you can now log in.",
		)
		return

	default:
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}
}

// render renders the forgot-password page with either an error
// message or a success message.
func (h *ForgotPassword) render(
	w http.ResponseWriter,
	errMsg string,
	success string,
) {
	data := ForgotPasswordPageData{
		Error:   errMsg,
		Success: success,
	}

	if err := render.RenderTemplates(
		w,
		"forgot-password.html",
		data,
	); err != nil {
		log.Println("forgot password render error:", err)
	}
}
