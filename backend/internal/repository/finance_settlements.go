package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/money"
)

// Imported statements are evidence, never spendable cash. A second finance
// administrator must reconcile all constituent receipts before cash is posted.
func (r *FinanceRepository) ImportSettlement(ctx context.Context, user *models.User, b models.SettlementBatch) (int64, error) {
	if user == nil {
		return 0, ErrAdminForbidden
	}
	if b.Direction == "" {
		b.Direction = "collection"
	}
	if b.Direction != "collection" && b.Direction != "customer_payout" && b.Direction != "provider_payable" {
		return 0, adminInvalid("invalid settlement direction")
	}
	sign := int64(-1)
	if b.Direction != "collection" {
		sign = 1
	}
	if b.Provider == "" || b.Reference == "" || b.Currency != "NGN" || b.Source == "" || len(b.Evidence) < 5 || len(b.Reason) < 5 || b.StatementReference == "" || len(b.Items) == 0 || len(b.Items) > 1000 || (b.Destination != "company_bank" && b.Destination != "provider_balance") {
		return 0, adminInvalid("provide the provider statement, destination, evidence and receipt details")
	}
	var gross, cost int64
	seen := map[string]bool{}
	for _, i := range b.Items {
		if seen[i.Reference] || i.Reference == "" || i.GrossKobo <= 0 || i.DeductionsKobo < 0 || i.NetKobo != i.GrossKobo+sign*i.DeductionsKobo {
			return 0, adminInvalid("invalid or duplicate settlement item")
		}
		seen[i.Reference] = true
		var err error
		gross, err = money.Add(gross, i.GrossKobo)
		if err != nil {
			return 0, err
		}
		cost, err = money.Add(cost, i.DeductionsKobo)
		if err != nil {
			return 0, err
		}
	}
	if gross != b.GrossKobo || cost != b.DeductionsKobo || b.NetKobo != gross+sign*cost || b.NetKobo < 0 {
		return 0, adminInvalid("settlement totals do not equal the receipt details")
	}
	if !json.Valid([]byte(b.Evidence)) {
		return 0, adminInvalid("settlement evidence must be JSON")
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
	var id int64
	var account any
	if b.Destination == "company_bank" {
		if b.CompanyAccountID <= 0 {
			return 0, adminInvalid("choose an approved business bank account")
		}
		account = b.CompanyAccountID
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO provider_settlement_batches(provider,provider_reference,currency,gross_kobo,deductions_kobo,net_kobo,destination,status,source,evidence,imported_by,statement_reference,statement_amount_kobo,company_account_id,reason,direction) VALUES($1,$2,$3,$4,$5,$6,$7,'imported',$8,$9::jsonb,$10,$11,$12,$13,$14,$15) ON CONFLICT(provider,provider_reference) DO NOTHING RETURNING id`, b.Provider, b.Reference, b.Currency, gross, cost, b.NetKobo, b.Destination, b.Source, b.Evidence, user.ID, b.StatementReference, b.StatementKobo, account, b.Reason, b.Direction).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var status string
		if err = tx.QueryRowContext(ctx, `SELECT id,status FROM provider_settlement_batches WHERE provider=$1 AND provider_reference=$2 FOR UPDATE`, b.Provider, b.Reference).Scan(&id, &status); err != nil {
			return 0, err
		}
		if status != "mismatch" {
			return 0, adminInvalid("this provider settlement was already imported")
		}
		// Only unposted mismatches may be corrected. Every imported version
		// remains immutable below, and fresh independent approval is required.
		if _, err = tx.ExecContext(ctx, `DELETE FROM provider_settlement_items WHERE batch_id=$1`, id); err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE provider_settlement_batches SET currency=$2,gross_kobo=$3,deductions_kobo=$4,net_kobo=$5,destination=$6,status='imported',source=$7,evidence=$8::jsonb,imported_by=$9,approved_by=NULL,statement_reference=$10,statement_amount_kobo=$11,company_account_id=$12,reason=$13,direction=$14 WHERE id=$1`, id, b.Currency, gross, cost, b.NetKobo, b.Destination, b.Source, b.Evidence, user.ID, b.StatementReference, b.StatementKobo, account, b.Reason, b.Direction); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	record, _ := json.Marshal(b)
	if _, err = tx.ExecContext(ctx, `INSERT INTO settlement_import_revisions(batch_id,imported_by,record) VALUES($1,$2,$3::jsonb)`, id, user.ID, string(record)); err != nil {
		return 0, err
	}
	for _, i := range b.Items {
		if _, err = tx.ExecContext(ctx, `INSERT INTO provider_settlement_items(batch_id,reference,gross_kobo,deductions_kobo,net_kobo) VALUES($1,$2,$3,$4,$5)`, id, i.Reference, i.GrossKobo, i.DeductionsKobo, i.NetKobo); err != nil {
			return 0, err
		}
	}
	if err = financeEvent(tx, actor, fmt.Sprintf("settlement:%d", id), "settlement_imported", b.Reason, b.Evidence); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

const settlementColumns = `id,provider,provider_reference,currency,destination,status,source,evidence::text,statement_reference,reason,gross_kobo,deductions_kobo,net_kobo,statement_amount_kobo,COALESCE(company_account_id,0),imported_by,approved_by,direction`

func scanSettlement(row interface{ Scan(...any) error }) (*models.SettlementBatch, error) {
	b := &models.SettlementBatch{}
	err := row.Scan(&b.ID, &b.Provider, &b.Reference, &b.Currency, &b.Destination, &b.Status, &b.Source, &b.Evidence, &b.StatementReference, &b.Reason, &b.GrossKobo, &b.DeductionsKobo, &b.NetKobo, &b.StatementKobo, &b.CompanyAccountID, &b.ImportedBy, &b.ApprovedBy, &b.Direction)
	return b, err
}
func (r *FinanceRepository) ReconcileSettlement(ctx context.Context, user *models.User, id int64, reason string) error {
	if user == nil {
		return ErrAdminForbidden
	}
	if len(strings.TrimSpace(reason)) < 5 {
		return adminInvalid("record the reconciliation review reason")
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
	b, err := scanSettlement(tx.QueryRowContext(ctx, `SELECT `+settlementColumns+` FROM provider_settlement_batches WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if b.ImportedBy == user.ID {
		return adminInvalid("a second administrator must reconcile the statement")
	}
	if b.Status == "matched" {
		return nil
	}
	if b.Status != "imported" && b.Status != "mismatch" {
		return adminInvalid("this settlement requires investigation")
	}
	rows, err := tx.QueryContext(ctx, `SELECT reference,gross_kobo,deductions_kobo,net_kobo FROM provider_settlement_items WHERE batch_id=$1 ORDER BY reference`, id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var i models.SettlementItem
		if err = rows.Scan(&i.Reference, &i.GrossKobo, &i.DeductionsKobo, &i.NetKobo); err != nil {
			rows.Close()
			return err
		}
		b.Items = append(b.Items, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	mismatch := b.NetKobo != b.StatementKobo
	details := []string{}
	if mismatch {
		details = append(details, "bank/provider balance statement differs from settlement net")
	}
	var extraCosts int64
	for _, i := range b.Items {
		if b.Direction != "collection" {
			if b.Direction == "provider_payable" {
				var due int64
				err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.credit_kobo::numeric-p.debit_kobo::numeric),0)::bigint FROM accounting_postings p JOIN accounting_journals j ON j.id=p.journal_id WHERE j.reference=$1 AND p.account_code='provider_payable' AND p.currency=$2`, i.Reference, b.Currency).Scan(&due)
				if err != nil {
					return err
				}
				if due != i.NetKobo || i.DeductionsKobo != 0 {
					mismatch = true
					details = append(details, i.Reference+": provider payable mismatch")
				}
			} else {
				var principal, cost int64
				var status string
				err = tx.QueryRowContext(ctx, `SELECT f.principal_kobo,COALESCE(f.provider_actual_kobo,0),f.state FROM wallet_fee_records f WHERE f.reference=$1 AND f.kind='withdrawal'`, i.Reference).Scan(&principal, &cost, &status)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return err
				}
				if err != nil || status != "successful" || principal != i.GrossKobo || cost != i.DeductionsKobo {
					mismatch = true
					details = append(details, i.Reference+": payout amount, fee or status mismatch")
				}
			}
			continue
		}
		var gross, recorded int64
		err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.debit_kobo::numeric-p.credit_kobo::numeric),0)::bigint FROM accounting_postings p JOIN accounting_journals j ON j.id=p.journal_id WHERE p.account_code='provider_clearing' AND p.currency=$2 AND j.reference IN ($1||':receipt',$1||':escrow')`, i.Reference, b.Currency).Scan(&gross)
		if err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.credit_kobo::numeric-p.debit_kobo::numeric),0)::bigint FROM accounting_postings p JOIN accounting_journals j ON j.id=p.journal_id WHERE p.account_code='provider_clearing' AND p.currency=$2 AND j.reference=$1||':provider-cost'`, i.Reference, b.Currency).Scan(&recorded)
		if err != nil {
			return err
		}
		var reversed bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounting_journals WHERE reference IN ($1||':receipt-reversal',$1||':chargeback'))`, i.Reference).Scan(&reversed)
		if err != nil {
			return err
		}
		var known bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_fee_records WHERE reference=$1 AND provider_actual_kobo IS NOT NULL) OR EXISTS(SELECT 1 FROM accounting_journals WHERE reference=$1||':provider-cost')`, i.Reference).Scan(&known)
		if err != nil {
			return err
		}
		if gross != i.GrossKobo || reversed || (known && recorded != i.DeductionsKobo) || recorded > i.DeductionsKobo {
			mismatch = true
			details = append(details, i.Reference+": missing, reversed, or mismatched receipt/cost")
		} else {
			extraCosts, err = money.Add(extraCosts, i.DeductionsKobo-recorded)
			if err != nil {
				return err
			}
		}
	}
	state := "matched"
	if mismatch {
		state = "mismatch"
	} else {
		destination := "provider_settled_cash"
		if b.Destination == "company_bank" {
			destination = fmt.Sprintf("company-bank:%d", b.CompanyAccountID)
		}
		if b.Direction == "collection" {
			err = postJournal(tx, fmt.Sprintf("settlement:%d", id), "provider_settlement", b.Currency, user.ID, b.Evidence, debit(destination, b.NetKobo), debit("provider_expense", extraCosts), credit("provider_clearing", b.NetKobo+extraCosts))
		} else {
			source := "provider_clearing"
			if b.Direction == "provider_payable" {
				source = "provider_payable"
			}
			err = postJournal(tx, fmt.Sprintf("settlement:%d", id), "provider_outbound_settlement", b.Currency, user.ID, b.Evidence, debit(source, b.NetKobo), credit(destination, b.NetKobo))
		}
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE provider_settlement_batches SET status=$2,approved_by=$3 WHERE id=$1`, id, state, user.ID); err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"mismatches": details, "statement_reference": b.StatementReference})
	if err = financeEvent(tx, actor, fmt.Sprintf("settlement:%d", id), "settlement_"+state, reason, string(evidence)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if mismatch {
		return adminInvalid("settlement mismatch recorded; no cash was made available")
	}
	return nil
}

// A provider collection reversal and its settlement reversal are distinct.
// This removes confirmed cash when a matched settlement is returned upstream.
func (r *FinanceRepository) ReverseSettlement(ctx context.Context, user *models.User, id int64, reason string) error {
	if user == nil || len(strings.TrimSpace(reason)) < 5 {
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
	actor, err := financeActor(ctx, tx, user.ID, "treasury_approve")
	if err != nil {
		return err
	}
	b, err := scanSettlement(tx.QueryRowContext(ctx, `SELECT `+settlementColumns+` FROM provider_settlement_batches WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return err
	}
	if b.ImportedBy == user.ID {
		return ErrAdminForbidden
	}
	if b.Status == "reversed" {
		return nil
	}
	if b.Status != "matched" {
		return adminInvalid("only matched settlements can be reversed")
	}
	ref := fmt.Sprintf("settlement:%d", id)
	if err = reverseJournal(tx, ref, ref+":reversal", reason); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE provider_settlement_batches SET status='reversed' WHERE id=$1`, id); err != nil {
		return err
	}
	if err = financeEvent(tx, actor, ref, "settlement_reversed", reason, "{}"); err != nil {
		return err
	}
	return tx.Commit()
}
