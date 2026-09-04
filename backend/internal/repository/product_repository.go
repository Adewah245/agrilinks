package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
)

// ProductRepository handles all database operations related to products.
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
	}
}

// ============================================================
// CREATE PRODUCT
// ============================================================

func (r *ProductRepository) Create(product *models.Product) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	if product.OwnerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	if strings.TrimSpace(product.OwnerType) == "" {
		return fmt.Errorf("product owner type is required")
	}

	if strings.TrimSpace(product.Name) == "" {
		return fmt.Errorf("product name is required")
	}

	if product.Quantity < 0 {
		return fmt.Errorf("product quantity cannot be negative")
	}

	if product.Price < 0 {
		return fmt.Errorf("product price cannot be negative")
	}

	if strings.TrimSpace(product.Unit) == "" {
		return fmt.Errorf("product unit is required")
	}

	if strings.TrimSpace(product.Category) == "" {
		return fmt.Errorf("product category is required")
	}

	if product.Status == "" {
		product.Status = "inactive"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO products (
			owner_type,
			owner_id,
			name,
			description,
			category,
			quantity,
			unit,
			price,
			state_id,
			lga_id,
			community,
			latitude,
			longitude,
			location_accuracy,
			status,
			initial_listed_quantity,
			first_listed_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			CASE
				WHEN $15 = 'active'
				THEN $6
				ELSE NULL
			END,
			CASE
				WHEN $15 = 'active'
				THEN CURRENT_TIMESTAMP
				ELSE NULL
			END
		)
		RETURNING id
	`

	err = tx.QueryRow(
		query,
		product.OwnerType,
		product.OwnerID,
		product.Name,
		product.Description,
		product.Category,
		product.Quantity,
		product.Unit,
		product.Price,
		product.StateID,
		product.LGAID,
		product.Community,
		product.Latitude,
		product.Longitude,
		product.LocationAccuracy,
		product.Status,
	).Scan(&product.ID)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// Save initial price history when a price exists.
	if product.Price > 0 {
		_, err = tx.Exec(`
			INSERT INTO product_price_history (
				product_id,
				price,
				unit,
				source
			)
			VALUES ($1, $2, $3, $4)
		`,
			product.ID,
			product.Price,
			product.Unit,
			"Agro-Shield",
		)

		if err != nil {
			return fmt.Errorf("failed to save product price history: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit product creation: %w", err)
	}

	return nil
}

// ============================================================
// LIST PRODUCTS BY OWNER
// ============================================================

func (r *ProductRepository) ListByOwner(
	ownerType string,
	ownerID int,
) ([]models.Product, error) {

	query := `
		SELECT
			id,
			owner_type,
			owner_id,
			name,
			description,
			category,
			quantity,
			unit,
			price,
			state_id,
			lga_id,
			community,
			latitude,
			longitude,
			location_accuracy,
			status,
			COALESCE(initial_listed_quantity, 0),
			first_listed_at,
			first_order_at,
			sold_out_at,
			COALESCE(created_at, CURRENT_TIMESTAMP),
			COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM products
		WHERE owner_type = $1
		AND owner_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product

	for rows.Next() {
		var product models.Product

		err := rows.Scan(
			&product.ID,
			&product.OwnerType,
			&product.OwnerID,
			&product.Name,
			&product.Description,
			&product.Category,
			&product.Quantity,
			&product.Unit,
			&product.Price,
			&product.StateID,
			&product.LGAID,
			&product.Community,
			&product.Latitude,
			&product.Longitude,
			&product.LocationAccuracy,
			&product.Status,
			&product.InitialListedQuantity,
			&product.FirstListedAt,
			&product.FirstOrderAt,
			&product.SoldOutAt,
			&product.CreatedAt,
			&product.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// ============================================================
// LIST AVAILABLE PRODUCTS
// ============================================================

func (r *ProductRepository) ListAvailable() ([]models.Product, error) {

	query := `
		SELECT
			p.id,
			p.owner_type,
			p.owner_id,
			p.name,
			p.description,
			p.category,
			p.quantity,
			p.unit,
			p.price,
			p.state_id,
			p.lga_id,
			p.community,
			COALESCE(p.latitude, 0),
			COALESCE(p.longitude, 0),
			COALESCE(p.location_accuracy, 0),
			p.status,
			COALESCE(p.initial_listed_quantity, 0),
			p.first_listed_at,
			p.first_order_at,
			p.sold_out_at,
			COALESCE(p.created_at, CURRENT_TIMESTAMP),
			COALESCE(p.updated_at, CURRENT_TIMESTAMP),
			COALESCE(u.full_name, o.name, ''),
			COALESCE(u.photo_url, '')
		FROM products p
		LEFT JOIN users u
			ON p.owner_type = 'user'
			AND u.id = p.owner_id
		LEFT JOIN organizations o
			ON p.owner_type = 'organization'
			AND o.id = p.owner_id
		WHERE p.status = 'active'
		AND p.quantity > 0
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product

	for rows.Next() {
		var product models.Product

		err := rows.Scan(
			&product.ID,
			&product.OwnerType,
			&product.OwnerID,
			&product.Name,
			&product.Description,
			&product.Category,
			&product.Quantity,
			&product.Unit,
			&product.Price,
			&product.StateID,
			&product.LGAID,
			&product.Community,
			&product.Latitude,
			&product.Longitude,
			&product.LocationAccuracy,
			&product.Status,
			&product.InitialListedQuantity,
			&product.FirstListedAt,
			&product.FirstOrderAt,
			&product.SoldOutAt,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.SellerName,
			&product.SellerPhotoURL,
		)

		if err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// ============================================================
// GET PRODUCT BY ID
// ============================================================

func (r *ProductRepository) GetByID(
	id int,
) (*models.Product, error) {

	query := `
		SELECT
			id,
			owner_type,
			owner_id,
			name,
			description,
			category,
			quantity,
			unit,
			price,
			state_id,
			lga_id,
			community,
			COALESCE(latitude, 0),
			COALESCE(longitude, 0),
			COALESCE(location_accuracy, 0),
			status,
			COALESCE(initial_listed_quantity, 0),
			first_listed_at,
			first_order_at,
			sold_out_at,
			COALESCE(created_at, CURRENT_TIMESTAMP),
			COALESCE(updated_at, CURRENT_TIMESTAMP)
		FROM products
		WHERE id = $1
	`

	var product models.Product

	err := r.db.QueryRow(query, id).Scan(
		&product.ID,
		&product.OwnerType,
		&product.OwnerID,
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Quantity,
		&product.Unit,
		&product.Price,
		&product.StateID,
		&product.LGAID,
		&product.Community,
		&product.Latitude,
		&product.Longitude,
		&product.LocationAccuracy,
		&product.Status,
		&product.InitialListedQuantity,
		&product.FirstListedAt,
		&product.FirstOrderAt,
		&product.SoldOutAt,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &product, nil
}

// ============================================================
// UPDATE PRODUCT
// ============================================================

func (r *ProductRepository) Update(
	product *models.Product,
) error {

	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE products
		SET
			name = $1,
			description = $2,
			category = $3,
			quantity = $4,
			unit = $5,
			price = $6,
			state_id = $7,
			lga_id = $8,
			community = $9,
			latitude = $10,
			longitude = $11,
			location_accuracy = $12,
			status = $13,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $14
		AND owner_type = $15
		AND owner_id = $16
	`

	result, err := tx.Exec(
		query,
		product.Name,
		product.Description,
		product.Category,
		product.Quantity,
		product.Unit,
		product.Price,
		product.StateID,
		product.LGAID,
		product.Community,
		product.Latitude,
		product.Longitude,
		product.LocationAccuracy,
		product.Status,
		product.ID,
		product.OwnerType,
		product.OwnerID,
	)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if product.Price > 0 {
		_, err = tx.Exec(`
			INSERT INTO product_price_history (
				product_id,
				price,
				unit,
				source
			)
			VALUES ($1, $2, $3, $4)
		`,
			product.ID,
			product.Price,
			product.Unit,
			"Agro-Shield",
		)

		if err != nil {
			return fmt.Errorf("failed to save product price history: %w", err)
		}
	}

	return tx.Commit()
}

// ============================================================
// UNLIST PRODUCT
// ============================================================

func (r *ProductRepository) Unlist(
	productID int,
	ownerType string,
	ownerID int,
) error {

	result, err := r.db.Exec(`
		UPDATE products
		SET
			status = 'inactive',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
	`,
		productID,
		ownerType,
		ownerID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ============================================================
// RELIST PRODUCT
// ============================================================

func (r *ProductRepository) Relist(
	productID int,
	ownerType string,
	ownerID int,
) error {

	result, err := r.db.Exec(`
		UPDATE products
		SET
			status = CASE
				WHEN quantity > 0 THEN 'active'
				ELSE 'sold_out'
			END,

			first_listed_at = COALESCE(
				first_listed_at,
				CURRENT_TIMESTAMP
			),

			initial_listed_quantity =
				CASE
					WHEN initial_listed_quantity IS NULL
					OR initial_listed_quantity < quantity
					THEN quantity
					ELSE initial_listed_quantity
				END,

			sold_out_at = CASE
				WHEN quantity > 0 THEN NULL
				ELSE COALESCE(sold_out_at, CURRENT_TIMESTAMP)
			END,

			updated_at = CURRENT_TIMESTAMP

		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
		AND quantity > 0
	`,
		productID,
		ownerType,
		ownerID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ============================================================
// DELETE PRODUCT
// ============================================================

func (r *ProductRepository) Delete(
	productID int,
	ownerType string,
	ownerID int,
) error {

	result, err := r.db.Exec(`
		DELETE FROM products
		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
	`,
		productID,
		ownerType,
		ownerID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ============================================================
// REDUCE PRODUCT QUANTITY
// ============================================================

func (r *ProductRepository) ReduceQuantity(
	productID int,
	amount float64,
) error {

	if productID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if amount <= 0 {
		return fmt.Errorf("quantity reduction must be positive")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var before float64

	err = tx.QueryRow(`
		SELECT quantity
		FROM products
		WHERE id = $1
		FOR UPDATE
	`,
		productID,
	).Scan(&before)

	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		UPDATE products
		SET
			quantity = quantity - $1,

			first_order_at = COALESCE(
				first_order_at,
				CURRENT_TIMESTAMP
			),

			sold_out_at = CASE
				WHEN quantity - $1 <= 0
				THEN CURRENT_TIMESTAMP
				ELSE sold_out_at
			END,

			status = CASE
				WHEN quantity - $1 <= 0
				THEN 'sold_out'
				ELSE status
			END,

			updated_at = CURRENT_TIMESTAMP

		WHERE id = $2
		AND quantity >= $1
	`,
		amount,
		productID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Record inventory movement.
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
		productID,
		before,
		-amount,
		before-amount,
		"order",
	)

	if err != nil {
		return fmt.Errorf("failed to save inventory history: %w", err)
	}

	return tx.Commit()
}

// ============================================================
// ADD PRODUCT IMAGES
// ============================================================

func (r *ProductRepository) AddImages(
	productID int,
	imageURLs []string,
) error {

	if productID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if len(imageURLs) == 0 {
		return fmt.Errorf("at least one product image is required")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	validCount := 0

	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)

		if imageURL == "" {
			continue
		}

		isPrimary := validCount == 0

		_, err := tx.Exec(`
			INSERT INTO product_images (
				product_id,
				image_url,
				is_primary,
				sort_order
			)
			VALUES ($1, $2, $3, $4)
		`,
			productID,
			imageURL,
			isPrimary,
			validCount,
		)

		if err != nil {
			return err
		}

		validCount++
	}

	if validCount == 0 {
		return fmt.Errorf("at least one valid product image is required")
	}

	return tx.Commit()
}

// ============================================================
// LIST PRODUCT IMAGES
// ============================================================

func (r *ProductRepository) ListImages(
	productID int,
) ([]models.ProductImage, error) {

	if productID <= 0 {
		return nil, fmt.Errorf("invalid product ID")
	}

	rows, err := r.db.Query(`
		SELECT
			id,
			product_id,
			image_url,
			is_primary,
			sort_order,
			created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY sort_order ASC, id ASC
	`,
		productID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.ProductImage

	for rows.Next() {
		var image models.ProductImage

		err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.ImageURL,
			&image.IsPrimary,
			&image.SortOrder,
			&image.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		images = append(images, image)
	}

	return images, rows.Err()
}
