package handlers

import (
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type AdminWorkspaceHandler struct {
	repo   *repository.AdminRepository
	wallet *services.WalletService
}

func NewAdminWorkspaceHandler(repo *repository.AdminRepository, wallet *services.WalletService) *AdminWorkspaceHandler {
	return &AdminWorkspaceHandler{repo: repo, wallet: wallet}
}
func adminCSRF(r *http.Request) string {
	token, _ := middleware.SessionIDFromRequest(r)
	hash := sha256.Sum256([]byte("agro-shield:admin-actions:" + token))
	return hex.EncodeToString(hash[:])
}

func (h *AdminWorkspaceHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		http.Error(w, "Please sign in", 401)
		return
	}
	var section models.AdminSection
	for _, item := range models.AdminSections {
		if item.Path == r.URL.Path {
			section = item
			break
		}
	}
	if section.Key == "" {
		http.NotFound(w, r)
		return
	}
	if !user.CanAdmin(section.Key) {
		h.auditFailure(r, user, "section_access", section.Path, "Permission denied")
		http.Error(w, "Your staff role does not have access to this section", 403)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		http.Error(w, "Page is out of range", 400)
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(search) > 200 {
		http.Error(w, "Search is too long", 400)
		return
	}
	data, err := h.repo.Page(r.Context(), user, section, search, page)
	if err != nil {
		log.Printf("admin page %s: %v", section.Key, err)
		http.Error(w, "Could not load the admin workspace", 500)
		return
	}
	data.CSRF = adminCSRF(r)
	if r.URL.Query().Get("notice") == "saved" {
		data.Notice = "The change was saved and recorded in the audit log."
	}
	if err := render.RenderTemplates(w, "admin.html", data); err != nil {
		log.Printf("admin template: %v", err)
	}
}

func (h *AdminWorkspaceHandler) Action(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		http.Error(w, "Please sign in", 401)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not read the form", 400)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.PostFormValue("csrf")), []byte(adminCSRF(r))) != 1 {
		h.auditFailure(r, user, "request_rejected", "admin", "Invalid request token")
		http.Error(w, "This form has expired. Reload the page and try again.", 403)
		return
	}
	f := map[string]string{}
	for key := range r.PostForm {
		f[key] = r.PostFormValue(key)
	}
	action := f["action"]
	section := repository.AdminActionSection(action)
	if action == "wallet_withdrawal" || action == "wallet_transfer" {
		section = "finance"
	}
	if !user.CanAdmin(section) {
		h.auditFailure(r, user, action, f["id"], "Permission denied")
		http.Error(w, "Your staff role cannot perform this action", 403)
		return
	}
	var err error
	if action == "staff_create" {
		phone := services.NormalizeNigerianPhone(f["phone"])
		if phone == "" {
			err = errors.New("enter a valid Nigerian phone")
		}
		f["phone"] = phone
		password := f["password"]
		if len(password) < 12 || len(password) > 72 {
			err = errors.New("use a staff password between 12 and 72 bytes")
		}
		if err == nil {
			var hash []byte
			hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			f["password_hash"] = string(hash)
		}
		delete(f, "password")
	}
	if err == nil {
		if section == "finance" {
			err = h.walletAction(r, user, f)
		} else {
			err = h.repo.Apply(r.Context(), user, f)
		}
	}
	if err != nil {
		h.auditFailure(r, user, action, f["id"], "Action failed")
		status := 400
		message := "Could not save the change. Check the record and try again."
		if errors.Is(err, repository.ErrAdminForbidden) {
			status = 403
			message = err.Error()
		} else if errors.Is(err, repository.ErrAdminInvalid) || errors.Is(err, repository.ErrPhoneExists) || errors.Is(err, repository.ErrEmailExists) {
			message = err.Error()
		} else if action == "staff_create" {
			message = "Check the staff name, phone, email and password (12–72 bytes)."
		}
		log.Printf("admin action %s failed: %v", action, err)
		for _, item := range models.AdminSections {
			if item.Path == f["return_to"] && user.CanAdmin(item.Key) {
				data, pageErr := h.repo.Page(r.Context(), user, item, "", 1)
				if pageErr == nil {
					data.Error = message
					data.CSRF = adminCSRF(r)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(status)
					if renderErr := render.RenderTemplates(w, "admin.html", data); renderErr != nil {
						log.Printf("admin action error template: %v", renderErr)
					}
					return
				}
			}
		}
		http.Error(w, message, status)
		return
	}
	path := "/admin"
	for _, item := range models.AdminSections {
		if item.Path == f["return_to"] && user.CanAdmin(item.Key) {
			path = item.Path
			break
		}
	}
	http.Redirect(w, r, path+"?notice=saved", http.StatusSeeOther)
}

func (h *AdminWorkspaceHandler) auditFailure(r *http.Request, user *models.User, action, target, detail string) {
	if err := h.repo.Audit(r.Context(), user, action, target, "{}", "{}", false, detail); err != nil {
		log.Printf("record failed admin request: %v", err)
	}
}

func (h *AdminWorkspaceHandler) walletAction(r *http.Request, user *models.User, f map[string]string) error {
	id, err := strconv.ParseInt(f["id"], 10, 64)
	if err != nil || id <= 0 {
		return repository.ErrAdminInvalid
	}
	reason := strings.TrimSpace(f["reason"])
	if reason == "" || len(reason) > 2000 {
		return errors.New("record a reason for the payment decision")
	}
	kind := "transfer"
	if f["action"] == "wallet_withdrawal" {
		kind = "withdrawal"
	}
	if (kind == "transfer" && f["value"] != "completed" && f["value"] != "rejected") || (kind == "withdrawal" && f["value"] != "approve" && f["value"] != "check" && f["value"] != "rejected") {
		return repository.ErrAdminInvalid
	}
	before, err := h.repo.WalletSnapshot(r.Context(), kind, id)
	if err != nil {
		return err
	}
	target := kind + ":" + f["id"]
	// The existing payment service owns provider transactions. Persist intent
	// before calling it, followed by the observed result if the call completes.
	if err = h.repo.Audit(r.Context(), user, f["action"]+"_started", target, before, before, true, reason+"; decision="+f["value"]); err != nil {
		return err
	}
	if kind == "transfer" {
		err = h.wallet.DecideTransfer(r.Context(), user, id, f["value"])
	} else {
		err = h.wallet.DecideWithdrawal(r.Context(), user, id, f["value"])
	}
	after, snapshotErr := h.repo.WalletSnapshot(r.Context(), kind, id)
	if snapshotErr != nil {
		after = before
	}
	detail := reason
	if snapshotErr != nil {
		detail += "; result snapshot unavailable"
	}
	auditErr := h.repo.Audit(r.Context(), user, f["action"], target, before, after, err == nil, detail)
	if err != nil {
		return err
	}
	if snapshotErr != nil {
		return snapshotErr
	}
	return auditErr
}
