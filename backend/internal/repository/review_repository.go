package repository

import (
	"backend/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrReviewRequiresDeliveredOrder = errors.New("reviews are available after delivery")
	ErrReviewAlreadyExists          = errors.New("this purchase has already been reviewed")
	ErrInvalidBadgeType             = errors.New("invalid trust badge type")
	ErrBadgeOwnerNotFound           = errors.New("trust badge owner not found")
)

type ReviewRepository struct{ db *sql.DB }

func NewReviewRepository(db *sql.DB) *ReviewRepository { return &ReviewRepository{db: db} }

// CreateProductReview stores buyer feedback only after the matching order has
// been delivered. The ownership and delivery checks are part of the INSERT, so
// a caller cannot attach a review to somebody else's purchase or an incomplete
// order.
func (r *ReviewRepository) CreateProductReview(review *models.ProductReview) error {
	err := r.db.QueryRow(`
		INSERT INTO product_reviews (product_id, reviewer_id, order_id, rating, comment)
		SELECT $1, $2, $3, $4, NULLIF($5, '')
		WHERE EXISTS (
			SELECT 1
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			WHERE o.id = $3
			  AND o.buyer_id = $2
			  AND oi.product_id = $1
			  AND o.status = 'delivered'
		)
		RETURNING id, created_at, updated_at
	`,
		review.ProductID,
		review.ReviewerID,
		review.OrderID,
		review.Rating,
		review.Comment,
	).Scan(&review.ID, &review.CreatedAt, &review.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrReviewRequiresDeliveredOrder
	}
	if uniqueViolationConstraint(err) == "unique_product_order_reviewer" {
		return ErrReviewAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create product review: %w", err)
	}
	return nil
}

// ListProductReviews returns product feedback with the reviewer's display name
// resolved for the product page.
func (r *ReviewRepository) ListProductReviews(
	productID int,
) ([]models.ProductReview, error) {
	rows, err := r.db.Query(`
		SELECT
			pr.id,
			pr.product_id,
			pr.reviewer_id,
			COALESCE(u.full_name, 'Buyer') AS reviewer_name,
			pr.order_id,
			pr.rating,
			COALESCE(pr.comment, ''),
			pr.created_at,
			pr.updated_at
		FROM product_reviews pr
		LEFT JOIN users u ON u.id = pr.reviewer_id
		WHERE pr.product_id = $1
		ORDER BY pr.created_at DESC, pr.id DESC
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("list product reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]models.ProductReview, 0)
	for rows.Next() {
		var review models.ProductReview
		if err := rows.Scan(
			&review.ID,
			&review.ProductID,
			&review.ReviewerID,
			&review.ReviewerName,
			&review.OrderID,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
			&review.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product reviews: %w", err)
	}
	return reviews, nil
}

// ProductRating returns the average product rating and number of reviews.
func (r *ReviewRepository) ProductRating(productID int) (float64, int, error) {
	if productID <= 0 {
		return 0, 0, fmt.Errorf("invalid product")
	}

	var average float64
	var count int
	err := r.db.QueryRow(`
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM product_reviews
		WHERE product_id = $1
	`, productID).Scan(&average, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("get product rating: %w", err)
	}
	return average, count, nil
}

// SellerRating aggregates product reviews for products owned by a user or
// organization. It is public reputation information, not private verification.
func (r *ReviewRepository) SellerRating(
	ownerType string,
	ownerID int,
) (float64, int, error) {
	ownerType, err := normalizeOwnerType(ownerType)
	if err != nil {
		return 0, 0, err
	}
	if ownerID <= 0 {
		return 0, 0, fmt.Errorf("invalid seller")
	}

	var average float64
	var count int
	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(pr.rating), 0), COUNT(*)
		FROM product_reviews pr
		JOIN products p ON p.id = pr.product_id
		WHERE p.owner_type = $1 AND p.owner_id = $2
	`, ownerType, ownerID).Scan(&average, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("get seller rating: %w", err)
	}
	return average, count, nil
}

