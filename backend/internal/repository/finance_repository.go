package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
)

type FinanceRepository struct{ db *sql.DB }

func NewFinanceRepository(db *sql.DB) *FinanceRepository { return &FinanceRepository{db: db} }
func financeLock(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(721001)`)
	return err
}
func financeActor(ctx context.Context, tx *sql.Tx, userID int, permission string) (*models.User, error) {
	u := &models.User{}
	err := tx.QueryRowContext(ctx, `SELECT id,full_name,is_admin,staff_role,staff_disabled,account_status FROM users WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, userID).Scan(&u.ID, &u.FullName, &u.IsAdmin, &u.StaffRole, &u.StaffDisabled, &u.AccountStatus)
	if err != nil {
		return nil, err
	}
	if !u.CanAdmin(permission) {
		return nil, ErrAdminForbidden
	}
	return u, nil
}
func financeEvent(tx *sql.Tx, user *models.User, reference, action, reason, evidence string) error {
	if evidence == "" {
		evidence = "{}"
	}
	var id any
	name, role := "System", "system"
	if user != nil {
		id = user.ID
		name = user.FullName
		role = user.AdminRole()
	}
	if _, err := tx.Exec(`INSERT INTO finance_action_events(reference,actor_id,action,reason,evidence) VALUES($1,$2,$3,$4,$5::jsonb)`, reference, id, action, reason, evidence); err != nil {
		return err
	}
	if user != nil {
		_, err := tx.Exec(`INSERT INTO admin_audit_logs(actor_id,actor_name,actor_role,action,target,before_value,after_value,succeeded,detail) VALUES($1,$2,$3,$4,$5,'{}',$6::jsonb,true,$7)`, user.ID, name, role, action, reference, evidence, reason)
		return err
	}
	return nil
}

func financeBalance(ctx context.Context, q walletQuery, code string) (int64, error) {
	var n int64
	err := q.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.debit_kobo::numeric-p.credit_kobo::numeric),0)::bigint FROM accounting_postings p WHERE account_code=$1 AND currency='NGN'`, code).Scan(&n)
	return n, err
}
func availableCompany(ctx context.Context, q walletQuery) (available, provider, settled, obligations, reserves int64, err error) {
	// Gross revenue and unsettled receivables do not enter this calculation.
	err = q.QueryRowContext(ctx, `WITH b AS (SELECT a.code,a.category,COALESCE(SUM(p.debit_kobo::numeric-p.credit_kobo::numeric),0) net FROM accounting_accounts a LEFT JOIN accounting_postings p ON p.account_code=a.code AND p.currency='NGN' GROUP BY a.code,a.category)
 SELECT COALESCE(SUM(net) FILTER(WHERE code='provider_settled_cash' OR code LIKE 'company-bank:%'),0)::bigint,
 COALESCE(SUM(GREATEST(-net,0)) FILTER(WHERE category='liability'),0)::bigint,
 (COALESCE(SUM(GREATEST(-net,0)) FILTER(WHERE code IN ('company_operating_reserve','chargeback_reserve')),0)+COALESCE(SUM(GREATEST(-net,0)) FILTER(WHERE code='provider_clearing'),0))::bigint,
 COALESCE(SUM(net) FILTER(WHERE code='provider_settled_cash'),0)::bigint FROM b`).Scan(&settled, &obligations, &reserves, &provider)
	if err != nil {
		return
	}
	available = unrestrictedCash(settled, obligations, reserves)
	provider = min(unrestrictedCash(provider, obligations, reserves), available)
	return
}
func unrestrictedCash(cash, obligations, reserves int64) int64 {
	if cash <= 0 || obligations >= cash {
		return 0
	}
	cash -= obligations
	if reserves >= cash {
		return 0
	}
	return cash - reserves
}
func (r *FinanceRepository) Available(ctx context.Context) (int64, int64, error) {
	a, p, _, _, _, err := availableCompany(ctx, r.db)
	return a, p, err
}
func (r *FinanceRepository) TransferPolicy(ctx context.Context) (*models.TransferPolicy, error) {
	p := &models.TransferPolicy{}
	err := r.db.QueryRowContext(ctx, `SELECT id,per_transfer_limit_kobo,daily_limit_kobo,senior_threshold_kobo,company_account_id FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`).Scan(&p.ID, &p.PerTransferLimit, &p.DailyLimit, &p.SeniorThreshold, &p.CompanyAccountID)
	return p, err
}
func (r *FinanceRepository) CompanyAccount(ctx context.Context, id int64) (*models.CompanyAccount, error) {
	a := &models.CompanyAccount{}
	err := r.db.QueryRowContext(ctx, `SELECT id,company_name,bank_name,bank_code,account_number,resolved_account_name,currency FROM company_settlement_accounts WHERE id=$1`, id).Scan(&a.ID, &a.CompanyName, &a.BankName, &a.BankCode, &a.AccountNumber, &a.AccountName, &a.Currency)
	return a, err
}
func (r *FinanceRepository) SaveCompanyQuote(ctx context.Context, q *models.CompanyQuote) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO company_withdrawal_quotes(id,initiated_by,amount_kobo,provider_fee_kobo,total_kobo,company_account_id,policy_id,provider_evidence) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb) RETURNING expires_at`, q.ID, q.InitiatedBy, q.AmountKobo, q.ProviderKobo, q.TotalKobo, q.CompanyAccountID, q.PolicyID, q.ProviderEvidence).Scan(&q.ExpiresAt)
}
func (r *FinanceRepository) CompanyQuote(ctx context.Context, userID int, key string) (*models.CompanyQuote, error) {
	q := &models.CompanyQuote{}
	err := r.db.QueryRowContext(ctx, `SELECT id,initiated_by,amount_kobo,provider_fee_kobo,total_kobo,company_account_id,policy_id,provider_evidence::text,expires_at FROM company_withdrawal_quotes WHERE id=$1 AND initiated_by=$2`, key, userID).Scan(&q.ID, &q.InitiatedBy, &q.AmountKobo, &q.ProviderKobo, &q.TotalKobo, &q.CompanyAccountID, &q.PolicyID, &q.ProviderEvidence, &q.ExpiresAt)
	return q, err
}

