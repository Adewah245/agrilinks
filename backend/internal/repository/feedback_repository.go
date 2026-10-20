package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
)

// ErrFeedbackRequiresOrder means the buyer has no order for this product and
// seller, so there is no verified purchase to review.
var ErrFeedbackRequiresOrder = errors.New("feedback requires a matching order")

// FeedbackRepository exposes the legacy buyer/product/farmer feedback API over
// the current product_reviews table.
//
// There is intentionally no separate feedback table. A product review already
// links the buyer, product, order, rating, and comment; the seller is resolved
// through products.owner_type and products.owner_id, which keeps one review
// system instead of duplicating the same records.
type FeedbackRepository struct {
	db *sql.DB
}

// NewFeedbackRepository creates a FeedbackRepository.
func NewFeedbackRepository(db *sql.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func validateFeedback(feedback *models.Feedback) error {
	if feedback == nil {
		return fmt.Errorf("feedback cannot be nil")
	}

	feedback.Comment = strings.TrimSpace(feedback.Comment)

	if feedback.UserID <= 0 {
		return fmt.Errorf("invalid feedback user")
	}
	if feedback.FarmerID <= 0 {
		return fmt.Errorf("invalid feedback seller")
	}
	if feedback.ProductID <= 0 {
		return fmt.Errorf("invalid feedback product")
	}
	if feedback.OrderID < 0 {
		return fmt.Errorf("invalid feedback order")
	}
	if feedback.Rating < 1 || feedback.Rating > 5 {
		return fmt.Errorf("feedback rating must be between 1 and 5")
	}

	return nil
}

// Create stores buyer feedback as a product review.
//
// When OrderID is omitted, the buyer's most recent order containing the product
// is used. The product must belong to the supplied seller, and the order must
// belong to the reviewer. These checks are part of the INSERT so a caller
// cannot attach feedback to somebody else's purchase.
func (r *FeedbackRepository) Create(feedback *models.Feedback) error {
	if err := validateFeedback(feedback); err != nil {
		return err
	}
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	query := `
		INSERT INTO product_reviews (
			product_id,
			reviewer_id,
			order_id,
			rating,
			comment
		)
		SELECT
			p.id,
			o.buyer_id,
			o.id,
			$4,
			NULLIF($5, '')
		FROM products p
		JOIN order_items oi
			ON oi.product_id = p.id
		JOIN orders o
			ON o.id = oi.order_id
		WHERE p.id = $3
		AND p.owner_type = 'user'
		AND p.owner_id = $2
		AND o.buyer_id = $1
		AND ($6 = 0 OR o.id = $6)
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT 1
		RETURNING id, order_id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		feedback.UserID,
		feedback.FarmerID,
		feedback.ProductID,
		feedback.Rating,
		feedback.Comment,
		feedback.OrderID,
	).Scan(
		&feedback.ID,
		&feedback.OrderID,
		&feedback.CreatedAt,
		&feedback.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFeedbackRequiresOrder
	}
	if err != nil {
		return fmt.Errorf("create feedback: %w", err)
	}

	return nil
}

// GetByFarmerID returns feedback for products owned by a user. Sellers are
// users in the current schema, so owner_type must be checked explicitly.
func (r *FeedbackRepository) GetByFarmerID(farmerID int) ([]models.Feedback, error) {
	if farmerID <= 0 {
		return []models.Feedback{}, nil
	}

	rows, err := r.db.Query(`
		SELECT
			pr.id,
			pr.reviewer_id,
			p.owner_id,
			pr.product_id,
			pr.order_id,
			pr.rating,
			COALESCE(pr.comment, ''),
			pr.created_at,
			pr.updated_at
		FROM product_reviews pr
		JOIN products p
			ON p.id = pr.product_id
		WHERE p.owner_type = 'user'
		AND p.owner_id = $1
		ORDER BY pr.created_at DESC, pr.id DESC
	`, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feedbacks := make([]models.Feedback, 0)
	for rows.Next() {
		var feedback models.Feedback
		if err := rows.Scan(
			&feedback.ID,
			&feedback.UserID,
			&feedback.FarmerID,
			&feedback.ProductID,
			&feedback.OrderID,
			&feedback.Rating,
			&feedback.Comment,
			&feedback.CreatedAt,
			&feedback.UpdatedAt,
		); err != nil {
			return nil, err
		}

		feedbacks = append(feedbacks, feedback)
	}

	return feedbacks, rows.Err()
}

// GetFarmerRating returns the average rating and number of product reviews for
// products owned by a user.
func (r *FeedbackRepository) GetFarmerRating(farmerID int) (float64, int, error) {
	if farmerID <= 0 {
		return 0, 0, nil
	}

	var average float64
	var total int

	err := r.db.QueryRow(`
		SELECT COALESCE(AVG(pr.rating), 0), COUNT(*)
		FROM product_reviews pr
		JOIN products p
			ON p.id = pr.product_id
		WHERE p.owner_type = 'user'
		AND p.owner_id = $1
	`, farmerID).Scan(&average, &total)
	if err != nil {
		return 0, 0, err
	}

	return average, total, nil
}
