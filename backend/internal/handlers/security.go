package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

type SecurityHandler struct {
	security *services.SecurityService
	wallet   *services.WalletService
}

func NewSecurityHandler(s *services.SecurityService, wallet *services.WalletService) *SecurityHandler {
	return &SecurityHandler{s, wallet}
}
func securityResponse(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrSecurityLimit) {
			status = 429
			w.Header().Set("Retry-After", "300")
		}
		if errors.Is(err, services.ErrSecurityUnavailable) {
			status = 503
		}
		w.WriteHeader(status)
		value = map[string]string{"error": err.Error()}
	}
	_ = json.NewEncoder(w).Encode(value)
}

func sensitiveActionError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if errors.Is(err, services.ErrSecurityLimit) {
		status = http.StatusTooManyRequests
		w.Header().Set("Retry-After", "60")
	}
	http.Error(w, err.Error(), status)
}
func (h *SecurityHandler) MFA(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r)
	session, _ := middleware.SessionIDFromRequest(r)
	if u.AdminRole() == "" {
		http.Error(w, "Administrator access required", 403)
		return
	}
	if r.Method == http.MethodGet {
		record, err := h.security.Repo.MFA(r.Context(), u.ID)
		if err != nil {
			http.Error(w, "Could not load security settings", 503)
			return
		}
		fresh, _ := h.security.FreshMFA(r.Context(), u.ID, session)
		destination := "/admin"
		if u.CanAdmin("finance") {
			destination = "/admin/finance"
		}
		_ = render.RenderPage(w, "admin-security.html", map[string]any{"Enabled": record != nil && record.Enabled, "Verified": fresh, "CSRF": adminCSRF(r), "Available": h.security.Available(), "Destination": destination})
		return
	}
	if err := r.ParseForm(); err != nil {
		securityResponse(w, nil, err)
		return
	}
	if r.PostFormValue("csrf") != adminCSRF(r) {
		http.Error(w, "Reload this form before continuing", 403)
		return
	}
	switch r.PostFormValue("action") {
	case "enroll", "replace":
		secret, uri, err := h.security.BeginMFA(r.Context(), u, session, r.PostFormValue("password"), r.PostFormValue("action") == "replace")
		securityResponse(w, map[string]string{"secret": secret, "uri": uri}, err)
	case "verify":
		codes, err := h.security.VerifyMFA(r.Context(), u, session, r.PostFormValue("code"))
		securityResponse(w, map[string]any{"verified": err == nil, "recoveryCodes": codes}, err)
	case "recover":
		err := h.security.RecoverMFA(r.Context(), u, session, r.PostFormValue("password"), r.PostFormValue("code"))
		securityResponse(w, map[string]bool{"verified": err == nil}, err)
	default:
		http.Error(w, "Choose a security action", 400)
	}
}
func (h *SecurityHandler) EvidenceChallenge(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r)
	session, _ := middleware.SessionIDFromRequest(r)
	if err := r.ParseForm(); err != nil {
		securityResponse(w, nil, err)
		return
	}
	token, err := h.security.StartEvidence(r.Context(), u.ID, session, r.PostFormValue("quote"))
	securityResponse(w, map[string]string{"token": token}, err)
}
func (h *SecurityHandler) EvidenceUpload(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r)
	session, _ := middleware.SessionIDFromRequest(r)
	r.Body = http.MaxBytesReader(w, r.Body, (2<<20)+(64<<10))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		securityResponse(w, nil, errors.New("the evidence upload is too large or invalid"))
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("selfie")
	if err != nil {
		securityResponse(w, nil, errors.New("take a selfie to continue"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		securityResponse(w, nil, err)
		return
	}
	e := models.WithdrawalEvidence{}
	for name, dest := range map[string]*float64{"latitude": &e.Latitude, "longitude": &e.Longitude, "accuracy": &e.AccuracyM} {
		v, err := strconv.ParseFloat(r.PostFormValue(name), 64)
		if err != nil {
			securityResponse(w, nil, errors.New("capture your current location"))
			return
		}
		*dest = v
	}
	e.CapturedAt, err = time.Parse(time.RFC3339Nano, r.PostFormValue("captured_at"))
	if err != nil {
		securityResponse(w, nil, errors.New("capture your current location"))
		return
	}
	err = h.security.SaveEvidence(r.Context(), u.ID, session, r.PostFormValue("token"), data, e)
	securityResponse(w, map[string]bool{"saved": err == nil}, err)
}
func (h *SecurityHandler) PrivatePhoto(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r)
	if err := h.security.Limit(r.Context(), u.ID, "private_photo", 60, time.Minute); err != nil {
		sensitiveActionError(w, err)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	owner, encrypted, err := h.security.Repo.Photo(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if owner != u.ID {
		session, _ := middleware.SessionIDFromRequest(r)
		fresh, err := h.security.FreshMFA(r.Context(), u.ID, session)
		if !u.CanAdmin("finance") || err != nil || !fresh {
			http.NotFound(w, r)
			return
		}
	}
	data, err := h.security.Open(encrypted, "verification-photo")
	if err != nil {
		http.Error(w, "Private evidence is unavailable", 503)
		return
	}
	session, _ := middleware.SessionIDFromRequest(r)
	if err = h.security.Repo.MarkPhotoViewed(r.Context(), u.ID, services.SecurityHash(session), id); err != nil {
		http.Error(w, "Could not record evidence access", 503)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "inline; filename=verification.jpg")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	_, _ = w.Write(data)
}
func (h *SecurityHandler) WithdrawalReview(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.UserFromContext(r)
	session, _ := middleware.SessionIDFromRequest(r)
	if !u.CanAdmin("finance") {
		http.Error(w, "Finance access required", 403)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	withdrawal, err := h.wallet.WithdrawalForReview(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	evidence, err := h.security.Repo.Evidence(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load evidence", 503)
		return
	}
	photoID, err := h.security.Repo.RegistrationPhoto(r.Context(), withdrawal.UserID)
	if err != nil {
		http.Error(w, "Could not load verification photo", 503)
		return
	}
	page := models.WithdrawalReviewPage{Withdrawal: withdrawal, Evidence: evidence, RegistrationPhotoID: photoID, CSRF: adminCSRF(r)}
	if evidence != nil && withdrawal.UserID != u.ID {
		if page.RegistrationPhotoID, err = h.security.Repo.ReviewEvidence(r.Context(), id, u.ID, services.SecurityHash(session)); err != nil {
			http.Error(w, "Verify your administrator authenticator again", 403)
			return
		}
	} else if withdrawal.UserID == u.ID {
		page.Error = "Another finance administrator must review your withdrawal."
	} else {
		page.Error = "This request has no private evidence. Reject it and ask the member to submit a new request."
	}
	_ = render.RenderPage(w, "withdrawal-review.html", page)
}

func RequireAdminMFA(s *services.SecurityService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, _ := middleware.UserFromContext(r)
		if u.AdminRole() == "" {
			http.Error(w, "Administrator access required", 403)
			return
		}
		session, _ := middleware.SessionIDFromRequest(r)
		fresh, err := s.FreshMFA(r.Context(), u.ID, session)
		if err != nil && !errors.Is(err, services.ErrSecurityUnavailable) {
			http.Error(w, "Authentication is unavailable", 503)
			return
		}
		if !fresh {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/admin/security", 303)
			} else {
				http.Error(w, repository.ErrMFARequired.Error(), 403)
			}
			return
		}
		next(w, r)
	}
}

// Shared member pages can expose administrative powers. Staff sessions must
// prove MFA there too; members retain the handlers' normal ownership checks.
func RequireAdminMFAForStaff(s *services.SecurityService, next http.HandlerFunc) http.HandlerFunc {
	protected := RequireAdminMFA(s, next)
	return func(w http.ResponseWriter, r *http.Request) {
		u, _ := middleware.UserFromContext(r)
		if u.AdminRole() != "" {
			protected(w, r)
			return
		}
		next(w, r)
	}
}

func LimitFinancialActions(s *services.SecurityService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			u, _ := middleware.UserFromContext(r)
			if err := s.Limit(r.Context(), u.ID, "finance_action", 20, time.Minute); err != nil {
				sensitiveActionError(w, err)
				return
			}
		}
		next(w, r)
	}
}
