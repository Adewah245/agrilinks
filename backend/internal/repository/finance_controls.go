package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend/internal/models"
)

// ChangePayload is a typed, reviewable snapshot. Only the fields for Kind are
// applied. Provider verification evidence is populated by the service layer.
type ChangePayload struct {
	FundingServiceKobo  int64     `json:"funding_service_kobo"`
	WithdrawalLowKobo   int64     `json:"withdrawal_low_kobo"`
	WithdrawalMidKobo   int64     `json:"withdrawal_mid_kobo"`
	WithdrawalHighKobo  int64     `json:"withdrawal_high_kobo"`
	ProviderBearer      string    `json:"provider_bearer"`
	ProviderQuoteTaxBPS int64     `json:"provider_quote_tax_bps"`
	EffectiveAt         time.Time `json:"effective_at"`
	CompanyName         string    `json:"company_name"`
	BankCode            string    `json:"bank_code"`
	BankName            string    `json:"bank_name"`
	AccountNumber       string    `json:"account_number"`
	AccountName         string    `json:"account_name"`
	Evidence            string    `json:"evidence"`
	PerTransferKobo     int64     `json:"per_transfer_kobo"`
	DailyLimitKobo      int64     `json:"daily_limit_kobo"`
	SeniorThresholdKobo int64     `json:"senior_threshold_kobo"`
	AmountKobo          int64     `json:"amount_kobo"`
	Account             string    `json:"account"`
	Release             bool      `json:"release"`
	StatementReference  string    `json:"statement_reference"`
}

var financeKey = regexp.MustCompile(`^[a-f0-9]{32}$`)

func validateChange(kind string, p ChangePayload) error {
	switch kind {
	case "fee_policy":
		if p.ProviderQuoteTaxBPS > 0 && len(p.Evidence) < 5 {
			return adminInvalid("provider quote tax treatment requires supporting evidence")
		}
		if p.FundingServiceKobo < 0 || p.WithdrawalLowKobo < 0 || p.WithdrawalMidKobo < 0 || p.WithdrawalHighKobo < 0 || p.FundingServiceKobo > 100000000 || p.WithdrawalLowKobo > 100000000 || p.WithdrawalMidKobo > 100000000 || p.WithdrawalHighKobo > 100000000 || p.ProviderQuoteTaxBPS < 0 || p.ProviderQuoteTaxBPS > 10000 || (p.ProviderBearer != "customer" && p.ProviderBearer != "platform") || p.EffectiveAt.IsZero() {
			return adminInvalid("invalid fee policy; supply rates, effective date and provider fee treatment")
		}
	case "company_bank":
		if len(p.CompanyName) < 3 || p.AccountName == "" || !regexp.MustCompile(`^[0-9]{10}$`).MatchString(p.AccountNumber) || p.BankCode == "" || p.BankName == "" || len(p.Evidence) < 5 {
			return adminInvalid("provide a verified company business account and business-ownership evidence")
		}
	case "transfer_limits":
		if p.PerTransferKobo <= 0 || p.DailyLimitKobo < p.PerTransferKobo || p.SeniorThresholdKobo <= 0 {
			return adminInvalid("invalid transfer limits")
		}
	case "reserve":
		if p.AmountKobo <= 0 || (p.Account != "company_operating_reserve" && p.Account != "chargeback_reserve") {
			return adminInvalid("choose an operating or chargeback reserve and positive amount")
		}
	case "statutory_obligation":
		if p.AmountKobo <= 0 || len(p.Evidence) < 5 {
			return adminInvalid("record the assessed obligation and supporting evidence; no tax rate is inferred")
		}
	case "opening_reconciliation":
		if p.AmountKobo <= 0 || p.Account != "provider_settled_cash" || len(p.StatementReference) < 3 || len(p.Evidence) < 5 {
			return adminInvalid("provide verified opening provider cash and a statement reference")
		}
	default:
		return adminInvalid("unknown financial change")
	}
	return nil
}
func (r *FinanceRepository) ProposeChange(ctx context.Context, user *models.User, key, kind, reason string, p ChangePayload) (int64, error) {
	if user == nil {
		return 0, ErrAdminForbidden
	}
	if !financeKey.MatchString(key) || len(strings.TrimSpace(reason)) < 5 || len(reason) > 2000 {
		return 0, adminInvalid("record a change reason and request key")
	}
	if err := validateChange(kind, p); err != nil {
		return 0, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return 0, err
	}
	actor, err := financeActor(ctx, tx, user.ID, "treasury")
	if err != nil {
		return 0, err
	}
	body, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO finance_change_requests(request_key,kind,payload,reason,initiated_by) VALUES($1,$2,$3::jsonb,$4,$5) ON CONFLICT(request_key) DO NOTHING RETURNING id`, key, kind, string(body), reason, user.ID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `SELECT id FROM finance_change_requests WHERE request_key=$1 AND kind=$2 AND payload=$3::jsonb AND reason=$4 AND initiated_by=$5`, key, kind, string(body), reason, user.ID).Scan(&id)
		if err != nil {
			return 0, ErrWalletRequestChanged
		}
		return id, nil
	}
	if err != nil {
		return 0, err
	}
	if err = financeEvent(tx, actor, fmt.Sprintf("change:%d", id), "finance_change_proposed", reason, string(body)); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

const changeColumns = `id,request_key,kind,payload::text,reason,status,decision_reason,initiated_by,approved_by,created_at`

func scanChange(row interface{ Scan(...any) error }) (*models.FinanceChange, error) {
	c := &models.FinanceChange{}
	err := row.Scan(&c.ID, &c.Key, &c.Kind, &c.Payload, &c.Reason, &c.Status, &c.DecisionReason, &c.InitiatedBy, &c.ApprovedBy, &c.CreatedAt)
	return c, err
}
func (r *FinanceRepository) Change(ctx context.Context, id int64) (*models.FinanceChange, error) {
	return scanChange(r.db.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM finance_change_requests WHERE id=$1`, id))
}

