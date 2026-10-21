package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
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
	if order == nil || len(order.Items) == 0 {
		return sql.ErrNoRows
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = tx.QueryRow(`
		INSERT INTO orders (
			buyer_id,
			total_amount,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`,
		order.BuyerID,
		order.TotalAmount,
		order.Status,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range order.Items {
		item := &order.Items[i]
		err = tx.QueryRow(`
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`, order.ID, item.ProductID, item.Quantity, item.UnitPrice, item.TotalPrice).
			Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateFromCart creates an order from cart items in one transaction.
//
// The transaction locks each product, rechecks availability, creates the
// order and items, records inventory and listing changes, and removes the
// purchased cart items. A failure at any step rolls back the whole checkout.
func (r *OrderRepository) CreateFromCart(
	order *models.Order,
	cartItemIDs []int,
) error {
	if order == nil || len(order.Items) == 0 {
		return fmt.Errorf("order must contain at least one item")
	}

	if len(cartItemIDs) != len(order.Items) {
		return fmt.Errorf("cart items do not match order items")
	}

	if order.BuyerID <= 0 {
		return fmt.Errorf("invalid buyer")
	}

	order.TotalAmount = 0

	for i := range order.Items {
		item := &order.Items[i]
		if cartItemIDs[i] <= 0 {
			return fmt.Errorf("invalid cart item")
		}
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return fmt.Errorf("invalid order item")
		}
		if item.UnitPrice < 0 {
			return fmt.Errorf("order item price cannot be negative")
		}

		item.TotalPrice = item.Quantity * item.UnitPrice
		order.TotalAmount += item.TotalPrice
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin checkout: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = tx.QueryRow(`
		INSERT INTO orders (buyer_id, total_amount, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, order.BuyerID, order.TotalAmount, order.Status).Scan(
		&order.ID,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	for i := range order.Items {
		item := &order.Items[i]

		var (
			quantityBefore float64
			status         string
			ownerType      string
			ownerID        int
		)

		err = tx.QueryRow(`
			SELECT quantity, status, owner_type, owner_id
			FROM products
			WHERE id = $1
			FOR UPDATE
		`, item.ProductID).Scan(
			&quantityBefore,
			&status,
			&ownerType,
			&ownerID,
		)
		if err == sql.ErrNoRows {
			return fmt.Errorf("product %d no longer exists", item.ProductID)
		}
		if err != nil {
			return fmt.Errorf("lock product %d: %w", item.ProductID, err)
		}

		if status != "active" {
			return fmt.Errorf("product %d is no longer available", item.ProductID)
		}

		if ownerType == "user" && ownerID == order.BuyerID {
			return fmt.Errorf("you cannot buy your own product")
		}

		if item.Quantity > quantityBefore {
			return fmt.Errorf(
				"only %.6g of product %d is available",
				quantityBefore,
				item.ProductID,
			)
		}

		quantityAfter := quantityBefore - item.Quantity
		newStatus := status
		if quantityAfter <= 0 {
			newStatus = "sold_out"
		}

		err = tx.QueryRow(`
			INSERT INTO order_items (
				order_id,
				product_id,
				quantity,
				unit_price,
				total_price
			)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`,
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.TotalPrice,
		).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return fmt.Errorf("create order item: %w", err)
		}

		_, err = tx.Exec(`
			UPDATE products
			SET
				quantity = $1,
				status = $2,
				sold_out_at = CASE
					WHEN $2 = 'sold_out'
						THEN COALESCE(sold_out_at, CURRENT_TIMESTAMP)
					ELSE sold_out_at
				END,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`, quantityAfter, newStatus, item.ProductID)
		if err != nil {
			return fmt.Errorf("reduce product inventory: %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO inventory_history (
				product_id,
				quantity_before,
				quantity_change,
				quantity_after,
				reason
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
			item.ProductID,
			quantityBefore,
			-item.Quantity,
			quantityAfter,
			"order_placed",
		)
		if err != nil {
			return fmt.Errorf("record inventory change: %w", err)
		}

		if newStatus != status {
			_, err = tx.Exec(`
				INSERT INTO product_status_history (
					product_id,
					old_status,
					new_status,
					reason,
					changed_by_type,
					changed_by_id
				)
				VALUES ($1, $2, $3, $4, 'user', $5)
			`,
				item.ProductID,
				status,
				newStatus,
				"order_placed",
				order.BuyerID,
			)
			if err != nil {
				return fmt.Errorf("record product status change: %w", err)
			}
		}

		result, err := tx.Exec(`
			DELETE FROM cart_items
			WHERE id = $1 AND buyer_id = $2
		`, cartItemIDs[i], order.BuyerID)
		if err != nil {
			return fmt.Errorf("remove purchased cart item: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check cart item removal: %w", err)
		}
		if affected != 1 {
			return fmt.Errorf("cart item %d was not found", cartItemIDs[i])
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checkout: %w", err)
	}

	return nil
}

// GetForBuyer returns an order only when it belongs to the buyer.
func (r *OrderRepository) GetForBuyer(
	orderID,
	buyerID int,
) (*models.Order, error) {
	if orderID <= 0 || buyerID <= 0 {
		return nil, fmt.Errorf("invalid order request")
	}

	order := &models.Order{}
	err := r.db.QueryRow(`
		SELECT id, buyer_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1 AND buyer_id = $2
	`, orderID, buyerID).Scan(
		&order.ID,
		&order.BuyerID,
		&order.Status,
		&order.TotalAmount,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	return order, nil
}

// MarkPaid confirms a pending order after verified payment.
func (r *OrderRepository) MarkPaid(orderID int) error {
	if orderID <= 0 {
		return fmt.Errorf("invalid order")
	}

	result, err := r.db.Exec(`
		UPDATE orders
		SET status = 'confirmed', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND status = 'pending'
	`, orderID)
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var status string
		err := r.db.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
		if err == sql.ErrNoRows {
			return fmt.Errorf("order not found")
		}
		if err != nil {
			return err
		}
		if status != "confirmed" && status != "processing" && status != "delivered" {
			return fmt.Errorf("order is not awaiting payment")
		}
	}

	return nil
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
			order_items.product_id,
			order_items.quantity,
			order_items.total_price,
			orders.status,
			orders.created_at,
			products.name,
			products.unit,
			COALESCE(
				seller_user.full_name,
				seller_org.name,
				''
			) AS seller_name,
			CASE
				WHEN products.owner_type = 'user' THEN products.owner_id
				ELSE 0
			END AS seller_id,
			COALESCE(
				(
					SELECT image_url
					FROM product_images
					WHERE product_id = products.id
					ORDER BY is_primary DESC, sort_order ASC, id ASC
					LIMIT 1
				),
				''
			) AS image_url,
			buyer.full_name AS buyer_name
		FROM orders
		JOIN order_items ON order_items.order_id = orders.id
		JOIN products ON products.id = order_items.product_id
		JOIN users buyer
			ON buyer.id = orders.buyer_id
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
			&o.ProductUnit,
			&o.SellerName,
			&o.SellerID,
			&o.ImageURL,
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
			order_items.product_id,
			order_items.quantity,
			order_items.total_price,
			orders.status,
			orders.created_at,
			products.name,
			products.unit,
			buyer.full_name,
			COALESCE(
				seller_user.full_name,
				seller_org.name,
				''
			) AS seller_name,
			CASE
				WHEN products.owner_type = 'user' THEN products.owner_id
				ELSE 0
			END AS seller_id,
			COALESCE(
				(
					SELECT image_url
					FROM product_images
					WHERE product_id = products.id
					ORDER BY is_primary DESC, sort_order ASC, id ASC
					LIMIT 1
				),
				''
			) AS image_url
		FROM orders
		JOIN order_items ON order_items.order_id = orders.id
		JOIN products ON products.id = order_items.product_id
		JOIN users buyer
			ON buyer.id = orders.buyer_id
		LEFT JOIN users seller_user
			ON products.owner_type = 'user'
			AND seller_user.id = products.owner_id
		LEFT JOIN organizations seller_org
			ON products.owner_type = 'organization'
			AND seller_org.id = products.owner_id
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
			&o.ProductUnit,
			&o.BuyerName,
			&o.SellerName,
			&o.SellerID,
			&o.ImageURL,
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
