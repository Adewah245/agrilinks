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
func (r *PaymentRepository) CreatePendingForOrder(buyerID, orderID int, reference string) (*models.Transaction, error) {
	t := &models.Transaction{}
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
		SELECT id, buyer_id, owner_type, owner_id, total_amount, 'NGN', 'bmoni', $3, 'bmoni', 'pending'
		FROM payable
		RETURNING id, order_id, buyer_id, seller_type, seller_id, amount, currency,
			payment_method, payment_reference, payment_provider, status, paid_at, created_at, updated_at`,
		orderID, buyerID, reference,
	).Scan(&t.ID, &t.OrderID, &t.BuyerID, &t.SellerType, &t.SellerID, &t.Amount,
		&t.Currency, &t.PaymentMethod, &t.PaymentReference, &t.PaymentProvider,
		&t.Status, &t.PaidAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("confirmed order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	return t, nil
}

func (r *PaymentRepository) ListByBuyer(buyerID int) ([]models.Transaction, error) {
	rows, err := r.db.Query(`
		SELECT t.id,t.order_id,t.buyer_id,t.seller_type,t.seller_id,t.amount,t.currency,
			COALESCE(t.payment_method,''),COALESCE(t.payment_reference,''),COALESCE(t.payment_provider,''),t.status,t.paid_at,t.created_at,t.updated_at,
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
	var result []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.OrderID, &t.BuyerID, &t.SellerType, &t.SellerID, &t.Amount, &t.Currency, &t.PaymentMethod, &t.PaymentReference, &t.PaymentProvider, &t.Status, &t.PaidAt, &t.CreatedAt, &t.UpdatedAt, &t.ProductName, &t.SellerName); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) GetWallet(ownerType string, ownerID int) (*models.Wallet, error) {
	w := &models.Wallet{}
	err := r.db.QueryRow(`SELECT id,owner_type,owner_id,balance,currency,created_at,updated_at FROM wallets WHERE owner_type=$1 AND owner_id=$2 AND currency='NGN'`, ownerType, ownerID).Scan(&w.ID, &w.OwnerType, &w.OwnerID, &w.Balance, &w.Currency, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}
