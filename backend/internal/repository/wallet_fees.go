package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
	"backend/internal/money"
)

func (r *WalletRepository) FeePolicy(ctx context.Context) (*models.FeePolicy, error) {
	return feePolicy(ctx, r.db)
}
func feePolicy(ctx context.Context, db walletQuery) (*models.FeePolicy, error) {
	p := &models.FeePolicy{}
	err := db.QueryRowContext(ctx, `SELECT id,withdrawal_provider_bearer,provider_quote_tax_bps,reason,created_at,funding_service_kobo,withdrawal_low_kobo,withdrawal_mid_kobo,withdrawal_high_kobo,effective_at FROM wallet_fee_policies WHERE effective_at<=CURRENT_TIMESTAMP ORDER BY effective_at DESC,id DESC LIMIT 1`).Scan(&p.ID, &p.WithdrawalProviderBearer, &p.ProviderQuoteTaxBPS, &p.Reason, &p.CreatedAt, &p.FundingServiceKobo, &p.WithdrawalLowKobo, &p.WithdrawalMidKobo, &p.WithdrawalHighKobo, &p.EffectiveAt)
	return p, err
}

const quoteColumns = `id,user_id,kind,currency,amount_kobo,service_kobo,provider_kobo,customer_provider_kobo,wallet_debit_kobo,wallet_credit_kobo,bank_credit_kobo,checkout_total_kobo,payment_method,bank_code,account_number,account_name,bank_name,policy_id,provider_evidence::text,expires_at,allocation_reference`

func scanFeeQuote(row interface{ Scan(...any) error }) (*models.FeeQuote, error) {
	q := &models.FeeQuote{}
	err := row.Scan(&q.ID, &q.UserID, &q.Kind, &q.Currency, &q.AmountKobo, &q.ServiceKobo, &q.ProviderKobo, &q.CustomerProviderKobo, &q.WalletDebitKobo, &q.WalletCreditKobo, &q.BankCreditKobo, &q.CheckoutTotalKobo, &q.PaymentMethod, &q.BankCode, &q.AccountNumber, &q.AccountName, &q.BankName, &q.PolicyID, &q.ProviderEvidence, &q.ExpiresAt, &q.AllocationReference)
	return q, err
}
func (r *WalletRepository) SaveFeeQuote(ctx context.Context, q *models.FeeQuote) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO wallet_fee_quotes(id,user_id,kind,currency,amount_kobo,service_kobo,provider_kobo,customer_provider_kobo,wallet_debit_kobo,wallet_credit_kobo,bank_credit_kobo,checkout_total_kobo,payment_method,bank_code,account_number,account_name,bank_name,policy_id,provider_evidence,allocation_reference) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::jsonb,$20) RETURNING expires_at`, q.ID, q.UserID, q.Kind, q.Currency, q.AmountKobo, q.ServiceKobo, q.ProviderKobo, q.CustomerProviderKobo, q.WalletDebitKobo, q.WalletCreditKobo, q.BankCreditKobo, q.CheckoutTotalKobo, q.PaymentMethod, q.BankCode, q.AccountNumber, q.AccountName, q.BankName, q.PolicyID, q.ProviderEvidence, q.AllocationReference).Scan(&q.ExpiresAt)
}
func (r *WalletRepository) FeeQuote(ctx context.Context, userID int, id, kind string, allowUsed bool) (*models.FeeQuote, error) {
	q, err := scanFeeQuote(r.db.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1 AND user_id=$2 AND kind=$3 AND (expires_at>CURRENT_TIMESTAMP OR $4)`, id, userID, kind, allowUsed))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("review a fresh fee quote before confirming this transaction")
	}
	return q, err
}

func walletEntry(tx *sql.Tx, walletID int64, kind string, n int64, reference string) error {
	if n == 0 {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,$2,$3::numeric/100,$4)`, walletID, kind, n, reference)
	return err
}

// Incoming money first repays verified reversal debt. Debt is never exposed as
// spendable balance, and every repayment remains visible in wallet history.
func creditWalletKobo(tx *sql.Tx, walletID, n int64, reference string) error {
	var debt int64
	var currency string
	if err := tx.QueryRow(`SELECT recovery_debt_kobo,currency FROM wallets WHERE id=$1 FOR UPDATE`, walletID).Scan(&debt, &currency); err != nil {
		return err
	}
	repayment := min(n, debt)
	if repayment > 0 {
		if err := ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
			return err
		}
		receivable := fmt.Sprintf("wallet-receivable:%d", walletID)
		if err := ensureAccount(tx, receivable, "asset"); err != nil {
			return err
		}
		if err := postJournal(tx, reference+":debt", "reversal_debt_repayment", currency, 0, "{}", debit(walletAccount(walletID), repayment), credit(receivable, repayment)); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`UPDATE wallets SET balance=balance+$2::numeric/100,recovery_debt_kobo=recovery_debt_kobo-$3,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, n-repayment, repayment); err != nil {
		return err
	}
	return walletEntry(tx, walletID, "debt_repayment", repayment, reference+":debt")
}

