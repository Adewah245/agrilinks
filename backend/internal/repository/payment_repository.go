package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"backend/internal/models"
)

// PaymentRepository owns persistence for payment attempts and wallet reads.
type PaymentRepository struct {
	db            *sql.DB
	commissionBPS int64
}

func NewPaymentRepository(db *sql.DB, commissionBPS ...int64) *PaymentRepository {
	r := &PaymentRepository{db: db}
	if len(commissionBPS) > 0 {
		r.commissionBPS = commissionBPS[0]
	}
	return r
}

// CreatePendingForOrder reuses an unfinished payment attempt under an order lock.
func (r *PaymentRepository) CreatePendingForOrder(buyerID, orderID int, reference, currency, provider string) (*models.Payment, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var status, amount, orderCurrency string
	err = tx.QueryRow(`SELECT status,total_amount::text,currency FROM orders WHERE id=$1 AND buyer_id=$2 FOR UPDATE`, orderID, buyerID).Scan(&status, &amount, &orderCurrency)
	if err != nil {
		return nil, err
	}
	if status != "pending" {
		return nil, fmt.Errorf("order is not waiting for payment")
	}
	var existing string
	err = tx.QueryRow(`SELECT payment_reference FROM transactions WHERE order_id=$1 AND status='pending' ORDER BY id LIMIT 1`, orderID).Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		reference = existing
	} else {
		var sellerType string
		var sellerID, sellers int
		err = tx.QueryRow(`SELECT min(p.owner_type),min(p.owner_id),count(DISTINCT (p.owner_type,p.owner_id)) FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE oi.order_id=$1`, orderID).Scan(&sellerType, &sellerID, &sellers)
		if err != nil {
			return nil, err
		}
		if sellers != 1 {
			return nil, fmt.Errorf("an order must have one seller")
		}
		_, err = tx.Exec(`INSERT INTO transactions(order_id,buyer_id,seller_type,seller_id,amount,currency,payment_method,payment_reference,payment_provider,status) VALUES($1,$2,$3,$4,$5::numeric,$6,$7,$8,$7,'pending')`, orderID, buyerID, sellerType, sellerID, amount, orderCurrency, provider, reference)
		if err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByReference(reference)
}

func (r *PaymentRepository) SaveCheckoutURL(reference, checkoutURL string) error {
	result, err := r.db.Exec(`UPDATE transactions SET checkout_url=$2,updated_at=CURRENT_TIMESTAMP WHERE payment_reference=$1 AND status='pending'`, reference, checkoutURL)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("payment is no longer waiting")
	}
	return nil
}

