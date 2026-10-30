package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/models"
)

var ErrInsufficientBalance = errors.New("your wallet does not have enough money")

type WalletRepository struct {
	db            *sql.DB
	currency      string
	commissionBPS int64
}

func NewWalletRepository(db *sql.DB, currency string, commissionBPS int64) *WalletRepository {
	return &WalletRepository{db: db, currency: currency, commissionBPS: commissionBPS}
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
	_, err = r.db.ExecContext(ctx, `INSERT INTO wallet_deposits(wallet_id,provider,reference,amount,currency) VALUES($1,'flutterwave',$2,$3::numeric,$4) ON CONFLICT(reference) DO NOTHING`, w.ID, reference, amount, r.currency)
	if err != nil {
		return nil, err
	}
	d := &models.WalletDeposit{}
	err = r.db.QueryRowContext(ctx, `SELECT id,provider,amount::text,currency,reference,status,checkout_url,created_at FROM wallet_deposits WHERE reference=$1 AND wallet_id=$2 AND amount=$3::numeric`, reference, w.ID, amount).
		Scan(&d.ID, &d.Provider, &d.Amount, &d.Currency, &d.Reference, &d.Status, &d.CheckoutURL, &d.CreatedAt)
	return d, err
}

func (r *WalletRepository) SaveDepositURL(ctx context.Context, reference, checkoutURL string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE wallet_deposits SET checkout_url=$2,updated_at=CURRENT_TIMESTAMP WHERE reference=$1 AND status='pending'`, reference, checkoutURL)
	return err
}

// FinalizeDeposit also handles transfers into a previously saved permanent
// account. The provider's transaction ID identifies each distinct bank transfer.
func (r *WalletRepository) FinalizeDeposit(ctx context.Context, reference, providerID, amount, currency, status string, event *models.WebhookEvent) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var walletID int64
	var savedAmount, savedCurrency, current string
	var priorID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT wallet_id,amount::text,currency,status,provider_transaction_id FROM wallet_deposits WHERE reference=$1 AND provider='flutterwave' FOR UPDATE`, reference).
		Scan(&walletID, &savedAmount, &savedCurrency, &current, &priorID)
	if err == sql.ErrNoRows && status == "successful" {
		err = tx.QueryRowContext(ctx, `SELECT w.id FROM wallet_virtual_accounts a JOIN wallets w ON w.owner_type='user' AND w.owner_id=a.user_id AND w.currency=a.currency WHERE a.tx_ref=$1 AND a.provider='flutterwave' AND a.status='active' AND a.currency=$2`, reference, currency).Scan(&walletID)
		if err != nil {
			return false, fmt.Errorf("wallet funding reference was not found: %w", err)
		}
		reference = "VA-" + providerID
		_, err = tx.ExecContext(ctx, `INSERT INTO wallet_deposits(wallet_id,provider,reference,amount,currency) VALUES($1,'flutterwave',$2,$3::numeric,$4) ON CONFLICT(reference) DO NOTHING`, walletID, reference, amount, currency)
		if err != nil {
			return false, err
		}
		err = tx.QueryRowContext(ctx, `SELECT wallet_id,amount::text,currency,status,provider_transaction_id FROM wallet_deposits WHERE reference=$1 FOR UPDATE`, reference).
			Scan(&walletID, &savedAmount, &savedCurrency, &current, &priorID)
	}
	if err != nil {
		return false, err
	}
	var matches bool
	if err = tx.QueryRowContext(ctx, `SELECT $1::numeric=$2::numeric`, savedAmount, amount).Scan(&matches); err != nil {
		return false, err
	}
	if savedCurrency != currency || (status == "successful" && !matches) {
		return false, fmt.Errorf("wallet payment amount or currency did not match")
	}
	if current == "successful" && priorID.Valid && priorID.String != providerID {
		return false, fmt.Errorf("this funding request already has a payment")
	}
	credited := false
	if current != "successful" && current != "reversed" {
		_, err = tx.ExecContext(ctx, `UPDATE wallet_deposits SET status=$2,provider_transaction_id=$3,verified_amount=$4::numeric,verified_currency=$5,updated_at=CURRENT_TIMESTAMP WHERE reference=$1`, reference, status, providerID, amount, currency)
		if err != nil {
			return false, err
		}
		if status == "successful" {
			result, err := tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference,description) VALUES($1,'deposit',$2::numeric,$3,'Wallet funding confirmed by the payment provider') ON CONFLICT(reference) DO NOTHING`, walletID, amount, reference)
			if err != nil {
				return false, err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return false, err
			}
			if n == 1 {
				if _, err = tx.ExecContext(ctx, `UPDATE wallets SET balance=balance+$2::numeric,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, amount); err != nil {
					return false, err
				}
				credited = true
			}
		}
	}
	if event != nil {
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
	var status, amount, currency string
	err = tx.QueryRowContext(ctx, `SELECT status,total_amount::text,currency FROM orders WHERE id=$1 AND buyer_id=$2 FOR UPDATE`, orderID, buyerID).Scan(&status, &amount, &currency)
	if err != nil {
		return err
	}
	if status != "pending" {
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
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric RETURNING id`, buyerID, currency, amount).Scan(&walletID)
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
	if err = creditOrderSale(tx, orderID, sellerType, sellerID, amount, currency, r.commissionBPS); err != nil {
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
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric RETURNING id`, senderID, r.currency, amount).Scan(&senderWallet)
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
	return tx.Commit()
}

