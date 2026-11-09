package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var ErrWalletConfirmation = errors.New("this confirmation expired or changed; start again with your PIN")
var ErrConfirmationCode = errors.New("invalid OTP code")
var ErrConfirmationAttempts = errors.New("too many OTP attempts; start again with your PIN")
var ErrConfirmationCooldown = errors.New("please wait before requesting another code")
var ErrConfirmationCompleted = errors.New("this request is already submitted; check recent activity")

const confirmationColumns = `token_hash,user_id,session_hash,kind,currency,amount::text,recipient_wallet,note,quote_id,bank_code,account_number,reference,otp_hash,max_attempts,attempts,expires_at,created_at,otp_sent_at,verified_at,invalidated_at,final_reference`

func scanConfirmation(row interface{ Scan(...any) error }) (*models.WalletConfirmation, error) {
	c := &models.WalletConfirmation{}
	err := row.Scan(&c.TokenHash, &c.UserID, &c.SessionHash, &c.Kind, &c.Currency, &c.Amount, &c.RecipientWallet, &c.Note, &c.QuoteID, &c.BankCode, &c.AccountNumber, &c.Reference, &c.OTPHash, &c.MaxAttempts, &c.Attempts, &c.ExpiresAt, &c.CreatedAt, &c.SentAt, &c.VerifiedAt, &c.InvalidatedAt, &c.FinalReference)
	return c, err
}
func activeConfirmationSession(ctx context.Context, tx *sql.Tx, userID int, session string, exclusive bool) error {
	lock := " FOR SHARE OF u,s"
	if exclusive {
		lock = " FOR UPDATE OF u FOR SHARE OF s"
	}
	var id int
	err := tx.QueryRowContext(ctx, `SELECT u.id FROM users u JOIN user_sessions s ON s.user_id=u.id WHERE u.id=$1 AND s.token_hash=$2 AND s.expires_at>CURRENT_TIMESTAMP AND u.account_status='active' AND NOT u.staff_disabled AND u.deleted_at IS NULL`+lock, userID, session).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWalletConfirmation
	}
	return err
}
func (r *WalletRepository) Confirmation(ctx context.Context, userID int, session, token string) (*models.WalletConfirmation, error) {
	c, err := scanConfirmation(r.db.QueryRowContext(ctx, `SELECT `+confirmationColumns+` FROM wallet_confirmations WHERE token_hash=$1 AND user_id=$2 AND session_hash=$3`, token, userID, session))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWalletConfirmation
	}
	return c, err
}
func sameConfirmation(a, b *models.WalletConfirmation) bool {
	return a.UserID == b.UserID && a.SessionHash == b.SessionHash && a.Kind == b.Kind && a.Currency == b.Currency && a.Amount == b.Amount && a.RecipientWallet == b.RecipientWallet && a.Note == b.Note && a.QuoteID == b.QuoteID && a.BankCode == b.BankCode && a.AccountNumber == b.AccountNumber
}
func (r *WalletRepository) CreateConfirmation(ctx context.Context, c *models.WalletConfirmation, cooldown time.Duration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = activeConfirmationSession(ctx, tx, c.UserID, c.SessionHash, true); err != nil {
		return err
	}
	prior, err := scanConfirmation(tx.QueryRowContext(ctx, `SELECT `+confirmationColumns+` FROM wallet_confirmations WHERE reference=$1 FOR UPDATE`, c.Reference))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		if !sameConfirmation(prior, c) {
			return ErrWalletRequestChanged
		}
		if prior.FinalReference.Valid {
			return ErrConfirmationCompleted
		}
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_transfers WHERE reference=$1 UNION ALL SELECT 1 FROM wallet_withdrawals WHERE reference=$1)`, c.Reference).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrConfirmationCompleted
	}
	var recent bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_confirmations WHERE user_id=$1 AND kind=$2 AND invalidated_at IS NULL AND final_reference IS NULL AND created_at>CURRENT_TIMESTAMP-$3*interval '1 second')`, c.UserID, c.Kind, cooldown.Seconds()).Scan(&recent); err != nil {
		return err
	}
	if recent {
		return ErrConfirmationCooldown
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_confirmations SET invalidated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND kind=$2 AND final_reference IS NULL AND invalidated_at IS NULL AND reference<>$3`, c.UserID, c.Kind, c.Reference); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_confirmations(token_hash,user_id,session_hash,kind,currency,amount,recipient_wallet,note,quote_id,bank_code,account_number,reference,otp_hash,max_attempts,expires_at)
 VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10,$11,$12,$13,$14,$15)
 ON CONFLICT(reference) DO UPDATE SET token_hash=EXCLUDED.token_hash,otp_hash=EXCLUDED.otp_hash,max_attempts=EXCLUDED.max_attempts,attempts=0,expires_at=EXCLUDED.expires_at,created_at=CURRENT_TIMESTAMP,otp_sent_at=NULL,verified_at=NULL,invalidated_at=NULL`, c.TokenHash, c.UserID, c.SessionHash, c.Kind, c.Currency, c.Amount, c.RecipientWallet, c.Note, c.QuoteID, c.BankCode, c.AccountNumber, c.Reference, c.OTPHash, c.MaxAttempts, c.ExpiresAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *WalletRepository) ConfirmationDelivered(ctx context.Context, token string, success bool) error {
	column := "otp_sent_at"
	if !success {
		column = "invalidated_at"
	}
	res, err := r.db.ExecContext(ctx, `UPDATE wallet_confirmations SET `+column+`=CURRENT_TIMESTAMP WHERE token_hash=$1 AND invalidated_at IS NULL AND expires_at>CURRENT_TIMESTAMP`, token)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrWalletConfirmation
	}
	return nil
}
func usableConfirmation(c *models.WalletConfirmation) bool {
	return !c.InvalidatedAt.Valid && time.Now().Before(c.ExpiresAt) && c.SentAt.Valid && c.Attempts < c.MaxAttempts
}
func (r *WalletRepository) VerifyConfirmation(ctx context.Context, userID int, session, token, code string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = activeConfirmationSession(ctx, tx, userID, session, false); err != nil {
		return err
	}
	c, err := scanConfirmation(tx.QueryRowContext(ctx, `SELECT `+confirmationColumns+` FROM wallet_confirmations WHERE token_hash=$1 AND user_id=$2 AND session_hash=$3 FOR UPDATE`, token, userID, session))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWalletConfirmation
	}
	if err != nil {
		return err
	}
	if c.Attempts >= c.MaxAttempts {
		return ErrConfirmationAttempts
	}
	if !usableConfirmation(c) || c.FinalReference.Valid {
		return ErrWalletConfirmation
	}
	if c.VerifiedAt.Valid {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(c.OTPHash), []byte(code)) != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE wallet_confirmations SET attempts=attempts+1 WHERE token_hash=$1`, token); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		return ErrConfirmationCode
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_confirmations SET verified_at=CURRENT_TIMESTAMP WHERE token_hash=$1`, token); err != nil {
		return err
	}
	return tx.Commit()
}
func confirmationResult(c *models.WalletConfirmation) *models.WalletConfirmationResult {
	status := "completed"
	if c.Kind == "withdrawal" {
		status = "pending"
	}
	return &models.WalletConfirmationResult{Kind: c.Kind, Status: status, Reference: c.Reference}
}
func (r *WalletRepository) AvailableConfirmationBalance(ctx context.Context, userID int, amount string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency=$2 AND recovery_debt_kobo=0 AND balance>=$3::numeric AND balance*100=COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||wallets.id AND p.currency=wallets.currency),0))`, userID, r.currency, amount).Scan(&ok)
	return ok, err
}
func (r *WalletRepository) CompleteConfirmation(ctx context.Context, userID int, session, token string, image []byte, e models.WithdrawalEvidence, quote *models.FeeQuote) (*models.WalletConfirmationResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = financeLock(ctx, tx); err != nil {
		return nil, err
	}
	if err = activeConfirmationSession(ctx, tx, userID, session, false); err != nil {
		return nil, err
	}
	c, err := scanConfirmation(tx.QueryRowContext(ctx, `SELECT `+confirmationColumns+` FROM wallet_confirmations WHERE token_hash=$1 AND user_id=$2 AND session_hash=$3 FOR UPDATE`, token, userID, session))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWalletConfirmation
	}
	if err != nil {
		return nil, err
	}
	if c.FinalReference.Valid {
		return confirmationResult(c), nil
	}
	if !usableConfirmation(c) || !c.VerifiedAt.Valid || e.CapturedAt.Before(c.VerifiedAt.Time.Add(-30*time.Second)) || len(image) == 0 {
		return nil, ErrWalletConfirmation
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_transfers WHERE reference=$1 UNION ALL SELECT 1 FROM wallet_withdrawals WHERE reference=$1)`, c.Reference).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWalletRequestChanged
	}
	var photoID int64
	if err = tx.QueryRowContext(ctx, `INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,$2,$3) RETURNING id`, userID, c.Kind, image).Scan(&photoID); err != nil {
		return nil, err
	}
	if c.Kind == "transfer" {
		if err = r.requestTransfer(ctx, tx, userID, c.RecipientWallet, c.Amount, c.Reference, c.Note, true); err != nil {
			return nil, err
		}
	} else {
		if quote == nil || quote.ID != c.QuoteID || quote.Amount() != c.Amount || quote.BankCode != c.BankCode || quote.AccountNumber != c.AccountNumber {
			return nil, ErrWalletRequestChanged
		}
		var bank string
		if err = tx.QueryRowContext(ctx, `SELECT bank_code FROM user_bank_accounts WHERE user_id=$1 AND bank_code=$2 AND account_number=$3 AND account_name=$4 FOR SHARE`, userID, c.BankCode, c.AccountNumber, quote.AccountName).Scan(&bank); err != nil {
			return nil, ErrWalletRequestChanged
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO withdrawal_evidence(token_hash,user_id,session_hash,quote_id,reference,expires_at,photo_id,latitude,longitude,accuracy_m,captured_at,received_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,CURRENT_TIMESTAMP)`, token, userID, session, c.QuoteID, c.Reference, c.ExpiresAt, photoID, e.Latitude, e.Longitude, e.AccuracyM, e.CapturedAt); err != nil {
			return nil, err
		}
		account := &models.UserBankAccount{UserID: userID, BankName: quote.BankName, BankCode: quote.BankCode, AccountName: quote.AccountName, AccountNumber: quote.AccountNumber}
		if err = r.requestWithdrawal(ctx, tx, userID, c.Amount, c.Reference, account, quote, models.WithdrawalAuthorization{TokenHash: token, SessionHash: session}); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_confirmations SET photo_id=$2,latitude=$3,longitude=$4,accuracy_m=$5,captured_at=$6,received_at=CURRENT_TIMESTAMP,final_reference=reference,completed_at=CURRENT_TIMESTAMP WHERE token_hash=$1`, token, photoID, e.Latitude, e.Longitude, e.AccuracyM, e.CapturedAt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return confirmationResult(c), nil
}
