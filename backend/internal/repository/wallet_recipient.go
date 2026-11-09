package repository

import (
	"backend/internal/models"
	"context"
)

// FindWalletRecipients reads existing user wallets without creating accounts.
// The caller supplies normalized aliases, and two rows expose ambiguous matches.
func (r *WalletRepository) FindWalletRecipients(ctx context.Context, wallet string, userID int, phone string) ([]models.WalletRecipient, error) {
	local, subscriber := "", ""
	if len(phone) == 13 {
		subscriber = phone[3:]
		local = "0" + subscriber
	}
	rows, err := r.db.QueryContext(ctx, `SELECT u.id,u.full_name,w.wallet_number,w.currency,COALESCE(u.photo_url,'')
		FROM wallets w JOIN users u ON u.id=w.owner_id
		WHERE w.owner_type='user' AND w.currency=$1 AND u.deleted_at IS NULL
		AND u.account_status='active' AND NOT u.staff_disabled
		AND (($2<>'' AND UPPER(w.wallet_number)=$2) OR ($3>0 AND u.id=$3)
		OR ($4<>'' AND regexp_replace(u.phone,'[^0-9]','','g') IN ($4,$5,$6)))
		ORDER BY u.id LIMIT 2`, r.currency, wallet, userID, phone, local, subscriber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recipients := make([]models.WalletRecipient, 0, 2)
	for rows.Next() {
		var recipient models.WalletRecipient
		if err := rows.Scan(&recipient.UserID, &recipient.FullName, &recipient.WalletNumber, &recipient.Currency, &recipient.PhotoURL); err != nil {
			return nil, err
		}
		recipients = append(recipients, recipient)
	}
	return recipients, rows.Err()
}
