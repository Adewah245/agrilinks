package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/models"
	"backend/internal/money"
)

var ErrInsufficientBalance = errors.New("your wallet does not have enough money")

type WalletRepository struct {
	db            *sql.DB
	users         *UserRepository
	currency      string
	commissionBPS int64
}

func NewWalletRepository(db *sql.DB, currency string, commissionBPS int64, users ...*UserRepository) *WalletRepository {
	r := &WalletRepository{db: db, currency: currency, commissionBPS: commissionBPS}
	if len(users) > 0 {
		r.users = users[0]
	}
	return r
}

func (r *WalletRepository) Wallet(ctx context.Context, userID int) (*models.Wallet, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO wallets(owner_type,owner_id,currency) SELECT 'user',id,$2 FROM users WHERE id=$1 AND deleted_at IS NULL ON CONFLICT(owner_type,owner_id,currency) DO NOTHING`, userID, r.currency)
	if err != nil {
		return nil, err
	}
	w := &models.Wallet{OwnerType: "user"}
	err = r.db.QueryRowContext(ctx, `SELECT id,owner_id,balance::text,currency,wallet_number,created_at,updated_at FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency=$2`, userID, r.currency).
		Scan(&w.ID, &w.UserID, &w.Balance, &w.Currency, &w.WalletNumber, &w.CreatedAt, &w.UpdatedAt)
	return w, err
}

func (r *WalletRepository) History(ctx context.Context, userID int) ([]models.WalletTransaction, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id,t.type,t.amount::text,t.reference,t.created_at FROM wallet_transactions t JOIN wallets w ON w.id=t.wallet_id WHERE w.owner_type='user' AND w.owner_id=$1 AND w.currency=$2 ORDER BY t.created_at DESC,t.id DESC LIMIT 200`, userID, r.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.WalletTransaction{}
	for rows.Next() {
		var x models.WalletTransaction
		if err = rows.Scan(&x.ID, &x.Type, &x.Amount, &x.Reference, &x.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (r *WalletRepository) Deposits(ctx context.Context, userID int) ([]models.WalletDeposit, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.provider,d.amount::text,d.currency,d.reference,d.status,d.checkout_url,d.created_at FROM wallet_deposits d JOIN wallets w ON w.id=d.wallet_id WHERE w.owner_type='user' AND w.owner_id=$1 AND w.currency=$2 ORDER BY d.created_at DESC,d.id DESC LIMIT 100`, userID, r.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.WalletDeposit{}
	for rows.Next() {
		var x models.WalletDeposit
		if err = rows.Scan(&x.ID, &x.Provider, &x.Amount, &x.Currency, &x.Reference, &x.Status, &x.CheckoutURL, &x.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (r *WalletRepository) CreateDeposit(ctx context.Context, userID int, reference, amount string) (*models.WalletDeposit, error) {
	w, err := r.Wallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return nil, err
	}
	// Serialize browser reloads as well as retries carrying the original key.
	var walletID int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM wallets WHERE id=$1 FOR UPDATE`, w.ID).Scan(&walletID); err != nil {
		return nil, err
	}
	var requestedExists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_deposits WHERE reference=$1)`, reference).Scan(&requestedExists); err != nil {
		return nil, err
	}
	if !requestedExists {
		var unfinished bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_deposits WHERE wallet_id=$1 AND provider='flutterwave' AND status IN ('pending','processing') AND reference NOT LIKE 'VA-%')`, w.ID).Scan(&unfinished); err != nil {
			return nil, err
		}
		if unfinished {
			return nil, errors.New("an online top-up is already pending; continue it from funding history before starting another")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_deposits(wallet_id,provider,reference,amount,currency) VALUES($1,'flutterwave',$2,$3::numeric,$4)`, w.ID, reference, amount, r.currency); err != nil {
			return nil, err
		}
	}

	d := &models.WalletDeposit{}
	err = tx.QueryRowContext(ctx, `SELECT id,provider,amount::text,currency,reference,status,checkout_url,created_at FROM wallet_deposits WHERE reference=$1 AND wallet_id=$2 AND amount=$3::numeric`, reference, w.ID, amount).
		Scan(&d.ID, &d.Provider, &d.Amount, &d.Currency, &d.Reference, &d.Status, &d.CheckoutURL, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWalletRequestChanged
	}
	if err != nil {
		return nil, err
	}
	return d, tx.Commit()
}

func (r *WalletRepository) SaveDepositURL(ctx context.Context, reference, checkoutURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE wallet_deposits SET checkout_url=$2,updated_at=CURRENT_TIMESTAMP WHERE reference=$1 AND status='pending'`, reference, checkoutURL)
	return err
}