const companyWithdrawalColumns = `id,reference,quote_id,currency,amount_kobo,quoted_provider_fee_kobo,total_reserved_kobo,company_account_id,initiated_by,approved_by,reason,approval_reason,status,COALESCE(provider_transfer_id,''),provider_submission_state,created_at,approved_at,verified_at,actual_provider_fee_kobo`

func scanCompanyWithdrawal(row interface{ Scan(...any) error }) (*models.CompanyWithdrawal, error) {
	w := &models.CompanyWithdrawal{}
	err := row.Scan(&w.ID, &w.Reference, &w.QuoteID, &w.Currency, &w.AmountKobo, &w.ProviderKobo, &w.TotalKobo, &w.CompanyAccountID, &w.InitiatedBy, &w.ApprovedBy, &w.Reason, &w.ApprovalReason, &w.Status, &w.ProviderID, &w.SubmissionState, &w.CreatedAt, &w.ApprovedAt, &w.VerifiedAt, &w.ActualProviderKobo)
	return w, err
}
func (r *FinanceRepository) CompanyWithdrawal(ctx context.Context, id int64) (*models.CompanyWithdrawal, error) {
	return scanCompanyWithdrawal(r.db.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE id=$1`, id))
}
func (r *FinanceRepository) CompanyWithdrawalByReference(ctx context.Context, ref string) (*models.CompanyWithdrawal, error) {
	return scanCompanyWithdrawal(r.db.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE reference=$1`, ref))
}

