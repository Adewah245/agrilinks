package repository

import (
	"backend/internal/models"
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrPrivateEvidence = errors.New("capture a fresh withdrawal selfie and location before submitting")
var ErrMFARequired = errors.New("verify your administrator authenticator before continuing")

type SecurityRepository struct{ db *sql.DB }

func NewSecurityRepository(db *sql.DB) *SecurityRepository { return &SecurityRepository{db: db} }

func (r *SecurityRepository) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `INSERT INTO security_rate_limits(key_hash) VALUES($1)
      ON CONFLICT(key_hash) DO UPDATE SET
      attempts=CASE WHEN security_rate_limits.started_at <= CURRENT_TIMESTAMP-($2*interval '1 second') THEN 1 ELSE LEAST(security_rate_limits.attempts+1,1000000) END,
      started_at=CASE WHEN security_rate_limits.started_at <= CURRENT_TIMESTAMP-($2*interval '1 second') THEN CURRENT_TIMESTAMP ELSE security_rate_limits.started_at END RETURNING attempts`, key, window.Seconds()).Scan(&count)
	return count <= limit, err
}
func (r *SecurityRepository) Event(ctx context.Context, userID int, kind, severity string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO security_events(user_id,kind,severity) VALUES(NULLIF($1,0),$2,$3)`, userID, kind, severity)
	if err != nil {
		return err
	}
	if kind == "wallet_action_failed" {
		_, err = r.db.ExecContext(ctx, `INSERT INTO security_events(user_id,kind,severity) SELECT $1,'repeated_wallet_failures','high' WHERE (SELECT count(*) FROM security_events WHERE user_id=$1 AND kind='wallet_action_failed' AND created_at>CURRENT_TIMESTAMP-interval '15 minutes')>=5 AND NOT EXISTS(SELECT 1 FROM security_events WHERE user_id=$1 AND kind='repeated_wallet_failures' AND created_at>CURRENT_TIMESTAMP-interval '15 minutes')`, userID)
	}
	return err
}
func (r *SecurityRepository) FreshMFA(ctx context.Context, userID int, session string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM user_sessions s JOIN admin_mfa m ON m.user_id=s.user_id WHERE s.user_id=$1 AND s.token_hash=$2 AND s.expires_at>CURRENT_TIMESTAMP AND s.mfa_verified_until>CURRENT_TIMESTAMP AND m.enabled)`, userID, session).Scan(&ok)
	return ok, err
}

type MFARecord struct {
	Secret            []byte
	ReplacementSecret []byte
	UseReplacement    bool
	Enabled           bool
	SetupSession      string
	SetupExpires      time.Time
	LastCounter       int64
}

func (r *SecurityRepository) MFA(ctx context.Context, id int) (*MFARecord, error) {
	m := &MFARecord{}
	err := r.db.QueryRowContext(ctx, `SELECT encrypted_secret,replacement_secret,enabled,setup_session_hash,setup_expires_at,last_counter FROM admin_mfa WHERE user_id=$1`, id).Scan(&m.Secret, &m.ReplacementSecret, &m.Enabled, &m.SetupSession, &m.SetupExpires, &m.LastCounter)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}
