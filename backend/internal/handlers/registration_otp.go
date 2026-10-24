package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/services"
)

const registrationCookieName = "agroshield_registration"

func (h *Register) PhoneCodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid phone confirmation request", http.StatusBadRequest)
		return
	}
	token, err := h.service.RequestRegistrationCode(r.Context(), r.PostForm.Get("phone"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrRegistrationOTPUnavailable) {
			status = http.StatusServiceUnavailable
		}
		if errors.Is(err, repository.ErrRegistrationOTPCooldown) {
			status = http.StatusTooManyRequests
			w.Header().Set("Retry-After", strconv.Itoa(durationSeconds(h.service.RegistrationCodeCooldown())))
		}
		http.Error(w, err.Error(), status)
		return
	}
	ttl := h.service.RegistrationCodeTTL()
	http.SetCookie(w, &http.Cookie{
		Name: registrationCookieName, Value: token, Path: "/register",
		HttpOnly: true, Secure: middleware.RequestIsHTTPS(r), SameSite: http.SameSiteStrictMode,
		MaxAge: durationSeconds(ttl), Expires: time.Now().Add(ttl),
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":     "Confirmation code sent. Enter the six-digit code from your SMS.",
		"retry_after": durationSeconds(h.service.RegistrationCodeCooldown()),
	})
}

func durationSeconds(d time.Duration) int { return int((d + time.Second - 1) / time.Second) }

func clearRegistrationCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: registrationCookieName, Path: "/register", MaxAge: -1,
		Expires: time.Unix(0, 0), HttpOnly: true, Secure: middleware.RequestIsHTTPS(r), SameSite: http.SameSiteStrictMode,
	})
}