func (r *PaymentRepository) ListByBuyer(buyerID int) ([]models.Payment, error) {
	rows, err := r.db.Query(`
		SELECT t.id,t.order_id,t.buyer_id,COALESCE(t.payment_reference,''),t.provider_transaction_id,
			t.amount::text,t.currency,t.status,COALESCE(t.payment_provider,''),t.created_at,t.updated_at,
			COALESCE((SELECT string_agg(p.name,', ' ORDER BY oi.id) FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE oi.order_id=t.order_id),''),COALESCE(u.full_name,o.name,''),t.review_required
		FROM transactions t
		LEFT JOIN users u ON t.seller_type='user' AND u.id=t.seller_id
		LEFT JOIN organizations o ON t.seller_type='organization' AND o.id=t.seller_id
		WHERE t.buyer_id=$1 ORDER BY t.created_at DESC`, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()
	var result []models.Payment
	for rows.Next() {
		var payment models.Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.BuyerID,
			&payment.Reference,
			&payment.TransactionID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
			&payment.Provider,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.ProductName,
			&payment.SellerName,
			&payment.ReviewRequired,
		); err != nil {
			return nil, err
		}
		result = append(result, payment)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) GetWallet(ownerType string, ownerID int, currency string) (*models.Wallet, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return nil, fmt.Errorf("wallet currency is required")
	}

	w := &models.Wallet{}
	err := r.db.QueryRow(
		`SELECT id, owner_id, balance::text, currency, wallet_number, created_at, updated_at
		 FROM wallets
		 WHERE owner_type=$1 AND owner_id=$2 AND currency=$3`,
		ownerType,
		ownerID,
		currency,
	).Scan(
		&w.ID,
		&w.UserID,
		&w.Balance,
		&w.Currency,
		&w.WalletNumber,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// GetPaymentProfile loads the private bank-account and wallet-number data
// required for transfers. Ordinary profile queries deliberately omit it.
func (r *PaymentRepository) GetPaymentProfile(userID int, currency string) (*models.User, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return nil, fmt.Errorf("wallet currency is required")
	}

	user := &models.User{}

	err := r.db.QueryRow(`
		SELECT
			u.id,
			u.full_name,
			COALESCE(b.bank_name, ''),
			COALESCE(b.bank_code, ''),
			COALESCE(b.account_name, ''),
			COALESCE(b.account_number, ''),
			COALESCE(w.wallet_number, '')
		FROM users u
		LEFT JOIN user_bank_accounts b
			ON b.user_id = u.id
		LEFT JOIN wallets w
			ON w.owner_type = 'user'
			AND w.owner_id = u.id
			AND w.currency = $2
		WHERE u.id = $1
	`, userID, currency).Scan(
		&user.ID,
		&user.FullName,
		&user.BankName,
		&user.BankCode,
		&user.AccountName,
		&user.AccountNumber,
		&user.WalletNumber,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpsertBankAccount creates or replaces a user's payout account.
func (r *PaymentRepository) UpsertBankAccount(account *models.UserBankAccount) error {
	_, err := r.db.Exec(`
		INSERT INTO user_bank_accounts (
			user_id,
			bank_name,
			bank_code,
			account_name,
			account_number
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			bank_name = EXCLUDED.bank_name,
			bank_code = EXCLUDED.bank_code,
			account_name = EXCLUDED.account_name,
			account_number = EXCLUDED.account_number,
			updated_at = CURRENT_TIMESTAMP
	`,
		account.UserID,
		account.BankName,
		account.BankCode,
		account.AccountName,
		account.AccountNumber,
	)
	if err != nil {
		return fmt.Errorf("save bank account: %w", err)
	}

	return nil
}

// GetByReference returns the payment attempt identified by a provider reference.
func (r *PaymentRepository) GetByReference(reference string) (*models.Payment, error) {
	if reference == "" {
		return nil, fmt.Errorf("payment reference is required")
	}

	payment := &models.Payment{}
	err := r.db.QueryRow(`
		SELECT id, order_id, buyer_id, COALESCE(payment_reference, ''),
		       provider_transaction_id, amount::text, currency, status,
		       COALESCE(payment_provider, ''), checkout_url, review_required, created_at, updated_at
		FROM transactions
		WHERE payment_reference = $1
	`, reference).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.BuyerID,
		&payment.Reference,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Provider,
		&payment.CheckoutURL,
		&payment.ReviewRequired,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	return payment, nil
}

// UpdateStatus records a verified provider result for one payment.
func (r *PaymentRepository) UpdateStatus(
	reference,
	status,
	providerTransactionID string,
) error {
	if reference == "" {
		return fmt.Errorf("payment reference is required")
	}

	result, err := r.db.Exec(`
		UPDATE transactions
		SET status = CASE
		        WHEN status = 'successful' THEN status
		        ELSE $1
		    END,
		    provider_transaction_id = CASE
		        WHEN $1 = 'successful' THEN NULLIF($2, '')
		        ELSE provider_transaction_id
		    END,
		    paid_at = CASE
		        WHEN $1 = 'successful' THEN COALESCE(paid_at, CURRENT_TIMESTAMP)
		        ELSE paid_at
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE payment_reference = $3
	`, status, providerTransactionID, reference)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("payment not found")
	}

	return nil
}

// FinalizePayment applies one verified provider result. The payment update,
// order confirmation, and optional webhook record share one database
// transaction, so a crash cannot leave a paid order awaiting payment or a
// processed webhook without its payment change.
func (r *PaymentRepository) FinalizePayment(
	reference,
	status,
	providerTransactionID string,
	webhook *models.WebhookEvent,
) (bool, error) {
	reference = strings.TrimSpace(reference)
	providerTransactionID = strings.TrimSpace(providerTransactionID)
	if reference == "" || providerTransactionID == "" {
		return false, fmt.Errorf("payment reference and provider transaction ID are required")
	}

	switch status {
	case models.PaymentPending, models.PaymentSuccessful, models.PaymentFailed:
	default:
		return false, fmt.Errorf("unsupported payment status %q", status)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin payment finalization: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var orderID int
	var orderStatus string
	// Every attempt for this order takes the same lock before touching money.
	err = tx.QueryRow(`SELECT o.id,o.status FROM orders o JOIN transactions t ON t.order_id=o.id WHERE t.payment_reference=$1 FOR UPDATE OF o`, reference).Scan(&orderID, &orderStatus)
	if err != nil {
		return false, err
	}
	var currentStatus, sellerType, amount, currency string
	var sellerID int
	var oldProviderID sql.NullString
	err = tx.QueryRow(`SELECT status,seller_type,seller_id,amount::text,currency,provider_transaction_id FROM transactions WHERE payment_reference=$1 FOR UPDATE`, reference).
		Scan(&currentStatus, &sellerType, &sellerID, &amount, &currency, &oldProviderID)
	if err != nil {
		return false, err
	}
	if currentStatus == models.PaymentSuccessful && oldProviderID.Valid && oldProviderID.String != providerTransactionID {
		return false, fmt.Errorf("a different provider transaction already paid this reference")
	}
	newlySuccessful := currentStatus != models.PaymentSuccessful && status == models.PaymentSuccessful
	if currentStatus == models.PaymentSuccessful {
		status = models.PaymentSuccessful
	}
	var alreadyPaid bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM transactions WHERE order_id=$1 AND status='successful' AND payment_reference<>$2)`, orderID, reference).Scan(&alreadyPaid)
	if err != nil {
		return false, err
	}
	reviewRequired := newlySuccessful && (alreadyPaid || orderStatus != "pending")
	_, err = tx.Exec(`UPDATE transactions SET status=$1,provider_transaction_id=$2,
        paid_at=CASE WHEN $1='successful' THEN COALESCE(paid_at,CURRENT_TIMESTAMP) ELSE paid_at END,
        review_required=review_required OR $4,updated_at=CURRENT_TIMESTAMP WHERE payment_reference=$3`, status, providerTransactionID, reference, reviewRequired)
	if err != nil {
		return false, err
	}
	if newlySuccessful && !reviewRequired {
		if err = confirmOrderAfterPayment(tx, orderID); err != nil {
			return false, err
		}
		if err = creditOrderSale(tx, orderID, sellerType, sellerID, amount, currency, r.commissionBPS); err != nil {
			return false, err
		}
	}
	newlySuccessful = newlySuccessful && !reviewRequired

	if webhook != nil {
		if err := recordWebhookEvent(tx, webhook); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit payment finalization: %w", err)
	}
	return newlySuccessful, nil
}

func confirmOrderAfterPayment(tx *sql.Tx, orderID int) error {
	result, err := tx.Exec(`
		UPDATE orders
		SET status = 'confirmed', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'pending'
	`, orderID)
	if err != nil {
		return fmt.Errorf("confirm paid order: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}

	var status string
	err = tx.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("order not found")
	}
	if err != nil {
		return fmt.Errorf("read order status: %w", err)
	}

	switch status {
	case "confirmed", "processing", "ready_for_delivery", "shipped", "delivered":
		return nil
	case "cancelled":
		return fmt.Errorf("order was cancelled before payment completed")
	default:
		return fmt.Errorf("order is not awaiting payment")
	}
}

func recordWebhookEvent(tx *sql.Tx, event *models.WebhookEvent) error {
	provider := strings.TrimSpace(event.Provider)
	transactionID := strings.TrimSpace(event.TransactionID)
	eventName := strings.TrimSpace(event.Event)
	if provider == "" || transactionID == "" || eventName == "" || len(event.Payload) == 0 {
		return fmt.Errorf("webhook event is incomplete")
	}

	_, err := tx.Exec(`
		INSERT INTO webhook_events (
			provider,
			transaction_id,
			event,
			payload,
			processed_at
		)
		VALUES ($1, $2, $3, $4::jsonb, CURRENT_TIMESTAMP)
		ON CONFLICT (provider, transaction_id, event)
		DO UPDATE SET
			payload = EXCLUDED.payload,
			processed_at = COALESCE(
				webhook_events.processed_at,
				CURRENT_TIMESTAMP
			)
	`, provider, transactionID, eventName, string(event.Payload))
	if err != nil {
		return fmt.Errorf("record webhook event: %w", err)
	}
	return nil
}

// HasWebhookBeenProcessed reports whether this provider event was completed.
func (r *PaymentRepository) HasWebhookBeenProcessed(
	transactionID,
	event string,
) (bool, error) {
	var processed bool
	err := r.db.QueryRow(`
		SELECT processed_at IS NOT NULL
		FROM webhook_events
		WHERE provider = 'flutterwave'
		  AND transaction_id = $1
		  AND event = $2
	`, transactionID, event).Scan(&processed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check webhook event: %w", err)
	}
	return processed, nil
}
