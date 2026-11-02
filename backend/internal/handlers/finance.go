package handlers

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/render"
	"backend/internal/repository"
	"backend/internal/services"
)

type FinanceHandler struct{ service *services.FinanceService }

func NewFinanceHandler(s *services.FinanceService) *FinanceHandler {
	return &FinanceHandler{service: s}
}
func (h *FinanceHandler) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok || !user.CanAdmin("revenue") {
		http.Error(w, "Finance access is required", 403)
		return
	}
	if h.service == nil {
		http.Error(w, "Finance service is unavailable", 503)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet && r.URL.Query().Get("export") == "ledger" {
		var buf bytes.Buffer
		if err := h.service.Repo.Export(r.Context(), user, &buf, r.URL.Query().Get("from"), r.URL.Query().Get("to")); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="agrilinks-ledger.csv"`)
		_, _ = w.Write(buf.Bytes())
		return
	}
	var actionErr error
	var quote *models.CompanyQuote
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid financial form", 400)
			return
		}
		csrf := adminCSRF(r)
		if csrf == "" || subtle.ConstantTimeCompare([]byte(r.PostFormValue("csrf")), []byte(csrf)) != 1 {
			http.Error(w, "Reload the finance page before continuing", 403)
			return
		}
		action := r.PostFormValue("action")
		if action != "quote" && action != "send_code" {
			actionErr = h.service.Authenticate(r.Context(), user, r.PostFormValue("current_password"), r.PostFormValue("transaction_pin"), r.PostFormValue("code"))
		}
		id, _ := strconv.ParseInt(r.PostFormValue("id"), 10, 64)
		reason := r.PostFormValue("reason")
		if actionErr == nil {
			switch action {
			case "send_code":
				actionErr = h.service.SendCode(r.Context(), user)
				if r.Header.Get("Accept") == "application/json" {
					w.Header().Set("Content-Type", "application/json")
					if actionErr != nil {
						w.WriteHeader(400)
						_ = json.NewEncoder(w).Encode(map[string]string{"message": actionErr.Error()})
					} else {
						_ = json.NewEncoder(w).Encode(map[string]string{"message": "Finance code sent. Enter it to confirm your action."})
					}
					return
				}
			case "quote":
				quote, actionErr = h.service.Quote(r.Context(), user, r.PostFormValue("amount"))
			case "request":
				_, actionErr = h.service.Repo.RequestCompanyWithdrawal(r.Context(), user, r.PostFormValue("quote_id"), reason)
			case "approve", "reject":
				actionErr = h.service.Repo.ApproveCompanyWithdrawal(r.Context(), user, id, action == "approve", reason)
			case "execute":
				actionErr = h.service.Execute(r.Context(), user, id)
			case "check":
				actionErr = h.service.Check(r.Context(), id)
			case "confirm_receipt":
				actionErr = h.service.ConfirmBankReceipt(r.Context(), user, id, r.PostFormValue("statement_reference"), r.PostFormValue("amount"), reason)
			case "change":
				var payload repository.ChangePayload
				payload, actionErr = financeChangeForm(r)
				if actionErr == nil && r.PostFormValue("kind") == "company_bank" {
					actionErr = h.service.VerifyCompanyBank(r.Context(), &payload)
				}
				if actionErr == nil {
					_, actionErr = h.service.Repo.ProposeChange(r.Context(), user, r.PostFormValue("request_key"), r.PostFormValue("kind"), reason, payload)
				}
			case "approve_change", "reject_change":
				actionErr = h.service.DecideChange(r.Context(), user, id, action == "approve_change", reason)
			case "import_settlement":
				var batch models.SettlementBatch
				decoder := json.NewDecoder(bytes.NewBufferString(r.PostFormValue("statement")))
				decoder.DisallowUnknownFields()
				actionErr = decoder.Decode(&batch)
				if actionErr == nil {
					batch.Reason = reason
					_, actionErr = h.service.Repo.ImportSettlement(r.Context(), user, batch)
				}
			case "reconcile":
				actionErr = h.service.Repo.ReconcileSettlement(r.Context(), user, id, reason)
			case "reverse_settlement":
				actionErr = h.service.Repo.ReverseSettlement(r.Context(), user, id, reason)
			default:
				actionErr = errors.New("choose a financial action")
			}
		}
		if actionErr == nil && quote == nil {
			http.Redirect(w, r, "/admin/revenue?notice="+action, http.StatusSeeOther)
			return
		}
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	page, err := h.service.Repo.Page(r.Context(), user, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		log.Printf("finance page: %v", err)
		http.Error(w, "Could not load accounting records", 500)
		return
	}
	page.CSRF = adminCSRF(r)
	page.Quote = quote
	if actionErr != nil {
		page.Error = actionErr.Error()
		w.WriteHeader(400)
	} else if r.URL.Query().Get("notice") != "" {
		page.Notice = "The financial action was recorded. Check the status below; an approval does not confirm a bank transfer."
	}
	if err = render.RenderTemplates(w, "platform_revenue.html", page); err != nil {
		log.Printf("finance template: %v", err)
	}
}
func financeChangeForm(r *http.Request) (repository.ChangePayload, error) {
	p := repository.ChangePayload{CompanyName: r.PostFormValue("company_name"), BankCode: r.PostFormValue("bank_code"), BankName: r.PostFormValue("bank_name"), AccountNumber: r.PostFormValue("account_number"), Evidence: r.PostFormValue("evidence"), ProviderBearer: r.PostFormValue("provider_bearer"), Account: r.PostFormValue("account"), Release: r.PostFormValue("release") == "yes", StatementReference: r.PostFormValue("statement_reference")}
	for name, dest := range map[string]*int64{"funding_fee": &p.FundingServiceKobo, "withdrawal_low": &p.WithdrawalLowKobo, "withdrawal_mid": &p.WithdrawalMidKobo, "withdrawal_high": &p.WithdrawalHighKobo, "per_transfer": &p.PerTransferKobo, "daily_limit": &p.DailyLimitKobo, "senior_threshold": &p.SeniorThresholdKobo, "amount": &p.AmountKobo} {
		if value := r.PostFormValue(name); value != "" {
			n, err := money.Parse(value)
			if err != nil {
				return p, err
			}
			*dest = n
		}
	}
	if value := r.PostFormValue("provider_tax_bps"); value != "" {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return p, err
		}
		p.ProviderQuoteTaxBPS = n
	}
	if value := r.PostFormValue("effective_at"); value != "" {
		t, err := time.Parse("2006-01-02T15:04", value)
		if err != nil {
			return p, err
		}
		p.EffectiveAt = t
	}
	return p, nil
}