func (r *FinanceRepository) DecideChange(ctx context.Context, user *models.User, id int64, approve bool, reason string) error {
	if user == nil {
		return ErrAdminForbidden
	}
	if len(strings.TrimSpace(reason)) < 5 || len(reason) > 2000 {
		return adminInvalid("record the second administrator's decision reason")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	actor, err := financeActor(ctx, tx, user.ID, "treasury_approve")
	if err != nil {
		return err
	}
	c, err := scanChange(tx.QueryRowContext(ctx, `SELECT `+changeColumns+` FROM finance_change_requests WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if c.InitiatedBy == user.ID {
		return adminInvalid("a different administrator must review this change")
	}
	if c.Status != "pending" {
		return adminInvalid("this change has already been decided")
	}
	if _, err = financeActor(ctx, tx, c.InitiatedBy, "treasury"); err != nil {
		return err
	}
	var p ChangePayload
	if err = json.Unmarshal([]byte(c.Payload), &p); err != nil {
		return err
	}
	next := "rejected"
	if approve {
		if err = validateChange(c.Kind, p); err != nil {
			return err
		}
		if (c.Kind == "company_bank" || c.Kind == "opening_reconciliation") && actor.AdminRole() != "super_admin" {
			return ErrAdminForbidden
		}
		next = "approved"
		switch c.Kind {
		case "fee_policy":
			_, err = tx.ExecContext(ctx, `INSERT INTO wallet_fee_policies(withdrawal_provider_bearer,provider_quote_tax_bps,created_by,approved_by,reason,funding_service_kobo,withdrawal_low_kobo,withdrawal_mid_kobo,withdrawal_high_kobo,effective_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, p.ProviderBearer, p.ProviderQuoteTaxBPS, c.InitiatedBy, user.ID, c.Reason, p.FundingServiceKobo, p.WithdrawalLowKobo, p.WithdrawalMidKobo, p.WithdrawalHighKobo, p.EffectiveAt)
		case "company_bank":
			var bankID int64
			evidence, _ := json.Marshal(map[string]string{"verification": p.Evidence, "statement": p.StatementReference})
			err = tx.QueryRowContext(ctx, `INSERT INTO company_settlement_accounts(company_name,bank_name,bank_code,account_number,resolved_account_name,currency,created_by,approved_by,reason,verification_evidence) VALUES($1,$2,$3,$4,$5,'NGN',$6,$7,$8,$9::jsonb) ON CONFLICT(bank_code,account_number) DO NOTHING RETURNING id`, p.CompanyName, p.BankName, p.BankCode, p.AccountNumber, p.AccountName, c.InitiatedBy, user.ID, c.Reason, string(evidence)).Scan(&bankID)
			if errors.Is(err, sql.ErrNoRows) {
				err = tx.QueryRowContext(ctx, `SELECT id FROM company_settlement_accounts WHERE bank_code=$1 AND account_number=$2 AND lower(company_name)=lower($3) AND lower(resolved_account_name)=lower($4)`, p.BankCode, p.AccountNumber, p.CompanyName, p.AccountName).Scan(&bankID)
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO accounting_accounts(code,name,category,company_settlement_account_id) VALUES($1,$2,'asset',$3) ON CONFLICT(code) DO NOTHING`, fmt.Sprintf("company-bank:%d", bankID), p.CompanyName+" — "+p.BankName, bankID)
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO finance_transfer_policies(per_transfer_limit_kobo,daily_limit_kobo,senior_threshold_kobo,company_account_id,change_id) SELECT per_transfer_limit_kobo,daily_limit_kobo,senior_threshold_kobo,$1,$2 FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`, bankID, id)
			}
		case "transfer_limits":
			_, err = tx.ExecContext(ctx, `INSERT INTO finance_transfer_policies(per_transfer_limit_kobo,daily_limit_kobo,senior_threshold_kobo,company_account_id,change_id) SELECT $1,$2,$3,company_account_id,$4 FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`, p.PerTransferKobo, p.DailyLimitKobo, p.SeniorThresholdKobo, id)
		case "reserve":
			if p.Release {
				var n int64
				n, err = financeBalance(ctx, tx, p.Account)
				if err == nil && -n < p.AmountKobo {
					return adminInvalid("reserve release exceeds the recorded reserve")
				}
				if err == nil {
					err = postJournal(tx, fmt.Sprintf("change:%d", id), "reserve_release", "NGN", user.ID, c.Payload, debit(p.Account, p.AmountKobo), credit("reserve_allocation", p.AmountKobo))
				}
			} else {
				err = postJournal(tx, fmt.Sprintf("change:%d", id), "reserve_allocation", "NGN", user.ID, c.Payload, debit("reserve_allocation", p.AmountKobo), credit(p.Account, p.AmountKobo))
			}
		case "statutory_obligation":
			err = postJournal(tx, fmt.Sprintf("change:%d", id), "statutory_obligation", "NGN", user.ID, c.Payload, debit("statutory_expense", p.AmountKobo), credit("tax_statutory_payable", p.AmountKobo))
		case "opening_reconciliation":
			var n int64
			n, err = financeBalance(ctx, tx, "opening_suspense")
			if err == nil && n < p.AmountKobo {
				return adminInvalid("verified opening cash cannot exceed unreconciled opening funds")
			}
			var used bool
			if err == nil {
				err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM finance_change_requests WHERE kind='opening_reconciliation' AND status='approved' AND payload->>'statement_reference'=$1)`, p.StatementReference).Scan(&used)
			}
			if used {
				return adminInvalid("this opening statement was already reconciled")
			}
			if err == nil {
				err = postJournal(tx, fmt.Sprintf("change:%d", id), "opening_reconciliation", "NGN", user.ID, c.Payload, debit("provider_settled_cash", p.AmountKobo), credit("opening_suspense", p.AmountKobo))
			}
		}
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE finance_change_requests SET status=$2,approved_by=$3,decided_at=CURRENT_TIMESTAMP,decision_reason=$4 WHERE id=$1`, id, next, user.ID, reason); err != nil {
		return err
	}
	if err = financeEvent(tx, actor, fmt.Sprintf("change:%d", id), "finance_change_"+next, reason, c.Payload); err != nil {
		return err
	}
	return tx.Commit()
}
