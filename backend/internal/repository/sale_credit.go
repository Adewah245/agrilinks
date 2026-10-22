package repository

import (
	"database/sql"
	"fmt"
)

// creditOrderSale is shared by wallet and provider payments. The saved order
// reference and order lock prevent the same sale from being credited twice.
func creditOrderSale(tx *sql.Tx, orderID int, sellerType string, sellerID int, amount, currency string, commissionBPS int64) error {
	var walletID int
	err := tx.QueryRow(`INSERT INTO wallets(owner_type,owner_id,currency) VALUES($1,$2,$3)
		ON CONFLICT(owner_type,owner_id,currency) DO UPDATE SET owner_id=EXCLUDED.owner_id RETURNING id`, sellerType, sellerID, currency).Scan(&walletID)
	if err != nil {
		return err
	}
	var net, fee string
	var hasNet, hasFee bool
	err = tx.QueryRow(`SELECT ($1::numeric-round($1::numeric*$2::numeric/10000,2))::text,round($1::numeric*$2::numeric/10000,2)::text,
		$1::numeric-round($1::numeric*$2::numeric/10000,2)>0,round($1::numeric*$2::numeric/10000,2)>0`, amount, commissionBPS).Scan(&net, &fee, &hasNet, &hasFee)
	if err != nil {
		return err
	}
	if hasNet {
		result, err := tx.Exec(`INSERT INTO wallet_transactions(wallet_id,type,amount,description,reference,reference_type,reference_id)
			VALUES($1,'sale_credit',$2::numeric,'Payment received for a product order',$3,'order',$4) ON CONFLICT(reference) DO NOTHING`, walletID, net, fmt.Sprintf("order-sale-%d", orderID), orderID)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 1 {
			if _, err = tx.Exec(`UPDATE wallets SET balance=balance+$2::numeric,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, net); err != nil {
				return err
			}
		}
	}
	if hasFee {
		_, err = tx.Exec(`INSERT INTO platform_revenue(order_id,reference,gross_amount,amount,currency,rate_percent)
			VALUES($1,$2,$3::numeric,$4::numeric,$5,$6::numeric/100) ON CONFLICT(reference) DO NOTHING`, orderID, fmt.Sprintf("order-fee-%d", orderID), amount, fee, currency, commissionBPS)
		if err != nil {
			return err
		}
	}
	return nil
}
