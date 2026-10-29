package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type SessionRepository struct{ db *sql.DB }

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, hash string, userID int, expires time.Time) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO user_sessions (token_hash, user_id, expires_at)
		SELECT $1, id, $3 FROM users WHERE id = $2 AND deleted_at IS NULL AND account_status='active' AND NOT staff_disabled`, hash, userID, expires)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *SessionRepository) UserID(ctx context.Context, hash string) (int, error) {
	var userID int
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM user_sessions
		WHERE token_hash = $1 AND expires_at > CURRENT_TIMESTAMP`, hash).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return userID, err
}

func (r *SessionRepository) Delete(ctx context.Context, hash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, hash)
	return err
}

func (r *SessionRepository) RevokeAll(ctx context.Context, userID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
	return err
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}