func (r *FinanceRepository) RequestCompanyWithdrawal(ctx context.Context, user *models.User, key, reason string) (*models.CompanyWithdrawal, error) {
	if user == nil {
		return nil, ErrAdminForbidden
	}
	if len(strings.TrimSpace(reason)) < 5 || len(reason) > 2000 {
		return nil, adminInvalid("record a withdrawal reason")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return nil, err
	}
	actor, err := financeActor(ctx, tx, user.ID, "treasury")
	if err != nil {
		return nil, err
	}
	ref := "COMPANY-" + key
	existing, err := scanCompanyWithdrawal(tx.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE reference=$1`, ref))
	if err == nil {
		if existing.InitiatedBy != user.ID || existing.Reason != reason {
			return nil, ErrWalletRequestChanged
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	q := &models.CompanyQuote{}
	err = tx.QueryRowContext(ctx, `SELECT amount_kobo,provider_fee_kobo,total_kobo,company_account_id,policy_id,provider_evidence::text FROM company_withdrawal_quotes WHERE id=$1 AND initiated_by=$2 AND expires_at>CURRENT_TIMESTAMP`, key, user.ID).Scan(&q.AmountKobo, &q.ProviderKobo, &q.TotalKobo, &q.CompanyAccountID, &q.PolicyID, &q.ProviderEvidence)
	if err != nil {
		return nil, adminInvalid("review a fresh company withdrawal quote")
	}
	var policyID, perTransfer, dailyLimit, currentDestination int64
	if err = tx.QueryRowContext(ctx, `SELECT id,per_transfer_limit_kobo,daily_limit_kobo,COALESCE(company_account_id,0) FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`).Scan(&policyID, &perTransfer, &dailyLimit, &currentDestination); err != nil {
		return nil, err
	}
	if policyID != q.PolicyID || currentDestination != q.CompanyAccountID {
		return nil, adminInvalid("company transfer settings changed; review a fresh quote")
	}
	if q.AmountKobo > perTransfer {
		return nil, adminInvalid("company transfer exceeds the approved per-transfer limit")
	}
	var spent int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_kobo::numeric),0)::bigint FROM company_withdrawals WHERE created_at>=date_trunc('day',CURRENT_TIMESTAMP AT TIME ZONE 'UTC') AND status NOT IN ('failed','reversed')`).Scan(&spent); err != nil {
		return nil, err
	}
	if spent > dailyLimit-q.AmountKobo {
		return nil, adminInvalid("company transfer exceeds the approved daily limit")
	}
	_, available, _, _, _, err := availableCompany(ctx, tx)
	if err != nil {
		return nil, err
	}
	if available < q.TotalKobo {
		return nil, adminInvalid("insufficient reconciled, settled and unrestricted company funds at the provider")
	}
	if err = postJournal(tx, ref+":reserve", "company_withdrawal_reservation", "NGN", user.ID, q.ProviderEvidence, debit("company_transfer_reserved", q.TotalKobo), credit("provider_settled_cash", q.TotalKobo)); err != nil {
		return nil, err
	}
	var id int64
	if err = tx.QueryRowContext(ctx, `INSERT INTO company_withdrawals(reference,quote_id,amount_kobo,quoted_provider_fee_kobo,total_reserved_kobo,company_account_id,initiated_by,reason) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, ref, key, q.AmountKobo, q.ProviderKobo, q.TotalKobo, q.CompanyAccountID, user.ID, reason).Scan(&id); err != nil {
		return nil, err
	}
	if err = financeEvent(tx, actor, ref, "company_withdrawal_requested", reason, "{}"); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.CompanyWithdrawal(ctx, id)
}

func (r *FinanceRepository) ApproveCompanyWithdrawal(ctx context.Context, user *models.User, id int64, approve bool, reason string) error {
	if user == nil {
		return ErrAdminForbidden
	}
	if len(strings.TrimSpace(reason)) < 5 || len(reason) > 2000 {
		return adminInvalid("record an approval decision reason")
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
	w, err := scanCompanyWithdrawal(tx.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if w.InitiatedBy == user.ID {
		return adminInvalid("the requesting administrator cannot approve or reject their own withdrawal")
	}
	if (approve && w.Status != "pending") || (!approve && w.Status != "pending" && w.Status != "approved") {
		return adminInvalid("the company withdrawal has already been reviewed or submitted")
	}
	var threshold, accountID int64
	if err = tx.QueryRowContext(ctx, `SELECT senior_threshold_kobo,COALESCE(company_account_id,0) FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`).Scan(&threshold, &accountID); err != nil {
		return err
	}
	if w.AmountKobo >= threshold && actor.AdminRole() != "super_admin" {
		return ErrAdminForbidden
	}
	next := "approved"
	if approve {
		if accountID != w.CompanyAccountID {
			return adminInvalid("the destination changed; reject this request and create a fresh quote")
		}
	} else {
		next = "failed"
		if err = reverseJournal(tx, w.Reference+":reserve", w.Reference+":reservation-reversal", "company withdrawal rejected"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE company_withdrawals SET status=$2,approved_by=$3,approved_at=CURRENT_TIMESTAMP,approval_reason=$4 WHERE id=$1`, id, next, user.ID, reason); err != nil {
		return err
	}
	if err = financeEvent(tx, actor, w.Reference, "company_withdrawal_"+next, reason, "{}"); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FinanceRepository) ClaimCompanyWithdrawal(ctx context.Context, id int64, executor *models.User) error {
	if executor == nil {
		return ErrAdminForbidden
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	actor, err := financeActor(ctx, tx, executor.ID, "treasury_approve")
	if err != nil {
		return err
	}
	w, err := scanCompanyWithdrawal(tx.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if w.Status != "approved" || w.SubmissionState != "not_submitted" || !w.ApprovedBy.Valid || int64(w.InitiatedBy) == w.ApprovedBy.Int64 {
		return errors.New("company transfer is not approved for execution")
	}
	if _, err = financeActor(ctx, tx, int(w.ApprovedBy.Int64), "treasury_approve"); err != nil {
		return err
	}
	if _, err = financeActor(ctx, tx, w.InitiatedBy, "treasury"); err != nil {
		return err
	}
	var destination int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(company_account_id,0) FROM finance_transfer_policies ORDER BY id DESC LIMIT 1`).Scan(&destination); err != nil {
		return err
	}
	var policyCurrent bool
	if err = tx.QueryRowContext(ctx, `SELECT policy_id=(SELECT id FROM finance_transfer_policies ORDER BY id DESC LIMIT 1) FROM company_withdrawal_quotes WHERE id=$1`, w.QuoteID).Scan(&policyCurrent); err != nil {
		return err
	}
	if !policyCurrent {
		return adminInvalid("transfer limits or destination changed; obtain a new approval")
	}
	if destination != w.CompanyAccountID {
		return adminInvalid("the approved company destination has changed")
	}

	// Pending requests already reduced settled cash; recheck obligations that
	// could have arisen since approval. Customer cover remains at the source.

	var deficit int64
	if err = tx.QueryRowContext(ctx, `WITH b AS (SELECT a.code,a.category,COALESCE(SUM(p.debit_kobo::numeric-p.credit_kobo::numeric),0) n FROM accounting_accounts a LEFT JOIN accounting_postings p ON p.account_code=a.code AND p.currency='NGN' GROUP BY a.code,a.category) SELECT GREATEST(0,COALESCE(SUM(GREATEST(-n,0)) FILTER(WHERE category='liability' OR code IN ('company_operating_reserve','chargeback_reserve','provider_clearing')),0)-COALESCE(SUM(n) FILTER(WHERE code='provider_settled_cash'),0))::bigint FROM b`).Scan(&deficit); err != nil {
		return err
	}
	if deficit > 0 {
		return adminInvalid("new financial obligations block this company transfer")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE company_withdrawals SET status='processing',provider_submission_state='unknown',processed_at=CURRENT_TIMESTAMP WHERE id=$1`, id); err != nil {
		return err
	}
	if err = financeEvent(tx, actor, w.Reference, "company_withdrawal_execution_started", "Approved transfer claimed once before provider submission", "{}"); err != nil {
		return err
	}
	return tx.Commit()
}

func claimProviderTransfer(tx *sql.Tx, provider, id, reference string) error {
	if id == "" {
		return errors.New("provider transfer identity is missing")
	}
	if _, err := tx.Exec(`INSERT INTO provider_transfer_receipts(provider,transfer_id,reference) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, provider, id, reference); err != nil {
		return err
	}
	var owner string
	if err := tx.QueryRow(`SELECT reference FROM provider_transfer_receipts WHERE provider=$1 AND transfer_id=$2`, provider, id).Scan(&owner); err != nil {
		return err
	}
	if owner != reference {
		return errors.New("provider transfer belongs to a different operation")
	}
	return nil
}
func (r *FinanceRepository) SaveCompanyTransferID(ctx context.Context, id int64, providerID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	w, err := scanCompanyWithdrawal(tx.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if w.Status != "processing" && w.Status != "successful" && w.Status != "failed" && w.Status != "reversed" {
		return errors.New("company withdrawal is not processing")
	}
	if w.ProviderID != "" && w.ProviderID != providerID {
		return errors.New("company transfer identity changed")
	}
	if err = claimProviderTransfer(tx, "flutterwave", providerID, w.Reference); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE company_withdrawals SET provider_transfer_id=$2,provider_submission_state='accepted' WHERE id=$1`, id, providerID); err != nil {
		return err
	}
	return tx.Commit()
}

type CompanySettlementProof struct {
	Actor              *models.User
	StatementReference string
	ReceivedKobo       int64
	Evidence           string
}

func (r *FinanceRepository) FinishCompanyWithdrawal(ctx context.Context, id int64, status string, actualFee int64, evidence string, proofs ...CompanySettlementProof) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	w, err := scanCompanyWithdrawal(tx.QueryRowContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if w.Status == status {
		return nil
	}
	if actualFee < 0 {
		return errors.New("invalid provider cost")
	}
	bankAccount := fmt.Sprintf("company-bank:%d", w.CompanyAccountID)
	if err = ensureAccount(tx, bankAccount, "asset"); err != nil {
		return err
	}
	switch status {
	case "successful":
		if len(proofs) != 1 || proofs[0].Actor == nil {
			return errors.New("bank settlement confirmation is required")
		}
		proof := proofs[0]
		actor, authErr := financeActor(ctx, tx, proof.Actor.ID, "treasury_approve")
		if authErr != nil {
			return authErr
		}
		if actor.ID == w.InitiatedBy || proof.ReceivedKobo != w.AmountKobo || len(strings.TrimSpace(proof.StatementReference)) < 3 || !json.Valid([]byte(proof.Evidence)) {
			return errors.New("a separate administrator must confirm the exact bank receipt with statement evidence")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO company_transfer_confirmations(withdrawal_id,company_account_id,statement_reference,received_kobo,confirmed_by,evidence) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, id, w.CompanyAccountID, proof.StatementReference, proof.ReceivedKobo, actor.ID, proof.Evidence); err != nil {
			return err
		}
		if err = financeEvent(tx, actor, w.Reference, "company_bank_receipt_confirmed", proof.StatementReference, proof.Evidence); err != nil {
			return err
		}
		if w.Status != "processing" || w.ProviderID == "" {
			return errors.New("company transfer has not been verified")
		}
		// The quoted amount is reserved; an overage remains an explicit provider
		// payable and further company withdrawals are restricted by that obligation.
		covered := min(actualFee, w.ProviderKobo)
		refund := w.ProviderKobo - covered
		if err = postJournal(tx, w.Reference+":paid", "company_bank_transfer", "NGN", 0, evidence, debit(bankAccount, w.AmountKobo), debit("provider_expense", actualFee), debit("provider_settled_cash", refund), credit("company_transfer_reserved", w.TotalKobo), credit("provider_payable", actualFee-covered)); err != nil {
			return err
		}
	case "failed":
		if w.Status != "processing" {
			return errors.New("company transfer cannot be refunded in its current state")
		}
		if err = reverseJournal(tx, w.Reference+":reserve", w.Reference+":reservation-reversal", "provider verified company payout failure"); err != nil {
			return err
		}
		if actualFee > 0 {
			if err = postJournal(tx, w.Reference+":failed-provider-fee", "provider_cost", "NGN", 0, evidence, debit("provider_expense", actualFee), credit("provider_payable", actualFee)); err != nil {
				return err
			}
		}
	case "reversed":
		if w.Status != "successful" {
			return errors.New("only a successful company transfer can be reversed")
		}
		if err = postJournal(tx, w.Reference+":reversed", "company_payout_reversal", "NGN", 0, evidence, debit("provider_settled_cash", w.AmountKobo), credit(bankAccount, w.AmountKobo)); err != nil {
			return err
		}
	default:
		return errors.New("invalid company transfer outcome")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE company_withdrawals SET status=$2,actual_provider_fee_kobo=$3,verified_at=CURRENT_TIMESTAMP WHERE id=$1`, id, status, actualFee); err != nil {
		return err
	}
	if err = financeEvent(tx, nil, w.Reference, "company_withdrawal_"+status, "Provider-verified transfer outcome", evidence); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FinanceRepository) ProcessingCompanyWithdrawals(ctx context.Context) ([]models.CompanyWithdrawal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+companyWithdrawalColumns+` FROM company_withdrawals WHERE status='processing' ORDER BY processed_at LIMIT 30`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.CompanyWithdrawal{}
	for rows.Next() {
		w, err := scanCompanyWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}
