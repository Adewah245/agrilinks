package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"backend/internal/models"
)

// PaymentRepository owns persistence for payment attempts and wallet reads.
type PaymentRepository struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) *PaymentRepository { return &PaymentRepository{db: db} }

// CreatePendingForOrder creates a payment attempt only for the order's buyer.
// Current orders contain one item, so its product owner is the transaction's
// seller. This avoids accepting seller or amount values from a browser form.
func (r *PaymentRepository) CreatePendingForOrder(
	buyerID,
	orderID int,
	reference,
	currency,
	provider string,
) (*models.Payment, error) {
	payment := &models.Payment{}
	err := r.db.QueryRow(`
		WITH payable AS (
			SELECT o.id, o.buyer_id, p.owner_type, p.owner_id, o.total_amount
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN products p ON p.id = oi.product_id
			WHERE o.id = $1 AND o.buyer_id = $2 AND o.status = 'pending'
			ORDER BY oi.id LIMIT 1
		)
		INSERT INTO transactions (order_id, buyer_id, seller_type, seller_id, amount, currency, payment_method, payment_reference, payment_provider, status)
		SELECT id, buyer_id, owner_type, owner_id, total_amount, $4, $5, $3, $5, 'pending'
		FROM payable
		RETURNING id, order_id, payment_reference, provider_transaction_id,
			amount::text, currency, status, payment_provider, created_at, updated_at`,
		orderID, buyerID, reference, currency, provider,
	).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.Reference,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.Provider,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pending order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	return payment, nil
}

func (r *PaymentRepository) ListByBuyer(buyerID int) ([]models.Payment, error) {
	rows, err := r.db.Query(`
		SELECT t.id,t.order_id,t.buyer_id,COALESCE(t.payment_reference,''),t.provider_transaction_id,
			t.amount::text,t.currency,t.status,COALESCE(t.payment_provider,''),t.created_at,t.updated_at,
			COALESCE(p.name,''),COALESCE(u.full_name,o.name,'')
		FROM transactions t
		LEFT JOIN order_items oi ON oi.order_id=t.order_id
		LEFT JOIN products p ON p.id=oi.product_id
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
		&w.FarmerID,
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
		       COALESCE(payment_provider, ''), created_at, updated_at
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

	var (
		orderID       int
		currentStatus string
	)
	err = tx.QueryRow(`
		SELECT order_id, status
		FROM transactions
		WHERE payment_reference = $1
		FOR UPDATE
	`, reference).Scan(&orderID, &currentStatus)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("payment not found")
	}
	if err != nil {
		return false, fmt.Errorf("lock payment: %w", err)
	}

	// A late failed-event delivery must never overwrite a successful payment.
	newlySuccessful := currentStatus != models.PaymentSuccessful && status == models.PaymentSuccessful
	if currentStatus == models.PaymentSuccessful {
		status = models.PaymentSuccessful
		newlySuccessful = false
	}

	_, err = tx.Exec(`
		UPDATE transactions
		SET status = $1,
		    provider_transaction_id = COALESCE(NULLIF($2, ''), provider_transaction_id),
		    paid_at = CASE
		        WHEN $1 = 'successful' THEN COALESCE(paid_at, CURRENT_TIMESTAMP)
		        ELSE paid_at
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE payment_reference = $3
	`, status, providerTransactionID, reference)
	if err != nil {
		return false, fmt.Errorf("finalize payment: %w", err)
	}

	if status == models.PaymentSuccessful {
		if err := confirmOrderAfterPayment(tx, orderID); err != nil {
			return false, err
		}
	}

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
