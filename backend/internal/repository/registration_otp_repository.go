package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend/internal/models"
)

var (
	ErrRegistrationOTPInvalid  = errors.New("the confirmation code is incorrect, expired, or already used; request a new code if needed")
	ErrRegistrationOTPCooldown = errors.New("please wait before requesting another confirmation code")
)

type RegistrationOTPRepository struct{ db *sql.DB }

func NewRegistrationOTPRepository(db *sql.DB) *RegistrationOTPRepository {
	return &RegistrationOTPRepository{db: db}
}

func (r *RegistrationOTPRepository) Create(ctx context.Context, code *models.RegistrationPhoneCode, ttl, cooldown time.Duration) error {
	// Reclaim abandoned challenges; no account or wallet exists for them.
	if _, err := r.db.ExecContext(ctx, `DELETE FROM registration_phone_codes WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '1 day'`); err != nil {
		return err
	}
	// The unique phone row makes cooldown and replacement atomic across browsers.
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO registration_phone_codes(phone,token_hash,code_hash,expires_at)
		VALUES($1,$2,$3,CURRENT_TIMESTAMP + $4 * INTERVAL '1 second')
		ON CONFLICT(phone) DO UPDATE SET
		    token_hash=EXCLUDED.token_hash, code_hash=EXCLUDED.code_hash,
		    attempts=0, created_at=CURRENT_TIMESTAMP,
		    expires_at=EXCLUDED.expires_at, consumed_at=NULL
		WHERE registration_phone_codes.created_at <= CURRENT_TIMESTAMP - $5 * INTERVAL '1 second'
		RETURNING expires_at`, code.Phone, code.TokenHash, code.CodeHash, ttl.Seconds(), cooldown.Seconds()).Scan(&code.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRegistrationOTPCooldown
	}
	return err
}

// Attempt reserves one attempt before comparing a hash so simultaneous requests
// cannot each use the last remaining attempt.
func (r *RegistrationOTPRepository) Attempt(ctx context.Context, tokenHash, phone string, maxAttempts int) (*models.RegistrationPhoneCode, error) {
	code := &models.RegistrationPhoneCode{TokenHash: tokenHash, Phone: phone}
	err := r.db.QueryRowContext(ctx, `
		UPDATE registration_phone_codes SET attempts=attempts+1
		WHERE token_hash=$1 AND phone=$2 AND consumed_at IS NULL
		  AND expires_at > CURRENT_TIMESTAMP AND attempts < $3
		RETURNING code_hash,attempts,expires_at`, tokenHash, phone, maxAttempts).Scan(&code.CodeHash, &code.Attempts, &code.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRegistrationOTPInvalid
	}
	return code, err
}

func (r *RegistrationOTPRepository) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM registration_phone_codes WHERE token_hash=$1`, tokenHash)
	return err
}

// Code consumption and account creation commit together. Resends, expiry,
// replay, or a changed phone invalidate a previously checked challenge.
func consumeRegistrationPhone(tx *sql.Tx, phone string, proof *models.RegistrationPhoneCode) error {
	if proof == nil || proof.Phone != phone || proof.Attempts <= 0 {
		return ErrRegistrationOTPInvalid
	}
	result, err := tx.Exec(`UPDATE registration_phone_codes SET consumed_at=CURRENT_TIMESTAMP
		WHERE phone=$1 AND token_hash=$2 AND code_hash=$3 AND consumed_at IS NULL
		AND expires_at > CURRENT_TIMESTAMP`, phone, proof.TokenHash, proof.CodeHash)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrRegistrationOTPInvalid
	}
	return nil
}
