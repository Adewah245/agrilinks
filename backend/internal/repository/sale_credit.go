package repository

import (
	"backend/internal/money"
	"database/sql"
	"fmt"
)

// Called only under the order/escrow lock after all release checks pass.
func creditOrderSale(tx *sql.Tx, orderID int, sellerType string, sellerID int, amount, currency string, bps int64, actorID int) error {
	gross, err := money.Positive(amount)
	if err != nil {
		return err
	}
	fee, err := money.BasisPoints(gross, bps)
	if err != nil {
		return err
	}
	net := gross - fee
	var walletID int64
	if err = tx.QueryRow(`INSERT INTO wallets(owner_type,owner_id,currency) VALUES($1,$2,$3) ON CONFLICT(owner_type,owner_id,currency) DO UPDATE SET owner_id=EXCLUDED.owner_id RETURNING id`, sellerType, sellerID, currency).Scan(&walletID); err != nil {
		return err
	}
	if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
		return err
	}
	ref := fmt.Sprintf("order-sale-%d", orderID)
	if err = postJournal(tx, ref, "escrow_release", currency, actorID, "{}", debit(fmt.Sprintf("escrow:%d", orderID), gross), credit("seller_payouts_payable", net), credit("marketplace_commission", fee)); err != nil {
		return err
	}
	if err = postJournal(tx, ref+":wallet", "seller_wallet_credit", currency, actorID, "{}", debit("seller_payouts_payable", net), credit(walletAccount(walletID), net)); err != nil {
		return err
	}
	if err = walletEntry(tx, walletID, "sale_credit", net, ref); err != nil {
		return err
	}
	if err = creditWalletKobo(tx, walletID, net, ref); err != nil {
		return err
	}
	if fee > 0 {
		if _, err = tx.Exec(`INSERT INTO platform_revenue(order_id,reference,gross_amount,amount,currency,rate_percent) VALUES($1,$2,$3::numeric/100,$4::numeric/100,$5,$6::numeric/100)`, orderID, fmt.Sprintf("order-fee-%d", orderID), gross, fee, currency, bps); err != nil {
			return err
		}
	}
	return nil
}