type DepositCosts struct{ ProviderKobo, ChargedKobo *int64 }

// Deposit fee, credit and revenue share the existing finalization transaction.
func applyDepositFees(tx *sql.Tx, walletID int64, reference string, gross int64, currency string, costs DepositCosts) error {
	var policyID int64
	if err := tx.QueryRow(`SELECT id FROM wallet_fee_policies WHERE effective_at<=CURRENT_TIMESTAMP ORDER BY effective_at DESC,id DESC LIMIT 1`).Scan(&policyID); err != nil {
		return err
	}
	var q *models.FeeQuote
	parts := strings.Split(reference, "-")
	if len(parts) == 3 && parts[0] == "WALLET" {
		var err error
		q, err = scanFeeQuote(tx.QueryRow(`SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1 AND kind='deposit' AND user_id=(SELECT owner_id FROM wallets WHERE id=$2)`, parts[2], walletID))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else if strings.HasPrefix(reference, "VA-") {
		var key string
		err := tx.QueryRow(`SELECT i.quote_id FROM wallet_funding_intents i JOIN wallets w ON w.owner_id=i.user_id AND w.owner_type='user' WHERE w.id=$1 AND i.collected_kobo=$2 AND i.state='pending' AND i.expires_at>CURRENT_TIMESTAMP ORDER BY i.created_at LIMIT 1 FOR UPDATE OF i`, walletID, gross).Scan(&key)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			q, err = scanFeeQuote(tx.QueryRow(`SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1`, key))
			if err != nil {
				return err
			}
		}
	}
	service, net, quoted, customerProvider := int64(0), int64(0), int64(0), int64(0)
	received := gross
	if costs.ChargedKobo != nil {
		if *costs.ChargedKobo < gross {
			return errors.New("verified collection is less than the payment amount")
		}
		received = *costs.ChargedKobo
	}
	state := "review"
	var quoteID any
	if q != nil && q.AmountKobo+q.ServiceKobo == gross && (costs.ChargedKobo == nil || *costs.ChargedKobo == q.CheckoutTotalKobo) {
		state = "successful"
		quoteID = q.ID
		policyID = q.PolicyID
		service = q.ServiceKobo
		net = q.AmountKobo
		quoted = q.ProviderKobo
		customerProvider = received - gross
		if q.Kind == "funding_preview" {
			if _, err := tx.Exec(`UPDATE wallet_funding_intents SET state='matched',deposit_reference=$2 WHERE quote_id=$1`, q.ID, reference); err != nil {
				return err
			}
			if err := reverseJournal(tx, "intent:"+q.ID, "intent:"+q.ID+":confirmed", "funding received"); err != nil {
				return err
			}
		}
	}
	var actual any
	if costs.ProviderKobo != nil {
		if *costs.ProviderKobo < 0 {
			return errors.New("negative provider fee")
		}
		actual = *costs.ProviderKobo
	}
	principal := gross
	if net > 0 {
		principal = net
	}
	if _, err := tx.Exec(`INSERT INTO wallet_fee_records(reference,wallet_id,kind,currency,principal_kobo,service_kobo,provider_quoted_kobo,provider_actual_kobo,provider_customer_kobo,wallet_credit_kobo,state,quote_id,policy_id) VALUES($1,$2,'deposit',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, reference, walletID, currency, principal, service, quoted, actual, customerProvider, net, state, quoteID, policyID); err != nil {
		return err
	}
	if state == "review" {
		if err := postJournal(tx, reference+":receipt", "unallocated_funding", currency, 0, `{"reason":"no matching confirmed funding quote"}`, debit("provider_clearing", received), credit("funding_unmatched", received)); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE wallet_deposits SET review_required=true WHERE reference=$1`, reference); err != nil {
			return err
		}
	} else {
		if err := ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
			return err
		}
		if err := postJournal(tx, reference+":receipt", "wallet_funding", currency, 0, q.ProviderEvidence, debit("provider_clearing", received), credit(walletAccount(walletID), net), credit("funding_fee_revenue", service), credit("provider_cost_recovery", customerProvider)); err != nil {
			return err
		}
		if err := walletEntry(tx, walletID, "deposit", net, reference); err != nil {
			return err
		}
		// The fee is external to the wallet credit, so it belongs in the fee
		// breakdown rather than pretending it reduced the wallet balance.
		if err := creditWalletKobo(tx, walletID, net, reference); err != nil {
			return err
		}
	}
	if costs.ProviderKobo != nil {
		if err := postJournal(tx, reference+":provider-cost", "provider_cost", currency, 0, `{"source":"verified_app_fee"}`, debit("provider_expense", *costs.ProviderKobo), credit("provider_clearing", *costs.ProviderKobo)); err != nil {
			return err
		}
	}
	return nil
}

