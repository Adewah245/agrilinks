package repository

import (
	"backend/internal/models"
	"database/sql"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *models.Order) error {
	query := `
	INSERT INTO orders (buyer_id, crop_id, quantity, total_price, status)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
	`
	err := r.db.QueryRow(
		query,
		order.BuyerID,
		order.CropID,
		order.Quantity,
		order.TotalPrice,
		order.Status,
	).Scan(&order.ID)

	return err
}

// ListByBuyer returns everything a buyer has purchased, with the crop name
// and seller name resolved for display.
func (r *OrderRepository) ListByBuyer(buyerID int) ([]models.Order, error) {
	query := `
	SELECT orders.id, orders.buyer_id, orders.crop_id, orders.quantity, orders.total_price, orders.status, orders.created_at,
	       crops.name, farmers.full_name
	FROM orders
	JOIN crops ON crops.id = orders.crop_id
	JOIN farmers ON farmers.id = crops.farmer_id
	WHERE orders.buyer_id = $1
	ORDER BY orders.created_at DESC
	`
	rows, err := r.db.Query(query, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.CropID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt, &o.CropName, &o.SellerName); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ListSalesByFarmer returns everything sold from a farmer's crops, with the
// crop name and buyer name resolved for display.
func (r *OrderRepository) ListSalesByFarmer(farmerID int) ([]models.Order, error) {
	query := `
	SELECT orders.id, orders.buyer_id, orders.crop_id, orders.quantity, orders.total_price, orders.status, orders.created_at,
	       crops.name, farmers.full_name
	FROM orders
	JOIN crops ON crops.id = orders.crop_id
	JOIN farmers ON farmers.id = orders.buyer_id
	WHERE crops.farmer_id = $1
	ORDER BY orders.created_at DESC
	`
	rows, err := r.db.Query(query, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.CropID, &o.Quantity, &o.TotalPrice, &o.Status, &o.CreatedAt, &o.CropName, &o.BuyerName); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
