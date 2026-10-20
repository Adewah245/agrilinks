package repository

import (
	"database/sql"
	"fmt"

	"backend/internal/models"
)

// PaymentRepository owns persistence for payment attempts and wallet reads.
type PaymentRepository struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) *PaymentRepository { return &PaymentRepository{db: db} }

// CreatePendingForOrder creates a payment attempt only for the order's buyer.
// Current orders contain one item, so its product owner is the transaction's
// seller. This avoids accepting seller or amount values from a browser form.
func (r *PaymentRepository) CreatePendingForOrder(buyerID, orderID int, reference string) (*models.Payment, error) {
	payment := &models.Payment{}
	err := r.db.QueryRow(`
		WITH payable AS (
			SELECT o.id, o.buyer_id, p.owner_type, p.owner_id, o.total_amount
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN products p ON p.id = oi.product_id
			WHERE o.id = $1 AND o.buyer_id = $2 AND o.status = 'confirmed'
			ORDER BY oi.id LIMIT 1
		)
		INSERT INTO transactions (order_id, buyer_id, seller_type, seller_id, amount, currency, payment_method, payment_reference, payment_provider, status)
		SELECT id, buyer_id, owner_type, owner_id, total_amount, 'NGN', 'flutterwave', $3, 'flutterwave', 'pending'
		FROM payable
		RETURNING id, order_id, payment_reference, provider_transaction_id,
			amount::text, currency, status, payment_provider, created_at, updated_at`,
		orderID, buyerID, reference,
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
		return nil, fmt.Errorf("confirmed order not found")
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

func (r *PaymentRepository) GetWallet(ownerType string, ownerID int) (*models.Wallet, error) {
	w := &models.Wallet{}
	err := r.db.QueryRow(
		`SELECT id, owner_id, balance::text, currency, wallet_number, created_at, updated_at
		 FROM wallets
		 WHERE owner_type=$1 AND owner_id=$2 AND currency='NGN'`,
		ownerType,
		ownerID,
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
func (r *PaymentRepository) GetPaymentProfile(userID int) (*models.User, error) {
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
			AND w.currency = 'NGN'
		WHERE u.id = $1
	`, userID).Scan(
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
