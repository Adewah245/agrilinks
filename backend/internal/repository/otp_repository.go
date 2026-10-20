package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/models"
)

// OTPRepository persists hashed one-time passcodes.
type OTPRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// Create stores a new OTP without its plaintext code.
func (r *OTPRepository) Create(otp *models.OTPCode) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	if otp == nil {
		return errors.New("OTP cannot be nil")
	}

	err := r.db.QueryRow(`
		INSERT INTO otp_codes (
			user_id,
			phone,
			purpose,
			code_hash,
			attempts,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`,
		otp.UserID,
		otp.Phone,
		otp.Purpose,
		otp.CodeHash,
		otp.Attempts,
		otp.ExpiresAt,
	).Scan(&otp.ID, &otp.CreatedAt)
	if err != nil {
		return fmt.Errorf("create OTP: %w", err)
	}

	return nil
}

// LatestActive returns the newest unconsumed OTP for a user and purpose.
func (r *OTPRepository) LatestActive(userID int, purpose string) (*models.OTPCode, error) {
	if r == nil || r.db == nil {
		return nil, sql.ErrConnDone
	}

	var otp models.OTPCode
	err := r.db.QueryRow(`
		SELECT
			id,
			user_id,
			phone,
			purpose,
			code_hash,
			attempts,
			expires_at,
			consumed_at,
			created_at
		FROM otp_codes
		WHERE user_id = $1
		  AND purpose = $2
		  AND consumed_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, userID, purpose).Scan(
		&otp.ID,
		&otp.UserID,
		&otp.Phone,
		&otp.Purpose,
		&otp.CodeHash,
		&otp.Attempts,
		&otp.ExpiresAt,
		&otp.ConsumedAt,
		&otp.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active OTP: %w", err)
	}

	return &otp, nil
}

// IncrementAttempts records a failed verification attempt.
func (r *OTPRepository) IncrementAttempts(id int64) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	_, err := r.db.Exec(`
		UPDATE otp_codes
		SET attempts = attempts + 1
		WHERE id = $1
		  AND consumed_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("increment OTP attempts: %w", err)
	}

	return nil
}

// Consume marks an OTP as used exactly once.
func (r *OTPRepository) Consume(id int64) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	_, err := r.db.Exec(`
		UPDATE otp_codes
		SET consumed_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND consumed_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("consume OTP: %w", err)
	}

	return nil
}

// Delete removes an OTP when delivery fails before the code reaches the user.
func (r *OTPRepository) Delete(id int64) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	_, err := r.db.Exec(`DELETE FROM otp_codes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete OTP: %w", err)
	}

	return nil
}