// ActiveBadges returns trust badges that are current and publicly displayable.
func (r *ReviewRepository) ActiveBadges(
	ownerType string,
	ownerID int,
) ([]models.VerificationBadge, error) {
	ownerType, err := normalizeOwnerType(ownerType)
	if err != nil {
		return nil, err
	}
	if ownerID <= 0 {
		return nil, fmt.Errorf("invalid badge owner")
	}

	rows, err := r.db.Query(`
		SELECT id, owner_type, owner_id, badge_type, issued_by,
		       COALESCE(description, ''), issued_at, expires_at, active
		FROM verification_badges
		WHERE owner_type = $1
		  AND owner_id = $2
		  AND active = TRUE
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY issued_at DESC, id DESC
	`, ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list trust badges: %w", err)
	}
	defer rows.Close()

	badges := make([]models.VerificationBadge, 0)
	for rows.Next() {
		var badge models.VerificationBadge
		var issuedBy sql.NullInt64
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&badge.ID,
			&badge.OwnerType,
			&badge.OwnerID,
			&badge.BadgeType,
			&issuedBy,
			&badge.Description,
			&badge.IssuedAt,
			&expiresAt,
			&badge.Active,
		); err != nil {
			return nil, fmt.Errorf("scan trust badge: %w", err)
		}
		badge.IssuedBy = intPointer(issuedBy)
		badge.ExpiresAt = timePointer(expiresAt)
		badges = append(badges, badge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trust badges: %w", err)
	}
	return badges, nil
}

// IssueBadge records a manual trust decision. Issuing a replacement badge
// deactivates the previous active badge of the same type in one transaction.
func (r *ReviewRepository) IssueBadge(
	badge *models.VerificationBadge,
	actorID int,
) error {
	if badge == nil {
		return fmt.Errorf("badge cannot be nil")
	}
	if actorID <= 0 {
		return fmt.Errorf("invalid badge issuer")
	}

	ownerType, err := normalizeOwnerType(badge.OwnerType)
	if err != nil {
		return err
	}
	if badge.OwnerID <= 0 {
		return fmt.Errorf("invalid badge owner")
	}
	badge.BadgeType = strings.TrimSpace(badge.BadgeType)
	if !validBadgeType(badge.BadgeType) {
		return ErrInvalidBadgeType
	}
	if badge.ExpiresAt != nil && badge.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("a badge cannot already be expired")
	}
	badge.Description = strings.TrimSpace(badge.Description)

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin badge issue: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var ownerExists bool
	switch ownerType {
	case "user":
		err = tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, badge.OwnerID).Scan(&ownerExists)
	case "organization":
		err = tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM organizations WHERE id = $1)`, badge.OwnerID).Scan(&ownerExists)
	}
	if err != nil {
		return fmt.Errorf("check trust badge owner: %w", err)
	}
	if !ownerExists {
		return ErrBadgeOwnerNotFound
	}

	if _, err := tx.Exec(`
		UPDATE verification_badges
		SET active = FALSE
		WHERE owner_type = $1
		  AND owner_id = $2
		  AND badge_type = $3
		  AND active = TRUE
	`, ownerType, badge.OwnerID, badge.BadgeType); err != nil {
		return fmt.Errorf("replace active badge: %w", err)
	}

	err = tx.QueryRow(`
		INSERT INTO verification_badges (
			owner_type,
			owner_id,
			badge_type,
			issued_by,
			description,
			expires_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		RETURNING id, issued_at, active
	`,
		ownerType,
		badge.OwnerID,
		badge.BadgeType,
		actorID,
		badge.Description,
		nullableTime(badge.ExpiresAt),
	).Scan(&badge.ID, &badge.IssuedAt, &badge.Active)
	if err != nil {
		return fmt.Errorf("issue trust badge: %w", err)
	}
	badge.OwnerType = ownerType
	badge.IssuedBy = &actorID

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit trust badge: %w", err)
	}
	return nil
}

// RevokeBadge removes a badge from public display while preserving its audit
// history.
func (r *ReviewRepository) RevokeBadge(badgeID int) error {
	if badgeID <= 0 {
		return fmt.Errorf("invalid badge")
	}
	result, err := r.db.Exec(`
		UPDATE verification_badges
		SET active = FALSE
		WHERE id = $1 AND active = TRUE
	`, badgeID)
	if err != nil {
		return fmt.Errorf("revoke trust badge: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ReviewRepository) MyVerification(userID int) (*models.Verification, error) {
	verification := &models.Verification{}
	err := r.db.QueryRow(`
		SELECT id, user_id, COALESCE(nin, ''), nin_verified,
		       COALESCE(nin_verification_reference, ''), verification_level,
		       verified_at, created_at
		FROM user_verifications
		WHERE user_id = $1
	`, userID).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.NIN,
		&verification.NINVerified,
		&verification.Reference,
		&verification.Level,
		&verification.VerifiedAt,
		&verification.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("verification: %w", err)
	}
	return verification, nil
}

func validBadgeType(badgeType string) bool {
	switch strings.TrimSpace(badgeType) {
	case "verified_farmer", "verified_organization", "government_partner", "trusted_seller":
		return true
	default:
		return false
	}
}
