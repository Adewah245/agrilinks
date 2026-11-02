package handlers

import (
	"backend/internal/middleware"
	"backend/internal/render"
	"backend/internal/repository"
	"crypto/subtle"
	"net/http"
	"strconv"
)

type EscrowHandler struct{ repo *repository.EscrowRepository }

func NewEscrowHandler(repo *repository.EscrowRepository) *EscrowHandler {
	return &EscrowHandler{repo: repo}
}
func (h *EscrowHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		http.Error(w, "Please sign in", 401)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	id, _ := strconv.Atoi(r.FormValue("order_id"))
	if id <= 0 {
		http.Error(w, "Choose an order", 400)
		return
	}
	var actionErr error
	if r.Method == http.MethodPost {
		if subtle.ConstantTimeCompare([]byte(r.PostFormValue("csrf")), []byte(adminCSRF(r))) != 1 {
			http.Error(w, "Reload the order before continuing", 403)
			return
		}
		actionErr = h.repo.Act(r.Context(), user, id, r.PostFormValue("action"), r.PostFormValue("reason"))
		if actionErr == nil {
			http.Redirect(w, r, "/orders/protection?order_id="+strconv.Itoa(id)+"&notice=saved", 303)
			return
		}
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	page, err := h.repo.Page(r.Context(), user, id)
	if err != nil {
		http.Error(w, "Purchase protection is unavailable for this order", 403)
		return
	}
	page.CSRF = adminCSRF(r)
	if actionErr != nil {
		page.Error = actionErr.Error()
		w.WriteHeader(400)
	}
	if r.URL.Query().Get("notice") == "saved" {
		page.Notice = "Your action was saved in the order's audit history."
	}
	_ = render.RenderTemplates(w, "purchase_protection.html", page)
}
