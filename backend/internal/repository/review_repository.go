package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
)

type ReviewRepository struct{ db *sql.DB }

func NewReviewRepository(db *sql.DB) *ReviewRepository { return &ReviewRepository{db} }
func (r *ReviewRepository) CreateProductReview(v *models.ProductReview) error {
	return r.db.QueryRow(`INSERT INTO product_reviews(product_id,reviewer_id,order_id,rating,comment) SELECT $1,$2,$3,$4,NULLIF($5,'') WHERE EXISTS(SELECT 1 FROM orders o JOIN order_items i ON i.order_id=o.id WHERE o.id=$3 AND o.buyer_id=$2 AND i.product_id=$1) RETURNING id,created_at,updated_at`, v.ProductID, v.ReviewerID, v.OrderID, v.Rating, v.Comment).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}
func (r *ReviewRepository) ListProductReviews(productID int) ([]models.ProductReview, error) {
	rows, e := r.db.Query(`SELECT id,product_id,reviewer_id,order_id,rating,COALESCE(comment,''),created_at,updated_at FROM product_reviews WHERE product_id=$1 ORDER BY created_at DESC`, productID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []models.ProductReview
	for rows.Next() {
		var v models.ProductReview
		if e = rows.Scan(&v.ID, &v.ProductID, &v.ReviewerID, &v.OrderID, &v.Rating, &v.Comment, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *ReviewRepository) MyVerification(userID int) (*models.Verification, error) {
	v := &models.Verification{}
	e := r.db.QueryRow(`SELECT id,user_id,COALESCE(nin,''),nin_verified,COALESCE(nin_verification_reference,''),verification_level,verified_at,created_at FROM user_verifications WHERE user_id=$1`, userID).Scan(&v.ID, &v.UserID, &v.NIN, &v.NINVerified, &v.Reference, &v.Level, &v.VerifiedAt, &v.CreatedAt)
	if e == sql.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, fmt.Errorf("verification: %w", e)
	}
	return v, nil
}
