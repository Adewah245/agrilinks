package repository

import (
	"context"
	"database/sql"
	"errors"
)

var ErrWalletRequestChanged = errors.New("this request has already been used with different details; reload the wallet to start a new request")

type walletQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func withdrawalMatches(ctx context.Context, q walletQuery, userID int, amount, reference, bankCode, number, currency string) (bool, error) {
	var matches bool
	err := q.QueryRowContext(ctx, `SELECT user_id=$2 AND amount=$3::numeric AND bank_code=$4 AND account_number=$5 AND currency=$6 FROM wallet_withdrawals WHERE reference=$1`, reference, userID, amount, bankCode, number, currency).Scan(&matches)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !matches {
		return false, ErrWalletRequestChanged
	}
	return true, nil
}
func (r *WalletRepository) WithdrawalMatches(ctx context.Context, userID int, amount, reference, bankCode, number string) (bool, error) {
	return withdrawalMatches(ctx, r.db, userID, amount, reference, bankCode, number, r.currency)
}
func transferMatches(ctx context.Context, q walletQuery, userID int, recipient, amount, reference, note, currency string) (bool, error) {
	var matches bool
	err := q.QueryRowContext(ctx, `SELECT t.sender_id=$2 AND w.wallet_number=$3 AND t.amount=$4::numeric AND t.note=$5 AND t.currency=$6 FROM wallet_transfers t JOIN wallets w ON w.id=t.recipient_wallet_id WHERE t.reference=$1`, reference, userID, recipient, amount, note, currency).Scan(&matches)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !matches {
		return false, ErrWalletRequestChanged
	}
	return true, nil
}
func (r *WalletRepository) TransferMatches(ctx context.Context, userID int, recipient, amount, reference, note string) (bool, error) {
	return transferMatches(ctx, r.db, userID, recipient, amount, reference, note, r.currency)
}

// Serialize retries across every server instance before checking for a receipt.
// The request and ledger commit together; rollback leaves the key reusable.
func lockWalletRequest(ctx context.Context, tx *sql.Tx, reference string) error {
	if reference == "" {
		return errors.New("a transaction reference is required")
	}
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,4137))`, reference)
	return err
}

// One Flutterwave receipt cannot fund a wallet and also pay an order.
func claimProviderReceipt(ctx context.Context, tx *sql.Tx, providerID, reference string) error {
	if providerID == "" {
		return errors.New("a provider receipt is required")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO payment_provider_receipts(provider,provider_id,reference) VALUES('flutterwave',$1,$2) ON CONFLICT DO NOTHING`, providerID, reference); err != nil {
		return err
	}
	var saved string
	if err := tx.QueryRowContext(ctx, `SELECT reference FROM payment_provider_receipts WHERE provider='flutterwave' AND provider_id=$1`, providerID).Scan(&saved); err != nil {
		return err
	}
	if saved != reference {
		return errors.New("this provider receipt is already assigned to another payment; finance review is required")
	}
	return nil
}

func (r *WalletRepository) ClaimDepositCheckout(ctx context.Context, reference string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE wallet_deposits SET checkout_claimed_at=CURRENT_TIMESTAMP WHERE reference=$1 AND status='pending' AND checkout_url='' AND checkout_claimed_at IS NULL`, reference)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *PaymentRepository) ClaimCheckout(reference string) (bool, error) {
	result, err := r.db.Exec(`UPDATE transactions SET checkout_claimed_at=CURRENT_TIMESTAMP WHERE payment_reference=$1 AND status='pending' AND checkout_url='' AND checkout_claimed_at IS NULL`, reference)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