func (r *SecurityRepository) BeginMFA(ctx context.Context, id int, session string, secret []byte, replace bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var res sql.Result
	if replace {
		// Keep the active factor and recovery codes until the new factor has
		// been proven. An abandoned replacement must never disable MFA.
		res, err = tx.ExecContext(ctx, `UPDATE admin_mfa SET replacement_secret=$3,setup_session_hash=$2,setup_expires_at=CURRENT_TIMESTAMP+interval '10 minutes' WHERE user_id=$1 AND enabled AND EXISTS(SELECT 1 FROM user_sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>CURRENT_TIMESTAMP AND mfa_verified_until>CURRENT_TIMESTAMP)`, id, session, secret)
	} else {
		res, err = tx.ExecContext(ctx, `INSERT INTO admin_mfa(user_id,encrypted_secret,setup_session_hash,setup_expires_at) VALUES($1,$3,$2,CURRENT_TIMESTAMP+interval '10 minutes') ON CONFLICT(user_id) DO UPDATE SET encrypted_secret=$3,setup_session_hash=$2,setup_expires_at=CURRENT_TIMESTAMP+interval '10 minutes',last_counter=-1 WHERE NOT admin_mfa.enabled AND (admin_mfa.setup_session_hash=$2 OR admin_mfa.setup_expires_at<=CURRENT_TIMESTAMP)`, id, session, secret)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrMFARequired
	}
	return tx.Commit()
}
func (r *SecurityRepository) ConfirmMFA(ctx context.Context, id int, session string, m *MFARecord, counter int64, hashes []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var res sql.Result
	if m.UseReplacement {
		res, err = tx.ExecContext(ctx, `UPDATE admin_mfa SET encrypted_secret=replacement_secret,replacement_secret=NULL,last_counter=$4,enabled_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND enabled AND replacement_secret=$3 AND setup_session_hash=$2 AND setup_expires_at>CURRENT_TIMESTAMP`, id, session, m.ReplacementSecret, counter)
	} else {
		res, err = tx.ExecContext(ctx, `UPDATE admin_mfa SET enabled=true,last_counter=$4,enabled_at=COALESCE(enabled_at,CURRENT_TIMESTAMP) WHERE user_id=$1 AND encrypted_secret=$3 AND last_counter<$4 AND enabled=$5 AND (enabled OR (setup_session_hash=$2 AND setup_expires_at>CURRENT_TIMESTAMP))`, id, session, m.Secret, counter, m.Enabled)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrMFARequired
	}
	if m.UseReplacement {
		if _, err = tx.ExecContext(ctx, `DELETE FROM admin_mfa_recovery_codes WHERE user_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id=$1 AND token_hash<>$2`, id, session); err != nil {
			return err
		}
	}
	if !m.Enabled || m.UseReplacement {
		for _, hash := range hashes {
			if _, err = tx.ExecContext(ctx, `INSERT INTO admin_mfa_recovery_codes(user_id,code_hash) VALUES($1,$2)`, id, hash); err != nil {
				return err
			}
		}
	}
	if err = markMFASession(ctx, tx, id, session); err != nil {
		return err
	}
	return tx.Commit()
}
func markMFASession(ctx context.Context, tx *sql.Tx, id int, session string) error {
	res, err := tx.ExecContext(ctx, `UPDATE user_sessions SET mfa_verified_until=LEAST(expires_at,CURRENT_TIMESTAMP+interval '15 minutes') WHERE user_id=$1 AND token_hash=$2 AND expires_at>CURRENT_TIMESTAMP`, id, session)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrMFARequired
	}
	return nil
}
func (r *SecurityRepository) RecoverMFA(ctx context.Context, id int, session, hash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize recovery with authenticator replacement and code revocation.
	var enabled bool
	if err = tx.QueryRowContext(ctx, `SELECT enabled FROM admin_mfa WHERE user_id=$1 FOR UPDATE`, id).Scan(&enabled); err != nil || !enabled {
		return ErrMFARequired
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM admin_mfa_recovery_codes c WHERE c.user_id=$1 AND c.code_hash=$2 AND EXISTS(SELECT 1 FROM admin_mfa m WHERE m.user_id=c.user_id AND m.enabled)`, id, hash)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrMFARequired
	}
	if err = markMFASession(ctx, tx, id, session); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SecurityRepository) Challenge(ctx context.Context, userID int, session, quote, reference, token string) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO withdrawal_evidence(token_hash,user_id,session_hash,quote_id,reference,expires_at) SELECT $1,$2,$3,id,$5,LEAST(expires_at,CURRENT_TIMESTAMP+interval '10 minutes') FROM wallet_fee_quotes WHERE id=$4 AND user_id=$2 AND kind='withdrawal' AND expires_at>CURRENT_TIMESTAMP`, token, userID, session, quote, reference)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrPrivateEvidence
	}
	return nil
}
func (r *SecurityRepository) SaveEvidence(ctx context.Context, userID int, session, token string, image []byte, e models.WithdrawalEvidence) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var created time.Time
	if err = tx.QueryRowContext(ctx, `SELECT created_at FROM withdrawal_evidence WHERE token_hash=$1 AND user_id=$2 AND session_hash=$3 AND expires_at>CURRENT_TIMESTAMP AND photo_id IS NULL AND withdrawal_id IS NULL FOR UPDATE`, token, userID, session).Scan(&created); err != nil {
		return ErrPrivateEvidence
	}
	if e.CapturedAt.Before(created.Add(-30 * time.Second)) {
		return ErrPrivateEvidence
	}
	var photoID int64
	if err = tx.QueryRowContext(ctx, `INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'withdrawal',$2) RETURNING id`, userID, image).Scan(&photoID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE withdrawal_evidence SET photo_id=$2,latitude=$3,longitude=$4,accuracy_m=$5,captured_at=$6,received_at=CURRENT_TIMESTAMP WHERE token_hash=$1`, token, photoID, e.Latitude, e.Longitude, e.AccuracyM, e.CapturedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *SecurityRepository) Evidence(ctx context.Context, id int64) (*models.WithdrawalEvidence, error) {
	e := &models.WithdrawalEvidence{}
	err := r.db.QueryRowContext(ctx, `SELECT withdrawal_id,user_id,photo_id,latitude,longitude,accuracy_m,captured_at,received_at FROM withdrawal_evidence WHERE withdrawal_id=$1`, id).Scan(&e.WithdrawalID, &e.UserID, &e.PhotoID, &e.Latitude, &e.Longitude, &e.AccuracyM, &e.CapturedAt, &e.ReceivedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return e, err
}
func (r *SecurityRepository) ReviewEvidence(ctx context.Context, id int64, userID int, session string) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO withdrawal_evidence_reviews(withdrawal_id,reviewer_id,session_hash) SELECT e.withdrawal_id,$2,$3 FROM withdrawal_evidence e WHERE e.withdrawal_id=$1 AND e.photo_id IS NOT NULL AND e.user_id<>$2 AND EXISTS(SELECT 1 FROM user_sessions s JOIN admin_mfa m ON m.user_id=s.user_id WHERE s.user_id=$2 AND s.token_hash=$3 AND s.expires_at>CURRENT_TIMESTAMP AND s.mfa_verified_until>CURRENT_TIMESTAMP AND m.enabled) ON CONFLICT(withdrawal_id,reviewer_id,session_hash) DO UPDATE SET reviewed_at=CURRENT_TIMESTAMP`, id, userID, session)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrPrivateEvidence
	}
	return nil
}
func (r *SecurityRepository) Photo(ctx context.Context, id int64) (int, []byte, error) {
	var userID int
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT user_id,encrypted_image FROM private_verification_photos WHERE id=$1`, id).Scan(&userID, &data)
	return userID, data, err
}

func (r *SecurityRepository) MarkPhotoViewed(ctx context.Context, userID int, session string, photoID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE withdrawal_evidence_reviews v SET photo_viewed_at=CURRENT_TIMESTAMP FROM withdrawal_evidence e WHERE v.withdrawal_id=e.withdrawal_id AND e.photo_id=$3 AND v.reviewer_id=$1 AND v.session_hash=$2`, userID, session, photoID)
	return err
}
func (r *SecurityRepository) RegistrationPhoto(ctx context.Context, userID int) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM private_verification_photos WHERE user_id=$1 AND purpose IN ('registration','legacy') ORDER BY created_at DESC,id DESC LIMIT 1`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

type LegacyPrivatePhoto struct {
	Path    string
	UserID  int
	PhotoID sql.NullInt64
}

func (r *SecurityRepository) LegacyPhotos(ctx context.Context) ([]LegacyPrivatePhoto, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT public_path,user_id,photo_id FROM legacy_private_photo_paths WHERE migrated_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []LegacyPrivatePhoto
	for rows.Next() {
		var p LegacyPrivatePhoto
		if err = rows.Scan(&p.Path, &p.UserID, &p.PhotoID); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}
func (r *SecurityRepository) ImportLegacyPhoto(ctx context.Context, p LegacyPrivatePhoto, data []byte) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT photo_id FROM legacy_private_photo_paths WHERE public_path=$1 FOR UPDATE`, p.Path).Scan(&id); err != nil {
		return err
	}
	if !id.Valid {
		if err = tx.QueryRowContext(ctx, `INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'legacy',$2) RETURNING id`, p.UserID, data).Scan(&id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE legacy_private_photo_paths SET photo_id=$2 WHERE public_path=$1`, p.Path, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *SecurityRepository) MarkLegacyPrivate(ctx context.Context, path string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE legacy_private_photo_paths SET migrated_at=CURRENT_TIMESTAMP WHERE public_path=$1`, path)
	return err
}

// Unsubmitted captures have no financial record to retain. Remove them after
// expiry. Completed withdrawal evidence is retained for 90 days; pending and
// uncertain payouts remain available for investigation.
func (r *SecurityRepository) Cleanup(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM withdrawal_evidence WHERE (withdrawal_id IS NULL AND expires_at<CURRENT_TIMESTAMP-interval '1 hour') OR withdrawal_id IN (SELECT id FROM wallet_withdrawals WHERE status IN ('paid','failed','reversed','rejected','cancelled') AND updated_at<CURRENT_TIMESTAMP-interval '90 days')`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM private_verification_photos p WHERE purpose='withdrawal' AND NOT EXISTS(SELECT 1 FROM withdrawal_evidence e WHERE e.photo_id=p.id)`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM security_rate_limits WHERE started_at<CURRENT_TIMESTAMP-interval '1 day'`); err != nil {
		return err
	}
	return tx.Commit()
}