func reverseDepositFees(tx *sql.Tx, walletID int64, reference, currency string) error {
	var net, service, gross int64
	var state string
	err := tx.QueryRow(`SELECT wallet_credit_kobo,service_kobo,principal_kobo,state FROM wallet_fee_records WHERE reference=$1 FOR UPDATE`, reference).Scan(&net, &service, &gross, &state)
	if errors.Is(err, sql.ErrNoRows) {
		var amount string
		if err = tx.QueryRow(`SELECT amount::text FROM wallet_deposits WHERE reference=$1`, reference).Scan(&amount); err != nil {
			return err
		}
		gross, err = money.Positive(amount)
		if err != nil {
			return err
		}
		if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
			return err
		}
		if err = postJournal(tx, reference+":legacy-reversal", "legacy_funding_reversal", currency, 0, `{"source":"verified_provider_reversal","historical_reference":true}`, debit(walletAccount(walletID), gross), credit("provider_clearing", gross)); err != nil {
			return err
		}
		return reduceWalletForReversal(tx, walletID, gross, reference+":legacy-reversal", currency)
	}
	if err != nil {
		return err
	}
	if state == "reversed" {
		return nil
	}
	if err = reverseJournal(tx, reference+":receipt", reference+":receipt-reversal", "provider verified deposit reversal"); err != nil {
		return err
	}

	var allocation bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounting_journals WHERE reference=$1||':allocation')`, reference).Scan(&allocation); err != nil {
		return err
	}
	if allocation {
		if err = reverseJournal(tx, reference+":allocation", reference+":allocation-reversal", "provider verified allocated funding reversal"); err != nil {
			return err
		}
	}
	if state != "review" {
		var balance string
		var debt int64
		if err = tx.QueryRow(`SELECT balance::text,recovery_debt_kobo FROM wallets WHERE id=$1 FOR UPDATE`, walletID).Scan(&balance, &debt); err != nil {
			return err
		}
		available, err := money.Parse(balance)
		if err != nil {
			return err
		}
		taken := min(available, net)

		if net > taken {
			receivable := fmt.Sprintf("wallet-receivable:%d", walletID)
			if err = ensureAccount(tx, receivable, "asset"); err != nil {
				return err
			}
			if err = postJournal(tx, reference+":reversal-debt", "reversal_receivable", currency, 0, "{}", debit(receivable, net-taken), credit(walletAccount(walletID), net-taken)); err != nil {
				return err
			}
		}
		nextDebt, err := money.Add(debt, net-taken)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE wallets SET balance=balance-$2::numeric/100,recovery_debt_kobo=$3,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, taken, nextDebt); err != nil {
			return err
		}
		if err = walletEntry(tx, walletID, "deposit_reversal", net, reference+":reversal"); err != nil {
			return err
		}

	}
	_, err = tx.Exec(`UPDATE wallet_fee_records SET state='reversed',updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference)
	return err
}

