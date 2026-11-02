package repository

import (
	"backend/internal/models"
	"context"
	"errors"
)

func (r *WalletRepository) UnallocatedFunding(ctx context.Context, userID int, reference string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.credit_kobo::numeric),0)::bigint FROM wallet_fee_records f JOIN wallets w ON w.id=f.wallet_id JOIN accounting_journals j ON j.reference=f.reference||':receipt' JOIN accounting_postings p ON p.journal_id=j.id AND p.account_code='funding_unmatched' WHERE f.reference=$1 AND f.state='review' AND w.owner_type='user' AND w.owner_id=$2`, reference, userID).Scan(&n)
	return n, err
}
func (r *WalletRepository) AllocateFunding(ctx context.Context, userID int, key string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	q, err := scanFeeQuote(tx.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1 AND user_id=$2 AND kind='funding_allocation' AND expires_at>CURRENT_TIMESTAMP`, key, userID))
	if err != nil {
		return err
	}
	var walletID int64
	var state, savedQuote string
	if err = tx.QueryRowContext(ctx, `SELECT f.wallet_id,f.state,COALESCE(f.quote_id,'') FROM wallet_fee_records f JOIN wallets w ON w.id=f.wallet_id WHERE f.reference=$1 AND w.owner_type='user' AND w.owner_id=$2 FOR UPDATE OF f,w`, q.AllocationReference, userID).Scan(&walletID, &state, &savedQuote); err != nil {
		return err
	}
	if state == "successful" && savedQuote == q.ID {
		return nil
	}
	if state != "review" {
		return errors.New("these funds have already been allocated or reversed")
	}
	var held int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.credit_kobo::numeric),0)::bigint FROM accounting_postings p JOIN accounting_journals j ON j.id=p.journal_id WHERE j.reference=$1||':receipt' AND p.account_code='funding_unmatched'`, q.AllocationReference).Scan(&held); err != nil {
		return err
	}
	if held != q.CheckoutTotalKobo || q.AmountKobo+q.ServiceKobo != held {
		return errors.New("the funding receipt no longer matches this review")
	}
	if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
		return err
	}
	if err = postJournal(tx, q.AllocationReference+":allocation", "funding_allocation", q.Currency, userID, q.ProviderEvidence, debit("funding_unmatched", held), credit(walletAccount(walletID), q.AmountKobo), credit("funding_fee_revenue", q.ServiceKobo)); err != nil {
		return err
	}
	if err = creditWalletKobo(tx, walletID, q.AmountKobo, q.AllocationReference); err != nil {
		return err
	}
	if err = walletEntry(tx, walletID, "deposit", q.AmountKobo, q.AllocationReference); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_fee_records SET state='successful',service_kobo=$2,principal_kobo=$3,wallet_credit_kobo=$3,quote_id=$4,policy_id=$5,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, q.AllocationReference, q.ServiceKobo, q.AmountKobo, q.ID, q.PolicyID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_deposits SET review_required=false,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, q.AllocationReference); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *WalletRepository) SaveAllocationQuote(ctx context.Context, q *models.FeeQuote) error {
	return r.SaveFeeQuote(ctx, q)
}
