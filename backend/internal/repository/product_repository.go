package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
)

// ProductRepository handles all database operations related to products.
//
// Products may belong to either an individual user or an organization.
// ProductRepository also maintains product images, price history,
// inventory history, and listing status history.
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
	}
}

// Create creates a new product.
//
// The product is created together with its initial price history,
// inventory history, and status history so that the complete lifecycle
// of the product is recorded from the beginning.
func (r *ProductRepository) Create(product *models.Product) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	if product.OwnerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	product.OwnerType = strings.TrimSpace(strings.ToLower(product.OwnerType))

	if product.OwnerType != "user" && product.OwnerType != "organization" {
		return fmt.Errorf("product owner type must be user or organization")
	}

	product.Name = strings.TrimSpace(product.Name)
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	product.Category = strings.TrimSpace(product.Category)
	if product.Category == "" {
		return fmt.Errorf("product category is required")
	}

	product.Unit = strings.TrimSpace(product.Unit)
	if product.Unit == "" {
		return fmt.Errorf("product unit is required")
	}

	if product.Quantity < 0 {
		return fmt.Errorf("product quantity cannot be negative")
	}

	if product.Price < 0 {
		return fmt.Errorf("product price cannot be negative")
	}

	product.Status = strings.TrimSpace(strings.ToLower(product.Status))

	if product.Status == "" {
		product.Status = "active"
	}

	switch product.Status {
	case "active", "sold_out", "inactive":
	default:
		return fmt.Errorf("invalid product status: %s", product.Status)
	}

	// A product with no available quantity cannot be actively sold.
	if product.Quantity <= 0 && product.Status == "active" {
		product.Status = "sold_out"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin product creation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var productID int

	err = tx.QueryRow(`
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
				WHEN $15 = 'active' THEN $6
				ELSE NULL
			END,
			CASE
				WHEN $15 = 'active' THEN CURRENT_TIMESTAMP
				ELSE NULL
			END
		)
		RETURNING id
	`,
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
	).Scan(&productID)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	product.ID = productID

	// Record the initial price.
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
		return fmt.Errorf("failed to save initial product price history: %w", err)
	}

	// Record the initial inventory state.
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
		product.ID,
		0,
		product.Quantity,
		product.Quantity,
		"initial_stock",
	)

	if err != nil {
		return fmt.Errorf("failed to save initial inventory history: %w", err)
	}

	// Record the initial listing status.
	_, err = tx.Exec(`
		INSERT INTO product_status_history (
			product_id,
			old_status,
			new_status,
			reason,
			changed_by_type,
			changed_by_id
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		product.ID,
		nil,
		product.Status,
		"product_created",
		product.OwnerType,
		product.OwnerID,
	)

	if err != nil {
		return fmt.Errorf("failed to save initial product status history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit product creation: %w", err)
	}

	return nil
}

// ListByOwner returns all products belonging to a specific owner.
//
// ownerType must be either "user" or "organization".
func (r *ProductRepository) ListByOwner(
	ownerType string,
	ownerID int,
) ([]models.Product, error) {

	ownerType = strings.TrimSpace(strings.ToLower(ownerType))

	if ownerType != "user" && ownerType != "organization" {
		return nil, fmt.Errorf("product owner type must be user or organization")
	}

	if ownerID <= 0 {
		return nil, fmt.Errorf("invalid product owner")
	}

	rows, err := r.db.Query(`
		SELECT
			id,
			owner_type,
			owner_id,
			name,
			COALESCE(description, ''),
			category,
			quantity,
			unit,
			price,
			state_id,
			lga_id,
			COALESCE(community, ''),
			COALESCE(latitude, 0),
			COALESCE(longitude, 0),
			COALESCE(location_accuracy, 0),
			status,
			COALESCE(initial_listed_quantity, 0),
			first_listed_at,
			first_order_at,
			sold_out_at,
			created_at,
			updated_at
		FROM products
		WHERE owner_type = $1
		AND owner_id = $2
		ORDER BY created_at DESC, id DESC
	`,
		ownerType,
		ownerID,
	)

	if err != nil {
		return nil, fmt.Errorf("list products by owner: %w", err)
	}

	defer rows.Close()

	products := make([]models.Product, 0)

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

// ListAvailable returns all products currently available for purchase.
//
// Only products with an active status and a quantity greater than zero
// are returned.
func (r *ProductRepository) ListAvailable() ([]models.Product, error) {
	rows, err := r.db.Query(`
		SELECT
			p.id,
			p.owner_type,
			p.owner_id,
			p.name,
			COALESCE(p.description, ''),
			p.category,
			p.quantity,
			p.unit,
			p.price,
			p.state_id,
			p.lga_id,
			COALESCE(p.community, ''),
			COALESCE(p.latitude, 0),
			COALESCE(p.longitude, 0),
			COALESCE(p.location_accuracy, 0),
			p.status,
			COALESCE(p.initial_listed_quantity, 0),
			p.first_listed_at,
			p.first_order_at,
			p.sold_out_at,
			p.created_at,
			p.updated_at,
			COALESCE(u.full_name, o.name, ''),
			COALESCE(u.photo_url, o.logo_url, '')
		FROM products p
		LEFT JOIN users u
			ON p.owner_type = 'user'
			AND u.id = p.owner_id
		LEFT JOIN organizations o
			ON p.owner_type = 'organization'
			AND o.id = p.owner_id
		WHERE p.status = 'active'
		AND p.quantity > 0
		ORDER BY p.created_at DESC, p.id DESC
	`)

	if err != nil {
		return nil, fmt.Errorf("list available products: %w", err)
	}

	defer rows.Close()

	products := make([]models.Product, 0)

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan available product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate available products: %w", err)
	}

	return products, nil
}

// GetByID returns a product by its database ID.
//
// sql.ErrNoRows is returned when the product does not exist.
func (r *ProductRepository) GetByID(id int) (*models.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid product ID")
	}

	var product models.Product

	err := r.db.QueryRow(`
		SELECT
			id,
			owner_type,
			owner_id,
			name,
			COALESCE(description, ''),
			category,
			quantity,
			unit,
			price,
			state_id,
			lga_id,
			COALESCE(community, ''),
			COALESCE(latitude, 0),
			COALESCE(longitude, 0),
			COALESCE(location_accuracy, 0),
			status,
			COALESCE(initial_listed_quantity, 0),
			first_listed_at,
			first_order_at,
			sold_out_at,
			created_at,
			updated_at
		FROM products
		WHERE id = $1
	`,
		id,
	).Scan(
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

	return &product, nil
}

// Update updates an existing product and records any important changes.
//
// Price changes are recorded in product_price_history.
// Quantity changes are recorded in inventory_history.
// Status changes are recorded in product_status_history.
func (r *ProductRepository) Update(product *models.Product) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	if product.ID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if product.OwnerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	product.OwnerType = strings.TrimSpace(strings.ToLower(product.OwnerType))

	if product.OwnerType != "user" && product.OwnerType != "organization" {
		return fmt.Errorf("product owner type must be user or organization")
	}

	product.Name = strings.TrimSpace(product.Name)
	if product.Name == "" {
		return fmt.Errorf("product name is required")
	}

	product.Category = strings.TrimSpace(product.Category)
	if product.Category == "" {
		return fmt.Errorf("product category is required")
	}

	product.Unit = strings.TrimSpace(product.Unit)
	if product.Unit == "" {
		return fmt.Errorf("product unit is required")
	}

	if product.Quantity < 0 {
		return fmt.Errorf("product quantity cannot be negative")
	}

	if product.Price < 0 {
		return fmt.Errorf("product price cannot be negative")
	}

	product.Status = strings.TrimSpace(strings.ToLower(product.Status))

	if product.Status == "" {
		if product.Quantity > 0 {
			product.Status = "active"
		} else {
			product.Status = "sold_out"
		}
	}

	switch product.Status {
	case "active", "sold_out", "inactive":
	default:
		return fmt.Errorf("invalid product status: %s", product.Status)
	}

	if product.Status == "active" && product.Quantity <= 0 {
		product.Status = "sold_out"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin product update transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		oldPrice    float64
		oldQuantity float64
		oldStatus   string
		oldUnit     string
	)

	err = tx.QueryRow(`
		SELECT
			price,
			quantity,
			status,
			unit
		FROM products
		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
		FOR UPDATE
	`,
		product.ID,
		product.OwnerType,
		product.OwnerID,
	).Scan(
		&oldPrice,
		&oldQuantity,
		&oldStatus,
		&oldUnit,
	)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
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
	`,
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

	// Record price history only when the price or pricing unit changes.
	if oldPrice != product.Price || oldUnit != product.Unit {
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

	// Record inventory changes.
	if oldQuantity != product.Quantity {
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
			product.ID,
			oldQuantity,
			product.Quantity-oldQuantity,
			product.Quantity,
			"product_update",
		)

		if err != nil {
			return fmt.Errorf("failed to save inventory history: %w", err)
		}
	}

	// Record status changes.
	if oldStatus != product.Status {
		_, err = tx.Exec(`
			INSERT INTO product_status_history (
				product_id,
				old_status,
				new_status,
				reason,
				changed_by_type,
				changed_by_id
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			product.ID,
			oldStatus,
			product.Status,
			"product_update",
			product.OwnerType,
			product.OwnerID,
		)

		if err != nil {
			return fmt.Errorf("failed to save product status history: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit product update: %w", err)
	}

	return nil
}

// Unlist changes a product's status to inactive.
//
// The product remains in the database and can be relisted by its owner.
func (r *ProductRepository) Unlist(
	productID int,
	ownerType string,
	ownerID int,
) error {
	ownerType = strings.TrimSpace(strings.ToLower(ownerType))

	if productID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if ownerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	if ownerType != "user" && ownerType != "organization" {
		return fmt.Errorf("product owner type must be user or organization")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin unlist transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var oldStatus string

	err = tx.QueryRow(`
		SELECT status
		FROM products
		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
		FOR UPDATE
	`,
		productID,
		ownerType,
		ownerID,
	).Scan(&oldStatus)

	if err != nil {
		return err
	}

	if oldStatus == "inactive" {
		return nil
	}

	_, err = tx.Exec(`
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
		return fmt.Errorf("unlist product: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO product_status_history (
			product_id,
			old_status,
			new_status,
			reason,
			changed_by_type,
			changed_by_id
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		productID,
		oldStatus,
		"inactive",
		"owner_unlisted",
		ownerType,
		ownerID,
	)

	if err != nil {
		return fmt.Errorf("save unlist status history: %w", err)
	}

	return tx.Commit()
}

// Relist changes an inactive or sold-out product back to active
// when it has available quantity.
func (r *ProductRepository) Relist(
	productID int,
	ownerType string,
	ownerID int,
) error {
	ownerType = strings.TrimSpace(strings.ToLower(ownerType))

	if productID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if ownerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	if ownerType != "user" && ownerType != "organization" {
		return fmt.Errorf("product owner type must be user or organization")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin relist transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		oldStatus string
		quantity  float64
	)

	err = tx.QueryRow(`
		SELECT
			status,
			quantity
		FROM products
		WHERE id = $1
		AND owner_type = $2
		AND owner_id = $3
		FOR UPDATE
	`,
		productID,
		ownerType,
		ownerID,
	).Scan(
		&oldStatus,
		&quantity,
	)

	if err != nil {
		return err
	}

	if quantity <= 0 {
		return fmt.Errorf("product cannot be relisted without available quantity")
	}

	_, err = tx.Exec(`
		UPDATE products
		SET
			status = 'active',
			initial_listed_quantity = CASE
				WHEN initial_listed_quantity IS NULL
				OR initial_listed_quantity < quantity
				THEN quantity
				ELSE initial_listed_quantity
			END,
			first_listed_at = COALESCE(
				first_listed_at,
				CURRENT_TIMESTAMP
			),
			sold_out_at = NULL,
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
		return fmt.Errorf("relist product: %w", err)
	}

	if oldStatus != "active" {
		_, err = tx.Exec(`
			INSERT INTO product_status_history (
				product_id,
				old_status,
				new_status,
				reason,
				changed_by_type,
				changed_by_id
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			productID,
			oldStatus,
			"active",
			"owner_relisted",
			ownerType,
			ownerID,
		)

		if err != nil {
			return fmt.Errorf("save relist status history: %w", err)
		}
	}

	return tx.Commit()
}

// Delete permanently deletes a product belonging to the specified owner.
//
// Related product images, favorites, price history, inventory history,
// and status history are automatically removed through database
// cascading foreign keys.
func (r *ProductRepository) Delete(
	productID int,
	ownerType string,
	ownerID int,
) error {
	ownerType = strings.TrimSpace(strings.ToLower(ownerType))

	if productID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if ownerID <= 0 {
		return fmt.Errorf("invalid product owner")
	}

	if ownerType != "user" && ownerType != "organization" {
		return fmt.Errorf("product owner type must be user or organization")
	}

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
		return fmt.Errorf("delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted product: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ReduceQuantity decreases the available quantity of a product.
//
// The operation uses row-level locking so simultaneous orders cannot
// incorrectly reduce the same inventory.
//
// When quantity reaches zero, the product is automatically marked
// as sold_out and the status change is recorded.
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
		return fmt.Errorf("begin quantity reduction transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var (
		before    float64
		oldStatus string
	)

	err = tx.QueryRow(`
		SELECT
			quantity,
			status
		FROM products
		WHERE id = $1
		FOR UPDATE
	`,
		productID,
	).Scan(
		&before,
		&oldStatus,
	)

	if err != nil {
		return err
	}

	if before < amount {
		return fmt.Errorf(
			"insufficient product quantity: available %.2f, requested %.2f",
			before,
			amount,
		)
	}

	after := before - amount

	newStatus := oldStatus

	if after <= 0 {
		after = 0
		newStatus = "sold_out"
	}

	_, err = tx.Exec(`
		UPDATE products
		SET
			quantity = $1,
			first_order_at = COALESCE(
				first_order_at,
				CURRENT_TIMESTAMP
			),
			sold_out_at = CASE
				WHEN $1 <= 0
				THEN COALESCE(sold_out_at, CURRENT_TIMESTAMP)
				ELSE sold_out_at
			END,
			status = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`,
		after,
		newStatus,
		productID,
	)

	if err != nil {
		return fmt.Errorf("reduce product quantity: %w", err)
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
		productID,
		before,
		-after+before,
		after,
		"order",
	)

	if err != nil {
		return fmt.Errorf("failed to save inventory history: %w", err)
	}

	if oldStatus != newStatus {
		_, err = tx.Exec(`
			INSERT INTO product_status_history (
				product_id,
				old_status,
				new_status,
				reason,
				changed_by_type,
				changed_by_id
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
			productID,
			oldStatus,
			newStatus,
			"inventory_depleted",
			"system",
			nil,
		)

		if err != nil {
			return fmt.Errorf("failed to save sold out status history: %w", err)
		}
	}

	return tx.Commit()
}

// AddImages adds one or more images to a product.
//
// If the product does not already have a primary image, the first valid
// image supplied becomes the primary image.
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
		return fmt.Errorf("begin add product images transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool

	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM products
			WHERE id = $1
		)
	`,
		productID,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("check product before adding images: %w", err)
	}

	if !exists {
		return sql.ErrNoRows
	}

	var hasPrimary bool

	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM product_images
			WHERE product_id = $1
			AND is_primary = TRUE
		)
	`,
		productID,
	).Scan(&hasPrimary)

	if err != nil {
		return fmt.Errorf("check primary product image: %w", err)
	}

	var nextSortOrder int

	err = tx.QueryRow(`
		SELECT COALESCE(MAX(sort_order) + 1, 0)
		FROM product_images
		WHERE product_id = $1
	`,
		productID,
	).Scan(&nextSortOrder)

	if err != nil {
		return fmt.Errorf("get next product image sort order: %w", err)
	}

	inserted := 0

	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)

		if imageURL == "" {
			continue
		}

		isPrimary := false

		if !hasPrimary && inserted == 0 {
			isPrimary = true
		}

		_, err = tx.Exec(`
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
			nextSortOrder,
		)

		if err != nil {
			return fmt.Errorf("add product image: %w", err)
		}

		inserted++
		nextSortOrder++
	}

	if inserted == 0 {
		return fmt.Errorf("at least one valid product image is required")
	}

	return tx.Commit()
}

// ListImages returns all images belonging to a product.
//
// Images are ordered by sort order and then by database ID.
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
		return nil, fmt.Errorf("list product images: %w", err)
	}

	defer rows.Close()

	images := make([]models.ProductImage, 0)

	for rows.Next() {
		var image models.ProductImage

		if err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.ImageURL,
			&image.IsPrimary,
			&image.SortOrder,
			&image.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product images: %w", err)
	}

	return images, nil
}
