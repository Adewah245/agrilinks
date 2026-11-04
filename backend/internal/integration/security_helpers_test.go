package integration_test

import (
	"backend/internal/app"
	"backend/internal/models"
	"backend/internal/services"
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/jpeg"
	"strconv"
	"testing"
	"time"
)

func verificationJPEG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 16, 16)), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// Existing workflow fixtures start with an MFA-verified staff session. Security
// tests create raw sessions separately and exercise enrollment and verification.
func authorizeFixtureAdmin(t *testing.T, c *app.Container, id int, token string) {
	t.Helper()
	ctx := context.Background()
	u, err := c.UserRepo.GetByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if u.AdminRole() == "" {
		return
	}
	m, err := c.Security.Repo.MFA(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		secret, err := c.Security.Seal([]byte("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"), "admin-mfa:"+strconv.Itoa(id))
		if err != nil {
			t.Fatal(err)
		}
		if err = c.Security.Repo.BeginMFA(ctx, id, services.SecurityHash(token), secret, false); err != nil {
			t.Fatal(err)
		}
		m, err = c.Security.Repo.MFA(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = c.Security.Repo.ConfirmMFA(ctx, id, services.SecurityHash(token), m, m.LastCounter+1, nil); err != nil {
		t.Fatal(err)
	}
}

func fixtureEvidence(t *testing.T, db *sql.DB, userID int, reference string) models.WithdrawalAuthorization {
	t.Helper()
	hash := services.SecurityHash(reference)
	var photoID int64
	if err := db.QueryRow(`INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'withdrawal',$2) RETURNING id`, userID, []byte("fixture ciphertext")).Scan(&photoID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO withdrawal_evidence(token_hash,user_id,session_hash,quote_id,reference,expires_at,photo_id,latitude,longitude,accuracy_m,captured_at,received_at) VALUES($1,$2,$1,'',$3,CURRENT_TIMESTAMP+interval '10 minutes',$4,7.7,8.5,10,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, hash, userID, reference, photoID); err != nil {
		t.Fatal(err)
	}
	return models.WithdrawalAuthorization{TokenHash: hash, SessionHash: hash}
}

func captureEvidence(t *testing.T, c *app.Container, u *models.User, session, quote string) string {
	t.Helper()
	ctx := context.Background()
	token, err := c.Security.StartEvidence(ctx, u.ID, session, quote)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Security.SaveEvidence(ctx, u.ID, session, token, verificationJPEG(t), models.WithdrawalEvidence{Latitude: 7.7, Longitude: 8.5, AccuracyM: 10, CapturedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return token
}

func fixtureReviewEvidence(t *testing.T, c *app.Container, id int64, adminID int, session string) {
	t.Helper()
	ctx := context.Background()
	hash := services.SecurityHash(session)
	if err := c.Security.Repo.ReviewEvidence(ctx, id, adminID, hash); err != nil {
		t.Fatal(err)
	}
	evidence, err := c.Security.Repo.Evidence(ctx, id)
	if err != nil || evidence == nil {
		t.Fatal("missing fixture evidence", err)
	}
	if err = c.Security.Repo.MarkPhotoViewed(ctx, adminID, hash, evidence.PhotoID); err != nil {
		t.Fatal(err)
	}
}