// FinalizeDeposit also handles transfers into a previously saved permanent
// account. The provider's transaction ID identifies each distinct bank transfer.
func (r *WalletRepository) FinalizeDeposit(ctx context.Context, reference, providerID, amount, currency, status string, event *models.WebhookEvent, providerCosts ...DepositCosts) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return false, err
	}
	var walletID int64
	var savedAmount, savedCurrency, current string
	var priorID sql.NullString
	var feeVersion int
	err = tx.QueryRowContext(ctx, `SELECT wallet_id,amount::text,currency,status,provider_transaction_id,fee_version FROM wallet_deposits WHERE reference=$1 AND provider='flutterwave' FOR UPDATE`, reference).
		Scan(&walletID, &savedAmount, &savedCurrency, &current, &priorID, &feeVersion)
	if err == sql.ErrNoRows {
		// A verified transfer can arrive before the creation response is saved.
		// Ownership comes from the stored unique reference, never webhook input.
		err = tx.QueryRowContext(ctx, `SELECT w.id FROM wallet_virtual_accounts a JOIN wallets w ON w.owner_type='user' AND w.owner_id=a.user_id AND w.currency=a.currency WHERE a.tx_ref=$1 AND a.provider='flutterwave' AND a.status IN ('active','pending') AND a.currency=$2`, reference, currency).Scan(&walletID)
		if err != nil {
			return false, fmt.Errorf("wallet funding reference was not found: %w", err)
		}
		reference = "VA-" + providerID
		expectedWalletID := walletID
		_, err = tx.ExecContext(ctx, `INSERT INTO wallet_deposits(wallet_id,provider,reference,amount,currency) VALUES($1,'flutterwave',$2,$3::numeric,$4) ON CONFLICT(reference) DO NOTHING`, walletID, reference, amount, currency)
		if err != nil {
			return false, err
		}
		err = tx.QueryRowContext(ctx, `SELECT wallet_id,amount::text,currency,status,provider_transaction_id,fee_version FROM wallet_deposits WHERE reference=$1 FOR UPDATE`, reference).
			Scan(&walletID, &savedAmount, &savedCurrency, &current, &priorID, &feeVersion)
		if err == nil && walletID != expectedWalletID {
			return false, errors.New("provider transaction belongs to a different wallet")
		}
	}
	if err != nil {
		return false, err
	}
	var matches bool
	if err = tx.QueryRowContext(ctx, `SELECT $1::numeric=$2::numeric`, savedAmount, amount).Scan(&matches); err != nil {
		return false, err
	}
	if savedCurrency != currency || !matches {
		return false, fmt.Errorf("wallet payment amount or currency did not match")
	}
	if priorID.Valid && priorID.String != providerID {
		return false, fmt.Errorf("this funding request already has a payment")
	}
	if err = claimProviderReceipt(ctx, tx, providerID, reference); err != nil {
		return false, err
	}
	credited := false
	if current == "successful" && status == "reversed" {
		if err = reverseDepositFees(tx, walletID, reference, currency); err != nil {
			return false, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE wallet_deposits SET status='reversed',review_required=true,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference); err != nil {
			return false, err
		}
	}

	if current != "successful" && current != "reversed" {
		_, err = tx.ExecContext(ctx, `UPDATE wallet_deposits SET status=$2,provider_transaction_id=$3,verified_amount=$4::numeric,verified_currency=$5,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference, status, providerID, amount, currency)
		if err != nil {
			return false, err
		}
		if status == "successful" {
			gross, err := money.Positive(amount)
			if err != nil {
				return false, err
			}
			if feeVersion > 0 && currency == "NGN" {
				costs := DepositCosts{}
				if len(providerCosts) > 0 {
					costs = providerCosts[0]
				}
				if err = applyDepositFees(tx, walletID, reference, gross, currency, costs); err != nil {
					return false, err
				}
				if err = tx.QueryRowContext(ctx, `SELECT wallet_credit_kobo>0 FROM wallet_fee_records WHERE reference=$1`, reference).Scan(&credited); err != nil {
					return false, err
				}
			} else {
				if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
					return false, err
				}
				if err = postJournal(tx, reference+":receipt", "legacy_wallet_funding", currency, 0, "{}", debit("provider_clearing", gross), credit(walletAccount(walletID), gross)); err != nil {
					return false, err
				}
				if err = walletEntry(tx, walletID, "deposit", gross, reference); err != nil {
					return false, err
				}
				if err = creditWalletKobo(tx, walletID, gross, reference); err != nil {
					return false, err
				}
				credited = true
			}
		}

	}
	// Pending verification must not consume the provider event forever. A later
	// delivery of the same charge.completed event can carry the final result.
	if event != nil && status != "pending" && status != "processing" {
		if err = recordWebhookEvent(tx, event); err != nil {
			return false, err
		}
	}
	return credited, tx.Commit()
}

func (r *WalletRepository) PayOrder(ctx context.Context, buyerID, orderID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	var status, amount, currency string
	err = tx.QueryRowContext(ctx, `SELECT status,total_amount::text,currency FROM orders WHERE id=$1 AND buyer_id=$2 FOR UPDATE`, orderID, buyerID).Scan(&status, &amount, &currency)
	if err != nil {
		return err
	}
	if status != "pending" {
		var paid bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE order_id=$1 AND buyer_id=$2 AND payment_provider='wallet' AND status='successful')`, orderID, buyerID).Scan(&paid); err != nil {
			return err
		}
		if paid {
			return nil
		}
		return fmt.Errorf("this order is no longer waiting for payment")
	}
	var pending bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE order_id=$1 AND status IN ('pending','successful'))`, orderID).Scan(&pending); err != nil {
		return err
	}
	if pending {
		return fmt.Errorf("check the payment already started for this order before using your wallet")
	}
	var sellerType string
	var sellerID, sellerCount int
	err = tx.QueryRowContext(ctx, `SELECT min(p.owner_type),min(p.owner_id),count(DISTINCT(p.owner_type,p.owner_id)) FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE oi.order_id=$1`, orderID).Scan(&sellerType, &sellerID, &sellerCount)
	if err != nil {
		return err
	}
	if sellerCount != 1 {
		return fmt.Errorf("order must have one seller")
	}
	// Lock all participating wallets in ID order, including cross-purchases.
	_, err = tx.ExecContext(ctx, `INSERT INTO wallets(owner_type,owner_id,currency) VALUES('user',$1,$2),($3,$4,$2) ON CONFLICT(owner_type,owner_id,currency) DO NOTHING`, buyerID, currency, sellerType, sellerID)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM wallets WHERE currency=$1 AND ((owner_type='user' AND owner_id=$2) OR(owner_type=$3 AND owner_id=$4)) ORDER BY id FOR UPDATE`, currency, buyerID, sellerType, sellerID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var walletID int64
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric AND recovery_debt_kobo=0 AND balance*100=COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||wallets.id AND p.currency=wallets.currency),0) RETURNING id`, buyerID, currency, amount).Scan(&walletID)
	if err == sql.ErrNoRows {
		return ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	reference := fmt.Sprintf("wallet-order-%d", orderID)
	_, err = tx.ExecContext(ctx, `INSERT INTO transactions(order_id,buyer_id,seller_type,seller_id,amount,currency,payment_method,payment_reference,payment_provider,status,paid_at) VALUES($1,$2,$3,$4,$5::numeric,$6,'wallet',$7,'wallet','successful',CURRENT_TIMESTAMP)`, orderID, buyerID, sellerType, sellerID, amount, currency, reference)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference,reference_type,reference_id) VALUES($1,'order_payment',$2::numeric,$3,'order',$4)`, walletID, amount, reference, orderID)
	if err != nil {
		return err
	}
	if err = holdOrderSale(tx, orderID, sellerType, sellerID, amount, currency, r.commissionBPS); err != nil {
		return err
	}
	if err = confirmOrderAfterPayment(tx, orderID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WalletRepository) RequestTransfer(ctx context.Context, senderID int, recipientNumber, amount, reference, note string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	if err = lockWalletRequest(ctx, tx, reference); err != nil {
		return err
	}
	if found, err := transferMatches(ctx, tx, senderID, recipientNumber, amount, reference, note, r.currency); err != nil || found {
		return err
	}
	var recipientID int
	var recipientWallet int64
	var recipientPhone string
	err = tx.QueryRowContext(ctx, `SELECT w.id,w.owner_id,u.phone FROM wallets w JOIN users u ON u.id=w.owner_id WHERE w.wallet_number=$1 AND w.currency=$2 AND w.owner_type='user' AND u.deleted_at IS NULL`, recipientNumber, r.currency).Scan(&recipientWallet, &recipientID, &recipientPhone)
	if err != nil {
		return fmt.Errorf("recipient wallet was not found")
	}
	if recipientID == senderID {
		return fmt.Errorf("choose another person's wallet")
	}
	var senderWallet int64
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric AND recovery_debt_kobo=0 AND balance*100=COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||wallets.id AND p.currency=wallets.currency),0) RETURNING id`, senderID, r.currency, amount).Scan(&senderWallet)
	if err == sql.ErrNoRows {
		return ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transfers(sender_wallet_id,sender_id,recipient_wallet_id,recipient_id,recipient_phone,reference,amount,currency,note) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9)`, senderWallet, senderID, recipientWallet, recipientID, recipientPhone, reference, amount, r.currency, note)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'transfer_out',$2::numeric,$3)`, senderWallet, amount, reference+":out")
	if err != nil {
		return err
	}
	principal, err := money.Positive(amount)
	if err != nil {
		return err
	}
	for _, account := range []string{walletAccount(senderWallet), "transfer:" + reference} {
		if err = ensureAccount(tx, account, "liability"); err != nil {
			return err
		}
	}
	if err = postJournal(tx, reference+":reserve", "wallet_transfer_reservation", r.currency, senderID, "{}", debit(walletAccount(senderWallet), principal), credit("transfer:"+reference, principal)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WalletRepository) DecideTransfer(ctx context.Context, actorID int, admin bool, id int64, decision string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	var senderID int
	var senderWallet, recipientWallet int64
	var status, amount, reference string
	err = tx.QueryRowContext(ctx, `SELECT sender_id,sender_wallet_id,recipient_wallet_id,status,amount::text,reference FROM wallet_transfers WHERE id=$1 FOR UPDATE`, id).Scan(&senderID, &senderWallet, &recipientWallet, &status, &amount, &reference)
	if err != nil {
		return err
	}
	if status != "pending" {
		return fmt.Errorf("this transfer has already been handled")
	}
	if !(admin && (decision == "completed" || decision == "rejected")) && !(senderID == actorID && decision == "cancelled") {
		return fmt.Errorf("you cannot make this transfer change")
	}
	walletID, kind, suffix := senderWallet, "transfer_refund", ":refund"
	if decision == "completed" {
		walletID, kind, suffix = recipientWallet, "transfer_in", ":in"
	}
	principal, parseErr := money.Positive(amount)
	if parseErr != nil {
		return parseErr
	}
	if err = ensureAccount(tx, walletAccount(walletID), "liability"); err != nil {
		return err
	}
	if err = postJournal(tx, reference+suffix, "wallet_transfer_"+decision, r.currency, actorID, "{}", debit("transfer:"+reference, principal), credit(walletAccount(walletID), principal)); err != nil {
		return err
	}
	err = creditWalletKobo(tx, walletID, principal, reference+suffix)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,$2,$3::numeric,$4)`, walletID, kind, amount, reference+suffix)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE wallet_transfers SET status=$2,approved_by=CASE WHEN $3 THEN $4 ELSE approved_by END,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, decision, admin, actorID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WalletRepository) Transfers(ctx context.Context, userID int) ([]models.Transfer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id,t.sender_id,s.full_name,t.recipient_id,u.full_name,t.recipient_phone,t.amount::text,t.currency,t.note,t.reference,t.status,t.created_at,t.updated_at FROM wallet_transfers t JOIN users s ON s.id=t.sender_id JOIN users u ON u.id=t.recipient_id WHERE ($1=0 OR sender_id=$1 OR recipient_id=$1) ORDER BY t.created_at DESC,t.id DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Transfer{}
	for rows.Next() {
		var x models.Transfer
		if err = rows.Scan(&x.ID, &x.SenderID, &x.SenderName, &x.RecipientID, &x.RecipientName, &x.RecipientPhone, &x.Amount, &x.Currency, &x.Note, &x.Reference, &x.Status, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (r *WalletRepository) RequestWithdrawal(ctx context.Context, userID int, amount, reference string, account *models.UserBankAccount, quote *models.FeeQuote, auth models.WithdrawalAuthorization) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	if account == nil {
		return errors.New("save your bank account first")
	}
	if err = lockWalletRequest(ctx, tx, reference); err != nil {
		return err
	}
	if found, err := withdrawalMatches(ctx, tx, userID, amount, reference, account.BankCode, account.AccountNumber, r.currency); err != nil || found {
		return err
	}
	principal, err := money.Positive(amount)
	if err != nil {
		return err
	}
	policy, err := feePolicy(ctx, tx)
	if err != nil {
		return err
	}
	service, err := policy.WithdrawalFee(principal)
	if err != nil {
		return err
	}
	q := &models.FeeQuote{AmountKobo: principal, ServiceKobo: service}
	if quote != nil {
		q = quote
		if q.ID != "" {
			q, err = scanFeeQuote(tx.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM wallet_fee_quotes WHERE id=$1 AND user_id=$2 AND kind='withdrawal' AND expires_at>CURRENT_TIMESTAMP`, q.ID, userID))
			if err != nil {
				return err
			}
		}
		if q.BankCode != account.BankCode || q.AccountNumber != account.AccountNumber {
			return errors.New("bank destination does not match quote")
		}
	} else {
		q.PolicyID = policy.ID
		q.WalletDebitKobo, err = money.Add(principal, service)
		if err != nil {
			return err
		}
	}
	expectedDebit, err := money.Add(principal, service, q.CustomerProviderKobo)
	if err != nil {
		return err
	}
	if q.AmountKobo != principal || q.ServiceKobo != service || q.WalletDebitKobo != expectedDebit {
		return errors.New("withdrawal quote does not match")
	}
	var evidence string
	if err = tx.QueryRowContext(ctx, `SELECT token_hash FROM withdrawal_evidence WHERE token_hash=$1 AND session_hash=$2 AND user_id=$3 AND reference=$4 AND quote_id=$5 AND expires_at>CURRENT_TIMESTAMP AND photo_id IS NOT NULL AND withdrawal_id IS NULL FOR UPDATE`, auth.TokenHash, auth.SessionHash, userID, reference, q.ID).Scan(&evidence); err != nil {
		return ErrPrivateEvidence
	}
	var walletID int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency=$2 FOR UPDATE`, userID, r.currency).Scan(&walletID); err != nil {
		return err
	}
	var outstanding bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_withdrawals WHERE user_id=$1 AND status IN ('pending','processing'))`, userID).Scan(&outstanding); err != nil {
		return err
	}
	if outstanding {
		return errors.New("a withdrawal is already pending or processing; check its status before starting another")
	}
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric AND recovery_debt_kobo=0 AND balance*100=COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||wallets.id AND p.currency=wallets.currency),0) RETURNING id`, userID, r.currency, money.Format(q.WalletDebitKobo)).Scan(&walletID)
	if err == sql.ErrNoRows {
		return ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	var withdrawalID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO wallet_withdrawals(wallet_id,user_id,reference,amount,currency,bank_name,bank_code,account_name,account_number,name_check_status,name_check_resolved,name_check_at)
		VALUES($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9,'verified',$8,CURRENT_TIMESTAMP) RETURNING id`, walletID, userID, reference, amount, r.currency, account.BankName, account.BankCode, account.AccountName, account.AccountNumber).Scan(&withdrawalID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE withdrawal_evidence SET withdrawal_id=$2 WHERE token_hash=$1`, evidence, withdrawalID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'withdrawal_reserved',$2::numeric,$3)`, walletID, amount, reference+":reserved")
	if err != nil {
		return err
	}
	if err = reserveWithdrawalFees(tx, walletID, reference, r.currency, principal, q); err != nil {
		return err
	}
	return tx.Commit()
}

const withdrawalColumns = `w.id,w.user_id,u.full_name,w.amount::text,w.currency,w.bank_name,w.bank_code,w.account_name,w.account_number,w.reference,w.provider_transfer_id,w.status,w.name_check_status,w.created_at,w.updated_at`

func scanWithdrawal(row interface{ Scan(...any) error }) (*models.Withdrawal, error) {
	x := &models.Withdrawal{}
	err := row.Scan(&x.ID, &x.UserID, &x.FullName, &x.Amount, &x.Currency, &x.BankName, &x.BankCode, &x.AccountName, &x.AccountNumber, &x.Reference, &x.TransferID, &x.Status, &x.NameCheckStatus, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

func (r *WalletRepository) Withdrawal(ctx context.Context, id int64) (*models.Withdrawal, error) {
	return scanWithdrawal(r.db.QueryRowContext(ctx, `SELECT `+withdrawalColumns+` FROM wallet_withdrawals w JOIN users u ON u.id=w.user_id WHERE w.id=$1`, id))
}

func (r *WalletRepository) WithdrawalByReference(ctx context.Context, reference string) (*models.Withdrawal, error) {
	return scanWithdrawal(r.db.QueryRowContext(ctx, `SELECT `+withdrawalColumns+` FROM wallet_withdrawals w JOIN users u ON u.id=w.user_id WHERE w.reference=$1`, reference))
}

func (r *WalletRepository) Withdrawals(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+withdrawalColumns+` FROM wallet_withdrawals w JOIN users u ON u.id=w.user_id WHERE ($1=0 OR w.user_id=$1) ORDER BY w.created_at DESC,w.id DESC LIMIT 200`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Withdrawal{}
	for rows.Next() {
		x, err := scanWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *x)
	}
	return items, rows.Err()
}

func (r *WalletRepository) ClaimWithdrawal(ctx context.Context, id int64, adminID int, sessionHash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = financeActor(ctx, tx, adminID, "finance", r.users); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE wallet_withdrawals w SET status='processing',provider_submission_state='unknown',approved_by=$2,updated_at=CURRENT_TIMESTAMP WHERE w.id=$1 AND w.status='pending' AND w.provider_submission_state='not_submitted' AND $2>0 AND w.user_id<>$2
      AND EXISTS(SELECT 1 FROM withdrawal_evidence e JOIN withdrawal_evidence_reviews v ON v.withdrawal_id=e.withdrawal_id
        WHERE e.withdrawal_id=w.id AND e.photo_id IS NOT NULL AND v.reviewer_id=$2 AND v.session_hash=$3
        AND v.reviewed_at>CURRENT_TIMESTAMP-interval '15 minutes' AND v.photo_viewed_at>=v.reviewed_at
        AND COALESCE(v.registration_photo_id,0)=COALESCE((SELECT p.id FROM private_verification_photos p WHERE p.user_id=w.user_id AND p.purpose IN ('registration','legacy') ORDER BY p.created_at DESC,p.id DESC LIMIT 1),0)
        AND (v.registration_photo_id IS NULL OR v.registration_photo_viewed_at>=v.reviewed_at))
      AND EXISTS(SELECT 1 FROM user_sessions s JOIN admin_mfa m ON m.user_id=s.user_id JOIN users u ON u.id=s.user_id WHERE s.user_id=$2 AND s.token_hash=$3 AND s.expires_at>CURRENT_TIMESTAMP AND s.mfa_verified_until>CURRENT_TIMESTAMP AND m.enabled AND u.account_status='active' AND NOT u.staff_disabled AND u.deleted_at IS NULL)`, id, adminID, sessionHash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("this withdrawal is no longer pending or its private evidence review needs to be refreshed")
	}
	return tx.Commit()
}

func (r *WalletRepository) SaveTransferID(ctx context.Context, id int64, providerID string) error {
	if providerID == "" {
		return errors.New("provider transfer ID is missing")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	var reference string
	if err = tx.QueryRowContext(ctx, `SELECT reference FROM wallet_withdrawals WHERE id=$1 FOR UPDATE`, id).Scan(&reference); err != nil {
		return err
	}
	if err = claimProviderTransfer(tx, "flutterwave", providerID, reference); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE wallet_withdrawals SET provider_transfer_id=$2,provider_submission_state='accepted',updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND status IN ('processing','paid','failed','reversed') AND (provider_transfer_id IS NULL OR provider_transfer_id=$2)`, id, providerID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("the provider payout does not match an active withdrawal")
	}
	return tx.Commit()
}

// FinishWithdrawal either records a verified payout or returns reserved money.
// A processing request can be refunded only after the provider verifies failure.
func (r *WalletRepository) FinishWithdrawal(ctx context.Context, id int64, actorID int, admin bool, decision string, providerVerified bool, providerActual ...*int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return err
	}
	var userID int
	var walletID int64
	var status, reference, amount string
	err = tx.QueryRowContext(ctx, `SELECT user_id,wallet_id,status,reference,amount::text FROM wallet_withdrawals WHERE id=$1 FOR UPDATE`, id).Scan(&userID, &walletID, &status, &reference, &amount)
	if err != nil {
		return err
	}
	if (status == decision || status == "reversed") && providerVerified {
		return nil
	}
	if status == "paid" && decision == "reversed" && providerVerified {
		if err = reverseCustomerWithdrawal(tx, reference, r.currency, walletID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE wallet_withdrawals SET status='reversed',updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id); err != nil {
			return err
		}
		return tx.Commit()
	}
	valid := (status == "pending" && ((admin && decision == "rejected") || (userID == actorID && decision == "cancelled"))) || (status == "processing" && providerVerified && (decision == "paid" || decision == "failed"))
	if !valid {
		return fmt.Errorf("this withdrawal cannot be changed now")
	}
	var actual *int64
	if len(providerActual) > 0 {
		actual = providerActual[0]
	}
	extraRefund, err := settleWithdrawalFees(tx, reference, decision, r.currency, actual)
	if err != nil {
		return err
	}
	if decision != "paid" {
		principal, err := money.Positive(amount)
		if err != nil {
			return err
		}
		refund, err := money.Add(principal, extraRefund)
		if err != nil {
			return err
		}
		if err = creditWalletKobo(tx, walletID, refund, reference+":refund"); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'withdrawal_refund',$2::numeric,$3)`, walletID, amount, reference+":refund")
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE wallet_withdrawals SET status=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, decision)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WalletRepository) VirtualAccount(ctx context.Context, userID int) (*models.WalletVirtualAccount, error) {
	a := &models.WalletVirtualAccount{}
	err := scanVirtualAccount(r.db.QueryRowContext(ctx, `SELECT `+virtualAccountColumns+` FROM wallet_virtual_accounts WHERE user_id=$1`, userID), a)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (r *WalletRepository) PaymentExceptions(ctx context.Context) ([]models.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,order_id,buyer_id,payment_reference,amount::text,currency,status,created_at FROM transactions WHERE review_required ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Payment{}
	for rows.Next() {
		var p models.Payment
		if err = rows.Scan(&p.ID, &p.OrderID, &p.BuyerID, &p.Reference, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
