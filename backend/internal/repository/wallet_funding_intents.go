package repository

import (
	"backend/internal/models"
	"context"
	"database/sql"
	"errors"
)

func (r *WalletRepository) AcceptFundingIntent(ctx context.Context, userID int, key string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	q, err := scanFeeQuote(tx.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1 AND user_id=$2 AND kind='funding_preview' AND expires_at>CURRENT_TIMESTAMP`, key, userID))
	if err != nil {
		return err
	}
	var ready bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_virtual_accounts WHERE user_id=$1 AND status='active')`, userID).Scan(&ready); err != nil {
		return err
	}
	if !ready {
		return errors.New("activate your funding bank account first")
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_funding_intents WHERE quote_id=$1)`, key).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT quote_id FROM wallet_funding_intents WHERE user_id=$1 AND state='pending' AND (expires_at<=CURRENT_TIMESTAMP OR collected_kobo=$2) FOR UPDATE`, userID, q.CheckoutTotalKobo)
	if err != nil {
		return err
	}
	var previous []string
	for rows.Next() {
		var k string
		if err = rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		previous = append(previous, k)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, k := range previous {
		if err = reverseJournal(tx, "intent:"+k, "intent:"+k+":expired", "replaced by a newly confirmed funding instruction"); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_funding_intents SET state='expired' WHERE user_id=$1 AND state='pending' AND (expires_at<=CURRENT_TIMESTAMP OR collected_kobo=$2)`, userID, q.CheckoutTotalKobo); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_funding_intents(quote_id,user_id,collected_kobo) VALUES($1,$2,$3)`, key, userID, q.CheckoutTotalKobo); err != nil {
		return err
	}
	if err = postJournal(tx, "intent:"+key, "funding_awaiting_confirmation", q.Currency, userID, q.ProviderEvidence, debit("pending_provider_collections", q.CheckoutTotalKobo), credit("pending_customer_deposits", q.CheckoutTotalKobo)); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *WalletRepository) ActiveFundingIntent(ctx context.Context, userID int) (*models.FeeQuote, error) {
	var key string
	err := r.db.QueryRowContext(ctx, `SELECT quote_id FROM wallet_funding_intents WHERE user_id=$1 AND state='pending' AND expires_at>CURRENT_TIMESTAMP ORDER BY created_at DESC LIMIT 1`, userID).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FeeQuote(ctx, userID, key, "funding_preview", true)
}
func (r *WalletRepository) FeeHistory(ctx context.Context, userID int) (map[string]models.FeeBreakdown, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT f.reference,f.service_kobo,f.provider_customer_kobo,f.wallet_credit_kobo,f.wallet_debit_kobo,f.state FROM wallet_fee_records f JOIN wallets w ON w.id=f.wallet_id WHERE w.owner_type='user' AND w.owner_id=$1 ORDER BY f.created_at DESC LIMIT 300`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.FeeBreakdown{}
	for rows.Next() {
		var ref string
		var f models.FeeBreakdown
		if err = rows.Scan(&ref, &f.ServiceKobo, &f.ProviderKobo, &f.CreditKobo, &f.DebitKobo, &f.State); err != nil {
			return nil, err
		}
		out[ref] = f
	}
	return out, rows.Err()
}

func (r *WalletRepository) ProtectedOrders(ctx context.Context, userID int) ([]models.Escrow, error) {
	return NewEscrowRepository(r.db).List(ctx, userID)
}
