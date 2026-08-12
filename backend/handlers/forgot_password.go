package handlers

import (
	"log"
	"net/http"

	"backend/internal/services"
	"backend/render"
)

type ForgotPassword struct {
	auth *services.AuthService
}

func NewForgotPasswordHandler(auth *services.AuthService) *ForgotPassword {
	return &ForgotPassword{auth: auth}
}

type ForgotPasswordPageData struct {
	Error   string
	Success string
}

func (h *ForgotPassword) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := render.RenderTemplates(w, "forgot-password.html", ForgotPasswordPageData{}); err != nil {
			log.Println("render error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

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

		h.render(w, "", "Password updated — you can now log in.")

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ForgotPassword) render(w http.ResponseWriter, errMsg, success string) {
	if err := render.RenderTemplates(w, "forgot-password.html", ForgotPasswordPageData{Error: errMsg, Success: success}); err != nil {
		log.Println("render error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
