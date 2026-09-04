package repository

import (
	"backend/internal/models"
	"database/sql"
)

// NegotiationRepository handles database operations for product negotiations.
type NegotiationRepository struct {
	db *sql.DB
}

// NewNegotiationRepository creates a new NegotiationRepository.
func NewNegotiationRepository(db *sql.DB) *NegotiationRepository {
	return &NegotiationRepository{db: db}
}

// Create stores a new negotiation between a buyer and a product owner.
func (r *NegotiationRepository) Create(n *models.Negotiation) error {
	query := `
		INSERT INTO negotiations (
			product_id,
			buyer_id,
			owner_type,
			owner_id,
			quantity,
			status,
			round_count,
			max_rounds,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, $8)
		RETURNING id
	`

	return r.db.QueryRow(
		query,
		n.ProductID,
		n.BuyerID,
		n.OwnerType,
		n.OwnerID,
		n.Quantity,
		n.Status,
		n.MaxRounds,
		n.ExpiresAt,
	).Scan(&n.ID)
}

// IncrementRound increases the negotiation round count by one.
func (r *NegotiationRepository) IncrementRound(id int) error {
	_, err := r.db.Exec(`
		UPDATE negotiations
		SET round_count = round_count + 1
		WHERE id = $1
	`, id)

	return err
}

// UpdateStatus changes the lifecycle status of a negotiation.
func (r *NegotiationRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec(`
		UPDATE negotiations
		SET status = $1
		WHERE id = $2
	`, status, id)

	return err
}

// GetByID retrieves a negotiation together with its product,
// buyer, and owner information.
func (r *NegotiationRepository) GetByID(id int) (*models.Negotiation, error) {
	query := `
		SELECT
			n.id,
			n.product_id,
			n.buyer_id,
			n.owner_type,
			n.owner_id,
			n.quantity,
			n.status,
			n.round_count,
			n.max_rounds,
			n.created_at,
			n.expires_at,

			p.name,

			COALESCE(b.full_name, ''),

			CASE
				WHEN n.owner_type = 'user'
					THEN COALESCE(s.full_name, '')
				WHEN n.owner_type = 'organization'
					THEN COALESCE(o.name, '')
				ELSE ''
			END,

			COALESCE(b.photo_url, ''),

			CASE
				WHEN n.owner_type = 'user'
					THEN COALESCE(s.photo_url, '')
				ELSE COALESCE(o.logo_url, '')
			END

		FROM negotiations n

		JOIN products p
			ON p.id = n.product_id

		LEFT JOIN users b
			ON b.id = n.buyer_id

		LEFT JOIN users s
			ON n.owner_type = 'user'
			AND s.id = n.owner_id

		LEFT JOIN organizations o
			ON n.owner_type = 'organization'
			AND o.id = n.owner_id

		WHERE n.id = $1
	`

	var n models.Negotiation

	err := r.db.QueryRow(query, id).Scan(
		&n.ID,
		&n.ProductID,
		&n.BuyerID,
		&n.OwnerType,
		&n.OwnerID,
		&n.Quantity,
		&n.Status,
		&n.RoundCount,
		&n.MaxRounds,
		&n.CreatedAt,
		&n.ExpiresAt,
		&n.ProductName,
		&n.BuyerName,
		&n.SellerName,
		&n.BuyerPhotoURL,
		&n.SellerPhotoURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &n, nil
}

// ListForUser returns every negotiation in which the user is the buyer
// or is the owner of the negotiated product through a user-owned listing.
func (r *NegotiationRepository) ListForUser(userID int) ([]models.Negotiation, error) {
	query := `
		SELECT
			n.id,
			n.product_id,
			n.buyer_id,
			n.owner_type,
			n.owner_id,
			n.quantity,
			n.status,
			n.round_count,
			n.max_rounds,
			n.created_at,
			n.expires_at,

			p.name,

			COALESCE(b.full_name, ''),

			CASE
				WHEN n.owner_type = 'user'
					THEN COALESCE(s.full_name, '')
				WHEN n.owner_type = 'organization'
					THEN COALESCE(o.name, '')
				ELSE ''
			END,

			COALESCE(b.photo_url, ''),

			CASE
				WHEN n.owner_type = 'user'
					THEN COALESCE(s.photo_url, '')
				ELSE COALESCE(o.logo_url, '')
			END

		FROM negotiations n

		JOIN products p
			ON p.id = n.product_id

		LEFT JOIN users b
			ON b.id = n.buyer_id

		LEFT JOIN users s
			ON n.owner_type = 'user'
			AND s.id = n.owner_id

		LEFT JOIN organizations o
			ON n.owner_type = 'organization'
			AND o.id = n.owner_id

		WHERE n.buyer_id = $1
		OR (
			n.owner_type = 'user'
			AND n.owner_id = $1
		)

		ORDER BY n.created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var negotiations []models.Negotiation

	for rows.Next() {
		var n models.Negotiation

		if err := rows.Scan(
			&n.ID,
			&n.ProductID,
			&n.BuyerID,
			&n.OwnerType,
			&n.OwnerID,
			&n.Quantity,
			&n.Status,
			&n.RoundCount,
			&n.MaxRounds,
			&n.CreatedAt,
			&n.ExpiresAt,
			&n.ProductName,
			&n.BuyerName,
			&n.SellerName,
			&n.BuyerPhotoURL,
			&n.SellerPhotoURL,
		); err != nil {
			return nil, err
		}

		negotiations = append(negotiations, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return negotiations, nil
}