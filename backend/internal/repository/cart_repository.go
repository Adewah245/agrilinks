package repository

import (
	"backend/internal/models"
	"database/sql"
)

// CartRepository handles all database operations related to the buyer's cart.
type CartRepository struct {
	db *sql.DB
}

// NewCartRepository creates and returns a new CartRepository.
//
// Responsibility:
// - Connect the cart repository to the application's database.
func NewCartRepository(db *sql.DB) *CartRepository {
	return &CartRepository{db: db}
}

// Create adds an accepted negotiation to the buyer's cart.
//
// Responsibility:
// - Store the agreed product, quantity, and negotiated price.
// - Preserve the negotiated price.
// - Connect the cart item to the buyer, crop, and negotiation.
func (r *CartRepository) Create(item *models.CartItem) error {
	query := `
		INSERT INTO cart_items (
			buyer_id,
			crop_id,
			negotiation_id,
			quantity,
			price_per_unit,
			total_price
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		item.BuyerID,
		item.CropID,
		item.NegotiationID,
		item.Quantity,
		item.PricePerUnit,
		item.TotalPrice,
	).Scan(
		&item.ID,
		&item.CreatedAt,
	)
}

// ListByBuyer retrieves all cart items belonging to a buyer.
//
// Responsibility:
// - Show everything currently waiting in the buyer's cart.
// - Include product information.
// - Include seller information.
// - Include buyer information.
// - Use JOINs so account information is always current.
func (r *CartRepository) ListByBuyer(buyerID int) ([]models.CartItem, error) {
	query := `
		SELECT
			cart_items.id,
			cart_items.buyer_id,
			cart_items.crop_id,
			cart_items.negotiation_id,
			cart_items.quantity,
			cart_items.price_per_unit,
			cart_items.total_price,
			cart_items.created_at,

			crops.name,
			crops.unit,
			crops.image_url,

			seller.id,
			seller.full_name,
			seller.email,
			seller.phone,
			seller.location,
			seller.photo_url,

			buyer.full_name,
			buyer.email,
			buyer.phone,
			buyer.location,
			buyer.photo_url

		FROM cart_items

		JOIN crops
			ON crops.id = cart_items.crop_id

		JOIN farmers AS seller
			ON seller.id = crops.farmer_id

		JOIN farmers AS buyer
			ON buyer.id = cart_items.buyer_id

		WHERE cart_items.buyer_id = $1

		ORDER BY cart_items.created_at DESC
	`

	rows, err := r.db.Query(query, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.CartItem

	for rows.Next() {
		var item models.CartItem

		err := rows.Scan(
			&item.ID,
			&item.BuyerID,
			&item.CropID,
			&item.NegotiationID,
			&item.Quantity,
			&item.PricePerUnit,
			&item.TotalPrice,
			&item.CreatedAt,

			&item.CropName,
			&item.Unit,
			&item.ImageURL,

			&item.SellerID,
			&item.SellerName,
			&item.SellerEmail,
			&item.SellerPhone,
			&item.SellerLocation,
			&item.SellerPhotoURL,

			&item.BuyerName,
			&item.BuyerEmail,
			&item.BuyerPhone,
			&item.BuyerLocation,
			&item.BuyerPhotoURL,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// GetByID retrieves one cart item.
//
// Responsibility:
// - Find a specific cart item.
// - Make sure it belongs to the requested buyer.
// - Include complete product, seller, and buyer information.
func (r *CartRepository) GetByID(
	cartID,
	buyerID int,
) (*models.CartItem, error) {

	query := `
		SELECT
			cart_items.id,
			cart_items.buyer_id,
			cart_items.crop_id,
			cart_items.negotiation_id,
			cart_items.quantity,
			cart_items.price_per_unit,
			cart_items.total_price,
			cart_items.created_at,

			crops.name,
			crops.unit,
			crops.image_url,

			seller.id,
			seller.full_name,
			seller.email,
			seller.phone,
			seller.location,
			seller.photo_url,

			buyer.full_name,
			buyer.email,
			buyer.phone,
			buyer.location,
			buyer.photo_url

		FROM cart_items

		JOIN crops
			ON crops.id = cart_items.crop_id

		JOIN farmers AS seller
			ON seller.id = crops.farmer_id

		JOIN farmers AS buyer
			ON buyer.id = cart_items.buyer_id

		WHERE cart_items.id = $1
		  AND cart_items.buyer_id = $2
	`

	var item models.CartItem

	err := r.db.QueryRow(
		query,
		cartID,
		buyerID,
	).Scan(
		&item.ID,
		&item.BuyerID,
		&item.CropID,
		&item.NegotiationID,
		&item.Quantity,
		&item.PricePerUnit,
		&item.TotalPrice,
		&item.CreatedAt,

		&item.CropName,
		&item.Unit,
		&item.ImageURL,

		&item.SellerID,
		&item.SellerName,
		&item.SellerEmail,
		&item.SellerPhone,
		&item.SellerLocation,
		&item.SellerPhotoURL,

		&item.BuyerName,
		&item.BuyerEmail,
		&item.BuyerPhone,
		&item.BuyerLocation,
		&item.BuyerPhotoURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &item, nil
}

// Delete removes an item from the buyer's cart.
//
// Responsibility:
// - Remove a cart item the buyer no longer wants.
// - Make sure the item belongs to that buyer.
func (r *CartRepository) Delete(
	cartID,
	buyerID int,
) error {

	query := `
		DELETE FROM cart_items
		WHERE id = $1
		  AND buyer_id = $2
	`

	_, err := r.db.Exec(
		query,
		cartID,
		buyerID,
	)

	return err
}