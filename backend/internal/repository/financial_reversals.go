package repository

import (
	"backend/internal/money"
	"database/sql"
	"errors"
	"fmt"
)

// reduceWalletForReversal never exposes negative spendable money. A verified
// reversal of spent money becomes a separate receivable and blocks new debits.
// The caller posts the full liability debit before this projection adjustment.
func reduceWalletForReversal(tx *sql.Tx, walletID, n int64, reference, currency string) error {
	var balance, debt int64
	if err := tx.QueryRow(`SELECT (balance*100)::bigint,recovery_debt_kobo FROM wallets WHERE id=$1 FOR UPDATE`, walletID).Scan(&balance, &debt); err != nil {
		return err
	}
	taken := min(n, balance)
	unpaid := n - taken
	next, err := money.Add(debt, unpaid)
	if err != nil {
		return err
	}
	if unpaid > 0 {
		account := fmt.Sprintf("wallet-receivable:%d", walletID)
		if err = ensureAccount(tx, account, "asset"); err != nil {
			return err
		}
		if err = postJournal(tx, reference+":debt", "chargeback_receivable", currency, 0, "{}", debit(account, unpaid), credit(walletAccount(walletID), unpaid)); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE wallets SET balance=balance-$2::numeric/100,recovery_debt_kobo=$3,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, taken, next); err != nil {
		return err
	}
	return walletEntry(tx, walletID, "deposit_reversal", n, reference)
}
func reverseOrderPayment(tx *sql.Tx, reference, providerID, evidence string) error {
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM payment_reversals WHERE reference=$1)`, reference).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	var orderID int
	var amount, currency string
	if err := tx.QueryRow(`SELECT order_id,amount::text,currency FROM transactions WHERE payment_reference=$1 AND payment_provider='flutterwave' AND status='successful' AND provider_transaction_id=$2`, reference, providerID).Scan(&orderID, &amount, &currency); err != nil {
		return err
	}
	gross, err := money.Positive(amount)
	if err != nil {
		return err
	}
	e, err := scanEscrow(tx.QueryRow(`SELECT `+escrowSelect+escrowJoins+`WHERE e.order_id=$1 AND e.payment_reference=$2 FOR UPDATE OF e`, orderID, reference))
	if errors.Is(err, sql.ErrNoRows) {
		// Extra successful payments were held as refunds payable, never revenue.
		err = postJournal(tx, reference+":chargeback", "extra_payment_reversal", currency, 0, evidence, debit("customer_refunds_payable", gross), credit("provider_clearing", gross))
	} else if err == nil {
		switch e.State {
		case "held", "disputed":
			err = postJournal(tx, reference+":chargeback", "purchase_chargeback", currency, 0, evidence, debit(escrowLiability(e), gross), credit("provider_clearing", gross))
		case "released", "refunded":
			ownerType, ownerID, n := "user", e.BuyerID, gross
			if e.State == "released" {
				ownerType = e.SellerType
				ownerID = e.SellerID
				n = gross - e.CommissionKobo
			}
			var walletID int64
			err = tx.QueryRow(`SELECT id FROM wallets WHERE owner_type=$1 AND owner_id=$2 AND currency=$3 FOR UPDATE`, ownerType, ownerID, currency).Scan(&walletID)
			if err == nil {
				lines := []posting{debit(walletAccount(walletID), n), credit("provider_clearing", gross)}
				if e.State == "released" {
					lines = append(lines, debit("marketplace_commission", e.CommissionKobo))
				}
				err = postJournal(tx, reference+":chargeback", "purchase_chargeback", currency, 0, evidence, lines...)
			}
			if err == nil {
				err = reduceWalletForReversal(tx, walletID, n, reference+":chargeback-wallet", currency)
			}
		default:
			return errors.New("purchase cannot be reversed in its current state")
		}
		if err == nil {
			_, err = tx.Exec(`UPDATE order_escrows SET state='charged_back',updated_at=CURRENT_TIMESTAMP WHERE order_id=$1`, orderID)
		}
		if err == nil {
			_, err = tx.Exec(`INSERT INTO escrow_events(order_id,action,reason) VALUES($1,'provider_chargeback','Verified provider reversal; liability removed or recovery debt recorded')`, orderID)
		}
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO payment_reversals(reference,provider_transaction_id,amount_kobo,evidence) VALUES($1,$2,$3,$4::jsonb)`, reference, providerID, gross, evidence); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE transactions SET review_required=true,updated_at=CURRENT_TIMESTAMP WHERE payment_reference=$1`, reference)
	return err
}
func reverseCustomerWithdrawal(tx *sql.Tx, reference, currency string, walletID int64) error {
	var principal, service, provider, total int64
	err := tx.QueryRow(`SELECT principal_kobo,service_kobo,provider_customer_kobo,wallet_debit_kobo FROM wallet_fee_records WHERE reference=$1 AND state='successful' FOR UPDATE`, reference).Scan(&principal, &service, &provider, &total)
	if err != nil {
		return err
	}
	if err = postJournal(tx, reference+":reversed", "withdrawal_reversal", currency, 0, `{"source":"verified_provider_reversal"}`, debit("provider_clearing", principal), debit("withdrawal_fee_revenue", service), debit("provider_cost_recovery", provider), credit(walletAccount(walletID), total)); err != nil {
		return err
	}
	if err = creditWalletKobo(tx, walletID, total, reference+":reversed"); err != nil {
		return err
	}
	if err = walletEntry(tx, walletID, "withdrawal_refund", total, reference+":reversed"); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE wallet_fee_records SET state='reversed',updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference)
	return err
}
