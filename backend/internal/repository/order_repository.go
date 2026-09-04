package repository

import (
	"backend/internal/models"
	"database/sql"
)

// OrderRepository handles database operations for product orders.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new OrderRepository using the
// provided database connection.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create creates a new order for a product.
func (r *OrderRepository) Create(order *models.Order) error {
	query := `
		INSERT INTO orders (
			buyer_id,
			product_id,
			quantity,
			total_price,
			status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		order.BuyerID,
		order.ProductID,
		order.Quantity,
		order.TotalPrice,
		order.Status,
	).Scan(&order.ID)

	return err
}

// ListByBuyer returns all orders belonging to a buyer,
// with the product name and seller name resolved for display.
func (r *OrderRepository) ListByBuyer(
	buyerID int,
) ([]models.Order, error) {

	query := `
		SELECT
			orders.id,
			orders.buyer_id,
			orders.product_id,
			orders.quantity,
			orders.total_price,
			orders.status,
			orders.created_at,
			products.name,
			COALESCE(
				seller_user.full_name,
				seller_org.name,
				''
			) AS seller_name
		FROM orders
		JOIN products
			ON products.id = orders.product_id
		LEFT JOIN users seller_user
			ON products.owner_type = 'user'
			AND seller_user.id = products.owner_id
		LEFT JOIN organizations seller_org
			ON products.owner_type = 'organization'
			AND seller_org.id = products.owner_id
		WHERE orders.buyer_id = $1
		ORDER BY orders.created_at DESC, orders.id DESC
	`

	rows, err := r.db.Query(query, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order

	for rows.Next() {
		var o models.Order

		if err := rows.Scan(
			&o.ID,
			&o.BuyerID,
			&o.ProductID,
			&o.Quantity,
			&o.TotalPrice,
			&o.Status,
			&o.CreatedAt,
			&o.ProductName,
			&o.SellerName,
		); err != nil {
			return nil, err
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// ListSalesBySeller returns all orders for products owned by the
// specified seller, with the product name and buyer name resolved.
//
// sellerType must be either "user" or "organization".
func (r *OrderRepository) ListSalesBySeller(
	sellerType string,
	sellerID int,
) ([]models.Order, error) {

	query := `
		SELECT
			orders.id,
			orders.buyer_id,
			orders.product_id,
			orders.quantity,
			orders.total_price,
			orders.status,
			orders.created_at,
			products.name,
			buyer.full_name
		FROM orders
		JOIN products
			ON products.id = orders.product_id
		JOIN users buyer
			ON buyer.id = orders.buyer_id
		WHERE products.owner_type = $1
			AND products.owner_id = $2
		ORDER BY orders.created_at DESC, orders.id DESC
	`

	rows, err := r.db.Query(
		query,
		sellerType,
		sellerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order

	for rows.Next() {
		var o models.Order

		if err := rows.Scan(
			&o.ID,
			&o.BuyerID,
			&o.ProductID,
			&o.Quantity,
			&o.TotalPrice,
			&o.Status,
			&o.CreatedAt,
			&o.ProductName,
			&o.BuyerName,
		); err != nil {
			return nil, err
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