func (r *WalletRepository) DecideTransfer(ctx context.Context, actorID int, admin bool, id int64, decision string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
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
	_, err = tx.ExecContext(ctx, `UPDATE wallets SET balance=balance+$2::numeric,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, amount)
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

func (r *WalletRepository) RequestWithdrawal(ctx context.Context, userID int, amount, reference string, account *models.UserBankAccount) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var walletID int64
	err = tx.QueryRowContext(ctx, `UPDATE wallets SET balance=balance-$3::numeric,updated_at=CURRENT_TIMESTAMP WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND balance>=$3::numeric RETURNING id`, userID, r.currency, amount).Scan(&walletID)
	if err == sql.ErrNoRows {
		return ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_withdrawals(wallet_id,user_id,reference,amount,currency,bank_name,bank_code,account_name,account_number,name_check_status,name_check_resolved,name_check_at)
		VALUES($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9,'verified',$8,CURRENT_TIMESTAMP)`, walletID, userID, reference, amount, r.currency, account.BankName, account.BankCode, account.AccountName, account.AccountNumber)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'withdrawal_reserved',$2::numeric,$3)`, walletID, amount, reference+":reserved")
	if err != nil {
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

func (r *WalletRepository) ClaimWithdrawal(ctx context.Context, id int64, adminID int) error {
	result, err := r.db.ExecContext(ctx, `UPDATE wallet_withdrawals SET status='processing',approved_by=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND status='pending'`, id, adminID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("this withdrawal is already being handled")
	}
	return nil
}

func (r *WalletRepository) SaveTransferID(ctx context.Context, id int64, providerID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE wallet_withdrawals SET provider_transfer_id=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND status='processing'`, id, providerID)
	return err
}

// FinishWithdrawal either records a verified payout or returns reserved money.
// A processing request can be refunded only after the provider verifies failure.
func (r *WalletRepository) FinishWithdrawal(ctx context.Context, id int64, actorID int, admin bool, decision string, providerVerified bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userID int
	var walletID int64
	var status, reference, amount string
	err = tx.QueryRowContext(ctx, `SELECT user_id,wallet_id,status,reference,amount::text FROM wallet_withdrawals WHERE id=$1 FOR UPDATE`, id).Scan(&userID, &walletID, &status, &reference, &amount)
	if err != nil {
		return err
	}
	if status == decision && providerVerified {
		return nil
	}
	valid := (status == "pending" && ((admin && decision == "rejected") || (userID == actorID && decision == "cancelled"))) || (status == "processing" && providerVerified && (decision == "paid" || decision == "failed"))
	if !valid {
		return fmt.Errorf("this withdrawal cannot be changed now")
	}
	if decision != "paid" {
		_, err = tx.ExecContext(ctx, `UPDATE wallets SET balance=balance+$2::numeric,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, walletID, amount)
		if err != nil {
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