func reserveWithdrawalFees(tx *sql.Tx, walletID int64, reference, currency string, principal int64, q *models.FeeQuote) error {
	for _, account := range []string{walletAccount(walletID), "withdrawal:" + reference, "fees:" + reference} {
		if err := ensureAccount(tx, account, "liability"); err != nil {
			return err
		}
	}
	if err := postJournal(tx, reference+":reserve", "withdrawal_reservation", currency, 0, "{}", debit(walletAccount(walletID), q.WalletDebitKobo), credit("withdrawal:"+reference, principal), credit("fees:"+reference, q.ServiceKobo+q.CustomerProviderKobo)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO wallet_fee_records(reference,wallet_id,kind,currency,principal_kobo,service_kobo,provider_quoted_kobo,provider_customer_kobo,wallet_debit_kobo,state,quote_id,policy_id) VALUES($1,$2,'withdrawal',$3,$4,$5,$6,$7,$8,'reserved',NULLIF($9,''),$10)`, reference, walletID, currency, principal, q.ServiceKobo, q.ProviderKobo, q.CustomerProviderKobo, q.WalletDebitKobo, q.ID, q.PolicyID); err != nil {
		return err
	}
	if err := walletEntry(tx, walletID, "service_fee", q.ServiceKobo, reference+":service"); err != nil {
		return err
	}
	return walletEntry(tx, walletID, "provider_fee", q.CustomerProviderKobo, reference+":provider")
}

func settleWithdrawalFees(tx *sql.Tx, reference, decision, currency string, actual *int64) (int64, error) {
	var walletID, principal, service, quoted, customer, totalDebit int64
	var state string
	err := tx.QueryRow(`SELECT wallet_id,principal_kobo,service_kobo,provider_quoted_kobo,provider_customer_kobo,wallet_debit_kobo,state FROM wallet_fee_records WHERE reference=$1 FOR UPDATE`, reference).Scan(&walletID, &principal, &service, &quoted, &customer, &totalDebit, &state)
	if errors.Is(err, sql.ErrNoRows) {
		var amount string
		var id int64
		if err = tx.QueryRow(`SELECT amount::text,wallet_id FROM wallet_withdrawals WHERE reference=$1`, reference).Scan(&amount, &id); err != nil {
			return 0, err
		}
		n, err := money.Positive(amount)
		if err != nil {
			return 0, err
		}
		destination := "provider_clearing"
		if decision != "paid" {
			destination = walletAccount(id)
			if err = ensureAccount(tx, destination, "liability"); err != nil {
				return 0, err
			}
		}
		return 0, postJournal(tx, reference+":legacy-outcome", "legacy_withdrawal_"+decision, currency, 0, "{}", debit("withdrawal:"+reference, n), credit(destination, n))
	}
	if err != nil {
		return 0, err
	}
	if state != "reserved" {
		return 0, errors.New("withdrawal fees have already been settled")
	}
	cost := quoted
	if actual != nil {
		cost = *actual
	}
	source := `{"source":"accepted_provider_quote"}`
	if actual != nil {
		source = `{"source":"verified_transfer_fee"}`
	}
	if decision == "paid" {
		if actual == nil && quoted > 0 {
			return 0, errors.New("actual provider charge must be verified before settlement")
		}
		charged := min(customer, cost)
		refund := customer - charged
		if err = postJournal(tx, reference+":paid", "withdrawal_payout", currency, 0, source, debit("withdrawal:"+reference, principal), credit("provider_clearing", principal), debit("fees:"+reference, service+customer), credit("withdrawal_fee_revenue", service), credit("provider_cost_recovery", charged), credit(walletAccount(walletID), refund)); err != nil {
			return 0, err
		}
		if err = postJournal(tx, reference+":provider-cost", "provider_cost", currency, 0, source, debit("provider_expense", cost), credit("provider_clearing", cost)); err != nil {
			return 0, err
		}
		if err = creditWalletKobo(tx, walletID, refund, reference+":provider-adjustment"); err != nil {
			return 0, err
		}
		if err = walletEntry(tx, walletID, "fee_refund", refund, reference+":provider-adjustment"); err != nil {
			return 0, err
		}
		_, err = tx.Exec(`UPDATE wallet_fee_records SET state='successful',provider_actual_kobo=$2,provider_customer_kobo=$3,wallet_debit_kobo=wallet_debit_kobo-$4,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference, cost, charged, refund)
		return 0, err
	}

	if err = reverseJournal(tx, reference+":reserve", reference+":reservation-reversal", "withdrawal "+decision); err != nil {
		return 0, err
	}
	if actual != nil && cost > 0 {
		if err = postJournal(tx, reference+":failed-provider-cost", "provider_cost", currency, 0, source, debit("provider_expense", cost), credit("provider_clearing", cost)); err != nil {
			return 0, err
		}
	}
	// Failed/cancelled payouts return every customer fee. An explicitly reported
	// failed-transfer cost is borne by the platform, not silently passed on.

	if err = walletEntry(tx, walletID, "fee_refund", service+customer, reference+":fees-refunded"); err != nil {
		return 0, err
	}
	_, err = tx.Exec(`UPDATE wallet_fee_records SET state='refunded',updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference)
	return totalDebit - principal, err
}

func (r *WalletRepository) MarkWithdrawalRefused(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE wallet_withdrawals SET provider_submission_state='refused' WHERE id=$1 AND status='processing' AND provider_transfer_id IS NULL AND provider_submission_state='unknown'`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("withdrawal requires provider verification")
	}
	return nil
}

func (r *WalletRepository) DepositRequestExists(ctx context.Context, userID int, reference string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_deposits d JOIN wallets w ON w.id=d.wallet_id WHERE d.reference=$1 AND w.owner_type='user' AND w.owner_id=$2)`, reference, userID).Scan(&exists)
	return exists, err
}
