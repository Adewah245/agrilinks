package repository

import (
	"context"

	"backend/internal/models"
)

type FundingCheck struct{ Reference, ProviderID string }

// ClaimPendingFunding rotates unfinished deposits across workers. Updating the
// check time before the HTTP call prevents failing rows from starving others.
func (r *WalletRepository) ClaimPendingFunding(ctx context.Context, limit int) ([]FundingCheck, error) {
	rows, err := r.db.QueryContext(ctx, `WITH due AS (
        SELECT id FROM wallet_deposits WHERE provider='flutterwave' AND status IN ('pending','processing')
        AND updated_at <= CURRENT_TIMESTAMP - interval '2 minutes'
        ORDER BY updated_at,id LIMIT $1 FOR UPDATE SKIP LOCKED
    ), claimed AS (
        UPDATE wallet_deposits d SET updated_at=CURRENT_TIMESTAMP FROM due WHERE d.id=due.id RETURNING d.*
    ) SELECT CASE WHEN d.reference='VA-'||d.provider_transaction_id THEN a.tx_ref ELSE d.reference END,
        COALESCE(d.provider_transaction_id,'')
    FROM claimed d JOIN wallets w ON w.id=d.wallet_id
    LEFT JOIN wallet_virtual_accounts a ON a.user_id=w.owner_id AND w.owner_type='user'`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	checks := []FundingCheck{}
	for rows.Next() {
		var check FundingCheck
		if err := rows.Scan(&check.Reference, &check.ProviderID); err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, rows.Err()
}

// Flutterwave's reference endpoint returns one transaction. Rotate active
// accounts to recover the latest missed notification; older missing transfers
// can be verified individually by finance using their provider transaction ID.
func (r *WalletRepository) ClaimActiveFunding(ctx context.Context, limit int) ([]models.WalletVirtualAccount, error) {
	rows, err := r.db.QueryContext(ctx, `WITH due AS (
        SELECT id FROM wallet_virtual_accounts WHERE status='active' AND provider='flutterwave'
        AND (last_checked_at IS NULL OR last_checked_at <= CURRENT_TIMESTAMP - interval '2 minutes')
        ORDER BY last_checked_at NULLS FIRST,id LIMIT $1 FOR UPDATE SKIP LOCKED
    ) UPDATE wallet_virtual_accounts a SET last_checked_at=CURRENT_TIMESTAMP
        FROM due WHERE a.id=due.id RETURNING a.user_id,a.tx_ref`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []models.WalletVirtualAccount{}
	for rows.Next() {
		var account models.WalletVirtualAccount
		if err := rows.Scan(&account.UserID, &account.TxRef); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// Revision contains only committed, user-visible state. Background check
// timestamps deliberately do not trigger browser reloads.
func (r *WalletRepository) WalletRevision(ctx context.Context, userID int) (string, error) {
	var revision string
	err := r.db.QueryRowContext(ctx, `SELECT md5(concat_ws(':',w.balance,w.updated_at,
        (SELECT string_agg(d.id::text||':'||d.status,',' ORDER BY d.id) FROM
            (SELECT id,status FROM wallet_deposits WHERE wallet_id=w.id ORDER BY id DESC LIMIT 100) d),
        a.status,a.account_number,a.bank_name,a.account_name))
        FROM wallets w LEFT JOIN wallet_virtual_accounts a ON a.user_id=w.owner_id
        WHERE w.owner_type='user' AND w.owner_id=$1 AND w.currency=$2`, userID, r.currency).Scan(&revision)
	return revision, err
}
