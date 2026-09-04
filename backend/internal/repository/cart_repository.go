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
func NewCartRepository(db *sql.DB) *CartRepository {
	return &CartRepository{db: db}
}

// Create adds an accepted negotiation to the buyer's cart.
//
// The negotiated product price is preserved so that later marketplace
// price changes do not alter the agreed deal.
func (r *CartRepository) Create(item *models.CartItem) error {
	query := `
		INSERT INTO cart_items (
			buyer_id,
			product_id,
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
		item.ProductID,
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
// Product, seller, and buyer information are loaded through JOINs
// so the cart displays current account and product information.
func (r *CartRepository) ListByBuyer(buyerID int) ([]models.CartItem, error) {
	query := `
		SELECT
			cart_items.id,
			cart_items.buyer_id,
			cart_items.product_id,
			cart_items.negotiation_id,
			cart_items.quantity,
			cart_items.price_per_unit,
			cart_items.total_price,
			cart_items.created_at,

			products.name,
			products.unit,

			COALESCE(
				(
					SELECT pi.image_url
					FROM product_images pi
					WHERE pi.product_id = products.id
					  AND pi.is_primary = TRUE
					ORDER BY pi.sort_order ASC
					LIMIT 1
				),
				''
			),

			COALESCE(seller.id, organization.id),
			COALESCE(seller.full_name, organization.name, ''),
			COALESCE(seller.email, ''),
			COALESCE(seller.phone, ''),
			COALESCE(seller.location, ''),
			COALESCE(seller.photo_url, organization.logo_url, ''),

			buyer.full_name,
			COALESCE(buyer.email, ''),
			buyer.phone,
			COALESCE(buyer.location, ''),
			COALESCE(buyer.photo_url, '')

		FROM cart_items

		JOIN products
			ON products.id = cart_items.product_id

		LEFT JOIN users AS seller
			ON products.owner_type = 'user'
			AND seller.id = products.owner_id

		LEFT JOIN organizations AS organization
			ON products.owner_type = 'organization'
			AND organization.id = products.owner_id

		JOIN users AS buyer
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

		if err := rows.Scan(
			&item.ID,
			&item.BuyerID,
			&item.ProductID,
			&item.NegotiationID,
			&item.Quantity,
			&item.PricePerUnit,
			&item.TotalPrice,
			&item.CreatedAt,

			&item.ProductName,
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
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// GetByID retrieves one cart item belonging to the specified buyer.
//
// Product, seller, and buyer information are loaded through JOINs.
func (r *CartRepository) GetByID(cartID, buyerID int) (*models.CartItem, error) {
	query := `
		SELECT
			cart_items.id,
			cart_items.buyer_id,
			cart_items.product_id,
			cart_items.negotiation_id,
			cart_items.quantity,
			cart_items.price_per_unit,
			cart_items.total_price,
			cart_items.created_at,

			products.name,
			products.unit,

			COALESCE(
				(
					SELECT pi.image_url
					FROM product_images pi
					WHERE pi.product_id = products.id
					  AND pi.is_primary = TRUE
					ORDER BY pi.sort_order ASC
					LIMIT 1
				),
				''
			),

			COALESCE(seller.id, organization.id),
			COALESCE(seller.full_name, organization.name, ''),
			COALESCE(seller.email, ''),
			COALESCE(seller.phone, ''),
			COALESCE(seller.location, ''),
			COALESCE(seller.photo_url, organization.logo_url, ''),

			buyer.full_name,
			COALESCE(buyer.email, ''),
			buyer.phone,
			COALESCE(buyer.location, ''),
			COALESCE(buyer.photo_url, '')

		FROM cart_items

		JOIN products
			ON products.id = cart_items.product_id

		LEFT JOIN users AS seller
			ON products.owner_type = 'user'
			AND seller.id = products.owner_id

		LEFT JOIN organizations AS organization
			ON products.owner_type = 'organization'
			AND organization.id = products.owner_id

		JOIN users AS buyer
			ON buyer.id = cart_items.buyer_id

		WHERE cart_items.id = $1
		  AND cart_items.buyer_id = $2
	`

	var item models.CartItem

	err := r.db.QueryRow(query, cartID, buyerID).Scan(
		&item.ID,
		&item.BuyerID,
		&item.ProductID,
		&item.NegotiationID,
		&item.Quantity,
		&item.PricePerUnit,
		&item.TotalPrice,
		&item.CreatedAt,

		&item.ProductName,
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

// Delete removes a cart item belonging to the specified buyer.
func (r *CartRepository) Delete(cartID, buyerID int) error {
	query := `
		DELETE FROM cart_items
		WHERE id = $1
		  AND buyer_id = $2
	`

	_, err := r.db.Exec(query, cartID, buyerID)

	return err
}
