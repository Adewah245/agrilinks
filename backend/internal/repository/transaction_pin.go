package repository

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrTransactionPINRequired = errors.New("create your transaction PIN before moving money")
	ErrTransactionPINInvalid  = errors.New("the transaction PIN is incorrect")
	ErrTransactionPINLocked   = errors.New("too many incorrect PIN attempts; try again in 15 minutes")
	ErrTransactionPINExists   = errors.New("a transaction PIN has already been created for this account")
)

func (r *UserRepository) HasTransactionPIN(ctx context.Context, userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM user_transaction_pins WHERE user_id=$1)`, userID).Scan(&exists)
	return exists, err
}

func (r *UserRepository) CreateTransactionPIN(ctx context.Context, userID int, hash string) error {
	if _, err := bcrypt.Cost([]byte(hash)); err != nil {
		return errors.New("could not secure the transaction PIN")
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO user_transaction_pins(user_id,pin_hash) VALUES($1,$2) ON CONFLICT(user_id) DO NOTHING`, userID, hash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrTransactionPINExists
	}
	return nil
}

// Hash comparison and attempt accounting share a row lock. Concurrent callers
// cannot each use the last available attempt or reset another failed attempt.
func (r *UserRepository) VerifyTransactionPIN(ctx context.Context, userID int, pin string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var hash string
	var attempts int
	var locked bool
	err = tx.QueryRowContext(ctx, `SELECT pin_hash,CASE WHEN blocked_until <= CURRENT_TIMESTAMP THEN 0 ELSE failed_attempts END,
        COALESCE(blocked_until > CURRENT_TIMESTAMP,false) FROM user_transaction_pins WHERE user_id=$1 FOR UPDATE`, userID).Scan(&hash, &attempts, &locked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTransactionPINRequired
	}
	if err != nil {
		return err
	}
	if locked {
		return ErrTransactionPINLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) != nil {
		attempts++
		_, err = tx.ExecContext(ctx, `UPDATE user_transaction_pins SET failed_attempts=$2,
            blocked_until=CASE WHEN $2>=5 THEN CURRENT_TIMESTAMP+interval '15 minutes' ELSE NULL END WHERE user_id=$1`, userID, attempts)
		if err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		if attempts >= 5 {
			return ErrTransactionPINLocked
		}
		return ErrTransactionPINInvalid
	}
	if _, err = tx.ExecContext(ctx, `UPDATE user_transaction_pins SET failed_attempts=0,blocked_until=NULL WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}
