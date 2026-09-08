package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
)

// NegotiationRepository handles database operations for
// product negotiations.
type NegotiationRepository struct {
	db *sql.DB
}

// NewNegotiationRepository creates a new NegotiationRepository.
func NewNegotiationRepository(db *sql.DB) *NegotiationRepository {
	return &NegotiationRepository{
		db: db,
	}
}

// ============================================================
// CREATE
// ============================================================

// Create stores a new negotiation.
//
// The seller is NOT stored directly on negotiations.
// Seller ownership is derived from:
//
//	negotiations.product_id
//	      ↓
//	products.owner_type
//	products.owner_id
//
// initial_offer_price and current_offer_price are both stored
// because the buyer's offer can change during negotiation.
func (r *NegotiationRepository) Create(
	n *models.Negotiation,
) error {
	if n == nil {
		return fmt.Errorf("negotiation cannot be nil")
	}

	if n.ProductID <= 0 {
		return fmt.Errorf("invalid product ID")
	}

	if n.BuyerID <= 0 {
		return fmt.Errorf("invalid buyer ID")
	}

	if n.Quantity <= 0 {
		return fmt.Errorf(
			"negotiation quantity must be greater than zero",
		)
	}

	if n.InitialOfferPrice < 0 {
		return fmt.Errorf(
			"initial offer price cannot be negative",
		)
	}

	if n.CurrentOfferPrice < 0 {
		return fmt.Errorf(
			"current offer price cannot be negative",
		)
	}

	if n.MaxRounds <= 0 {
		return fmt.Errorf(
			"maximum negotiation rounds must be greater than zero",
		)
	}

	if n.ExpiresAt.IsZero() {
		return fmt.Errorf("negotiation expiry is required")
	}

	status := strings.ToLower(
		strings.TrimSpace(n.Status),
	)

	if status == "" {
		status = "open"
	}

	switch status {
	case "open", "accepted", "rejected", "expired", "closed":
		// valid
	default:
		return fmt.Errorf(
			"invalid negotiation status",
		)
	}

	query := `
		INSERT INTO negotiations (
			product_id,
			buyer_id,
			quantity,
			initial_offer_price,
			current_offer_price,
			round_count,
			max_rounds,
			status,
			expires_at
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
			$9
		)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		query,
		n.ProductID,
		n.BuyerID,
		n.Quantity,
		n.InitialOfferPrice,
		n.CurrentOfferPrice,
		n.RoundCount,
		n.MaxRounds,
		status,
		n.ExpiresAt,
	).Scan(
		&n.ID,
		&n.CreatedAt,
		&n.UpdatedAt,
	)

	if err != nil {
		return err
	}

	n.Status = status

	return nil
}

// ============================================================
// INCREMENT ROUND
// ============================================================

// IncrementRound increases the negotiation round count by one.
//
// The update is protected so a negotiation cannot continue
// beyond max_rounds.
func (r *NegotiationRepository) IncrementRound(
	id int,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid negotiation ID")
	}

	result, err := r.db.Exec(`
		UPDATE negotiations
		SET
			round_count = round_count + 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND round_count < max_rounds
			AND status = 'open'
			AND expires_at > CURRENT_TIMESTAMP
	`, id)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf(
			"negotiation cannot advance to another round",
		)
	}

	return nil
}

// ============================================================
// UPDATE STATUS
// ============================================================

// UpdateStatus changes the lifecycle status of a negotiation.
//
// accepted_at is automatically recorded when a negotiation is
// accepted.
//
// closed_at is automatically recorded when a negotiation is
// closed or rejected.
func (r *NegotiationRepository) UpdateStatus(
	id int,
	status string,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid negotiation ID")
	}

	status = strings.ToLower(
		strings.TrimSpace(status),
	)

	switch status {
	case "open", "accepted", "rejected", "expired", "closed":
		// valid
	default:
		return fmt.Errorf(
			"invalid negotiation status",
		)
	}

	query := `
		UPDATE negotiations
		SET
			status = $1,

			accepted_at = CASE
				WHEN $1 = 'accepted'
					THEN COALESCE(
						accepted_at,
						CURRENT_TIMESTAMP
					)
				ELSE accepted_at
			END,

			closed_at = CASE
				WHEN $1 IN (
					'rejected',
					'expired',
					'closed'
				)
					THEN COALESCE(
						closed_at,
						CURRENT_TIMESTAMP
					)
				ELSE closed_at
			END,

			updated_at = CURRENT_TIMESTAMP

		WHERE id = $2
	`

	result, err := r.db.Exec(
		query,
		status,
		id,
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
// GET BY ID
// ============================================================

// GetByID retrieves a negotiation together with:
//
//   - product information
//   - buyer information
//   - seller information
//
// Seller information is resolved through products.owner_type
// and products.owner_id.
func (r *NegotiationRepository) GetByID(
	id int,
) (*models.Negotiation, error) {
	if id <= 0 {
		return nil, fmt.Errorf(
			"invalid negotiation ID",
		)
	}

	query := `
		SELECT
			n.id,
			n.product_id,
			n.buyer_id,
			n.quantity,
			n.initial_offer_price,
			n.current_offer_price,
			n.status,
			n.round_count,
			n.max_rounds,
			n.expires_at,
			n.accepted_at,
			n.closed_at,
			n.created_at,
			n.updated_at,

			p.name,

			COALESCE(
				b.full_name,
				''
			) AS buyer_name,

			CASE
				WHEN p.owner_type = 'user'
					THEN COALESCE(
						s.full_name,
						''
					)

				WHEN p.owner_type = 'organization'
					THEN COALESCE(
						o.name,
						''
					)

				ELSE ''
			END AS seller_name,

			COALESCE(
				b.photo_url,
				''
			) AS buyer_photo_url,

			CASE
				WHEN p.owner_type = 'user'
					THEN COALESCE(
						s.photo_url,
						''
					)

				WHEN p.owner_type = 'organization'
					THEN COALESCE(
						o.logo_url,
						''
					)

				ELSE ''
			END AS seller_photo_url

		FROM negotiations n

		JOIN products p
			ON p.id = n.product_id

		LEFT JOIN users b
			ON b.id = n.buyer_id

		LEFT JOIN users s
			ON p.owner_type = 'user'
			AND s.id = p.owner_id

		LEFT JOIN organizations o
			ON p.owner_type = 'organization'
			AND o.id = p.owner_id

		WHERE n.id = $1
	`

	var n models.Negotiation

	err := r.db.QueryRow(
		query,
		id,
	).Scan(
		&n.ID,
		&n.ProductID,
		&n.BuyerID,
		&n.Quantity,
		&n.InitialOfferPrice,
		&n.CurrentOfferPrice,
		&n.Status,
		&n.RoundCount,
		&n.MaxRounds,
		&n.ExpiresAt,
		&n.AcceptedAt,
		&n.ClosedAt,
		&n.CreatedAt,
		&n.UpdatedAt,

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

// ============================================================
// LIST FOR USER
// ============================================================

// ListForUser returns negotiations relevant to a user.
//
// A user can participate in a negotiation in two ways:
//
//  1. The user is the buyer.
//  2. The user owns the negotiated product.
//
// Organization-owned products are also included when the user
// belongs to the organization through organization_members.
//
// This means the seller is always derived from the product.
func (r *NegotiationRepository) ListForUser(
	userID int,
) ([]models.Negotiation, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	query := `
		SELECT
			n.id,
			n.product_id,
			n.buyer_id,
			n.quantity,
			n.initial_offer_price,
			n.current_offer_price,
			n.status,
			n.round_count,
			n.max_rounds,
			n.expires_at,
			n.accepted_at,
			n.closed_at,
			n.created_at,
			n.updated_at,

			p.name,

			COALESCE(
				b.full_name,
				''
			) AS buyer_name,

			CASE
				WHEN p.owner_type = 'user'
					THEN COALESCE(
						s.full_name,
						''
					)

				WHEN p.owner_type = 'organization'
					THEN COALESCE(
						o.name,
						''
					)

				ELSE ''
			END AS seller_name,

			COALESCE(
				b.photo_url,
				''
			) AS buyer_photo_url,

			CASE
				WHEN p.owner_type = 'user'
					THEN COALESCE(
						s.photo_url,
						''
					)

				WHEN p.owner_type = 'organization'
					THEN COALESCE(
						o.logo_url,
						''
					)

				ELSE ''
			END AS seller_photo_url

		FROM negotiations n

		JOIN products p
			ON p.id = n.product_id

		LEFT JOIN users b
			ON b.id = n.buyer_id

		LEFT JOIN users s
			ON p.owner_type = 'user'
			AND s.id = p.owner_id

		LEFT JOIN organizations o
			ON p.owner_type = 'organization'
			AND o.id = p.owner_id

		WHERE
			n.buyer_id = $1

			OR (
				p.owner_type = 'user'
				AND p.owner_id = $1
			)

			OR (
				p.owner_type = 'organization'
				AND EXISTS (
					SELECT 1
					FROM organization_members om
					WHERE om.organization_id = p.owner_id
						AND om.user_id = $1
				)
			)

		ORDER BY
			n.created_at DESC,
			n.id DESC
	`

	rows, err := r.db.Query(
		query,
		userID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	negotiations := make(
		[]models.Negotiation,
		0,
	)

	for rows.Next() {
		var n models.Negotiation

		err := rows.Scan(
			&n.ID,
			&n.ProductID,
			&n.BuyerID,
			&n.Quantity,
			&n.InitialOfferPrice,
			&n.CurrentOfferPrice,
			&n.Status,
			&n.RoundCount,
			&n.MaxRounds,
			&n.ExpiresAt,
			&n.AcceptedAt,
			&n.ClosedAt,
			&n.CreatedAt,
			&n.UpdatedAt,

			&n.ProductName,
			&n.BuyerName,
			&n.SellerName,
			&n.BuyerPhotoURL,
			&n.SellerPhotoURL,
		)

		if err != nil {
			return nil, err
		}

		negotiations = append(
			negotiations,
			n,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return negotiations, nil
}