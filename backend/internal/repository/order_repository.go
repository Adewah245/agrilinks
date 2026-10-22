package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
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

// Create reserves stock and saves an unpaid order in one transaction.
// The caller provides product IDs and quantities; prices come from the database.
func (r *OrderRepository) Create(order *models.Order) error {
	return r.checkout(order, nil)
}

// CreateFromCart re-reads and locks the buyer's cart inside the checkout.
// Prices in accepted negotiations are kept even if the listing price changes.
func (r *OrderRepository) CreateFromCart(order *models.Order, cartItemIDs []int) error {
	return r.checkout(order, cartItemIDs)
}

func (r *OrderRepository) checkout(order *models.Order, cartItemIDs []int) error {
	if order == nil || order.BuyerID <= 0 || (cartItemIDs == nil && len(order.Items) == 0) {
		return fmt.Errorf("choose at least one product")
	}
	if len(order.Currency) != 3 {
		return fmt.Errorf("order currency is required")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Serializing a buyer's checkout also protects repeated request keys.
	var buyer int
	if err := tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, order.BuyerID).Scan(&buyer); err != nil {
		return err
	}
	if order.CheckoutKey != "" {
		err = tx.QueryRow(`SELECT id,status,total_amount,currency,created_at,updated_at FROM orders WHERE buyer_id=$1 AND checkout_key=$2`, order.BuyerID, order.CheckoutKey).
			Scan(&order.ID, &order.Status, &order.TotalAmount, &order.Currency, &order.CreatedAt, &order.UpdatedAt)
		if err == nil {
			return tx.Commit()
		}
		if err != sql.ErrNoRows {
			return err
		}
	}
	fromCart := cartItemIDs != nil
	if fromCart {
		// IDs and quantities supplied before this transaction are not trusted.
		rows, err := tx.Query(`SELECT id,product_id,quantity,price_per_unit FROM cart_items WHERE buyer_id=$1 ORDER BY product_id,id FOR UPDATE`, order.BuyerID)
		if err != nil {
			return err
		}
		order.Items = nil
		cartItemIDs = nil
		for rows.Next() {
			var id int
			var item models.OrderItem
			if err := rows.Scan(&id, &item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
				rows.Close()
				return err
			}
			cartItemIDs = append(cartItemIDs, id)
			order.Items = append(order.Items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(order.Items) == 0 {
			return fmt.Errorf("your cart is empty")
		}
	}
	sort.SliceStable(order.Items, func(i, j int) bool { return order.Items[i].ProductID < order.Items[j].ProductID })
	order.Status = "pending"
	err = tx.QueryRow(`INSERT INTO orders(buyer_id,total_amount,status,checkout_key,currency) VALUES($1,0,'pending',NULLIF($2,''),$3) RETURNING id,created_at,updated_at`, order.BuyerID, order.CheckoutKey, order.Currency).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return err
	}
	var sellerType string
	var sellerID int
	for i := range order.Items {
		item := &order.Items[i]
		if item.ProductID <= 0 || math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) || item.Quantity <= 0 || math.Round(item.Quantity*1e6)/1e6 != item.Quantity {
			return fmt.Errorf("enter a valid amount with at most six decimal places")
		}
		var before float64
		var status, ownerType, price string
		var ownerID int
		err = tx.QueryRow(`SELECT quantity,status,owner_type,owner_id,price::text FROM products WHERE id=$1 FOR UPDATE`, item.ProductID).
			Scan(&before, &status, &ownerType, &ownerID, &price)
		if err == sql.ErrNoRows {
			return fmt.Errorf("product no longer exists")
		}
		if err != nil {
			return err
		}
		if status != "active" || item.Quantity > before {
			return fmt.Errorf("the amount you asked for is no longer available")
		}
		if ownerType == "user" && ownerID == order.BuyerID {
			return fmt.Errorf("you cannot buy your own product")
		}
		if i == 0 {
			sellerType, sellerID = ownerType, ownerID
		} else if ownerType != sellerType || ownerID != sellerID {
			return fmt.Errorf("check out products from one seller at a time")
		}
		if fromCart {
			price = strconv.FormatFloat(item.UnitPrice, 'f', 2, 64)
		}
		// PostgreSQL NUMERIC does the money calculation, including line rounding.
		err = tx.QueryRow(`INSERT INTO order_items(order_id,product_id,quantity,unit_price,total_price) VALUES($1,$2,$3::numeric,$4::numeric,round($3::numeric*$4::numeric,2)) RETURNING id,unit_price,total_price,created_at`, order.ID, item.ProductID, strconv.FormatFloat(item.Quantity, 'f', 6, 64), price).
			Scan(&item.ID, &item.UnitPrice, &item.TotalPrice, &item.CreatedAt)
		if err != nil {
			return err
		}
		if item.TotalPrice <= 0 {
			return fmt.Errorf("the order amount must be greater than zero")
		}
		item.OrderID = order.ID
		var after float64
		var newStatus string
		err = tx.QueryRow(`UPDATE products SET quantity=quantity-$2::numeric,
            status=CASE WHEN quantity-$2::numeric=0 THEN 'sold_out' ELSE status END,
            sold_out_at=CASE WHEN quantity-$2::numeric=0 THEN CURRENT_TIMESTAMP ELSE sold_out_at END,
            updated_at=CURRENT_TIMESTAMP WHERE id=$1 RETURNING quantity,status`, item.ProductID, strconv.FormatFloat(item.Quantity, 'f', 6, 64)).Scan(&after, &newStatus)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO inventory_history(product_id,quantity_before,quantity_change,quantity_after,reason) VALUES($1,$2,$3,$4,'order_placed')`, item.ProductID, before, -item.Quantity, after)
		if err != nil {
			return err
		}
		if status != newStatus {
			_, err = tx.Exec(`INSERT INTO product_status_history(product_id,old_status,new_status,reason,changed_by_type,changed_by_id) VALUES($1,$2,$3,'order_placed','user',$4)`, item.ProductID, status, newStatus, order.BuyerID)
			if err != nil {
				return err
			}
		}
	}
	err = tx.QueryRow(`UPDATE orders SET total_amount=(SELECT SUM(total_price) FROM order_items WHERE order_id=$1) WHERE id=$1 RETURNING total_amount`, order.ID).Scan(&order.TotalAmount)
	if err != nil {
		return err
	}
	if fromCart {
		for _, id := range cartItemIDs {
			if _, err = tx.Exec(`DELETE FROM cart_items WHERE id=$1 AND buyer_id=$2`, id, order.BuyerID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
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
		SELECT id, buyer_id, status, total_amount, currency, created_at, updated_at
		FROM orders
		WHERE id = $1 AND buyer_id = $2
	`, orderID, buyerID).Scan(
		&order.ID,
		&order.BuyerID,
		&order.Status,
		&order.TotalAmount,
		&order.Currency,
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
            orders.total_amount,
            orders.currency,
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
			&o.TotalAmount,
			&o.Currency,
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
            orders.total_amount,
            orders.currency,
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
			&o.TotalAmount,
			&o.Currency,
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
