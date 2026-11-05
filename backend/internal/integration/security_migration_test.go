package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/testutil"
	"backend/migrations"
)

func TestSecurityReviewMigrationPreservesEvidenceAndReservedMoney(t *testing.T) {
	db, _ := testutil.Database(t, false)
	baseline := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		version, _ := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if version > 83 {
			continue
		}
		body, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		baseline[name] = &fstest.MapFile{Data: body}
	}
	if err := database.ApplyMigrations(db, baseline); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	owner := fixtureUser(t, c, 1, "buyer")
	admin := fixtureUser(t, c, 2, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	cookie := session(t, c, admin)
	repo := repository.NewWalletRepository(db, "NGN", 0)
	seedAccountingWallet(t, db, owner.ID, 100000)
	account := &models.UserBankAccount{BankName: "Test bank", BankCode: "001", AccountName: owner.FullName, AccountNumber: "0123456789"}
	paidAuth := fixtureEvidence(t, db, owner.ID, "historical-completed-review")
	if err := repo.RequestWithdrawal(ctx, owner.ID, "250.00", "historical-completed-review", account, nil, paidAuth); err != nil {
		t.Fatal(err)
	}
	paid, err := repo.WithdrawalByReference(ctx, "historical-completed-review")
	if err != nil {
		t.Fatal(err)
	}
	// Reconstruct the pre-084 approval and complete its real ledger postings.
	if _, err := db.Exec(`UPDATE wallet_withdrawals SET status='processing',provider_submission_state='unknown',approved_by=$2 WHERE id=$1`, paid.ID, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveTransferID(ctx, paid.ID, "historical-reviewed-transfer"); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishWithdrawal(ctx, paid.ID, 0, false, "paid", true); err != nil {
		t.Fatal(err)
	}
	var paidViewed sql.NullTime
	if err := db.QueryRow(`INSERT INTO withdrawal_evidence_reviews(withdrawal_id,reviewer_id,session_hash,photo_viewed_at) VALUES($1,$2,$3,CURRENT_TIMESTAMP) RETURNING photo_viewed_at`, paid.ID, admin.ID, services.SecurityHash(cookie.Value)).Scan(&paidViewed); err != nil {
		t.Fatal(err)
	}
	auth := fixtureEvidence(t, db, owner.ID, "historical-private-review")
	if err := repo.RequestWithdrawal(ctx, owner.ID, "250.00", "historical-private-review", account, nil, auth); err != nil {
		t.Fatal(err)
	}
	withdrawal, err := repo.WithdrawalByReference(ctx, "historical-private-review")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO withdrawal_evidence_reviews(withdrawal_id,reviewer_id,session_hash,photo_viewed_at) VALUES($1,$2,$3,CURRENT_TIMESTAMP)`, withdrawal.ID, admin.ID, services.SecurityHash(cookie.Value)); err != nil {
		t.Fatal(err)
	}
	var beforeBalance string
	var beforePhoto []byte
	if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, owner.ID).Scan(&beforeBalance); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT p.encrypted_image FROM private_verification_photos p JOIN withdrawal_evidence e ON e.photo_id=p.id WHERE e.withdrawal_id=$1`, withdrawal.ID).Scan(&beforePhoto); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := database.RunMigration(db); err != nil {
			t.Fatal(err)
		}
	}
	var afterBalance, status string
	var afterPhoto []byte
	var viewed sql.NullTime
	if err := db.QueryRow(`SELECT w.balance::text,d.status,p.encrypted_image,v.photo_viewed_at FROM wallet_withdrawals d JOIN wallets w ON w.id=d.wallet_id JOIN withdrawal_evidence e ON e.withdrawal_id=d.id JOIN private_verification_photos p ON p.id=e.photo_id JOIN withdrawal_evidence_reviews v ON v.withdrawal_id=d.id WHERE d.id=$1`, withdrawal.ID).Scan(&afterBalance, &status, &afterPhoto, &viewed); err != nil {
		t.Fatal(err)
	}
	if afterBalance != beforeBalance || status != "pending" || !bytes.Equal(beforePhoto, afterPhoto) {
		t.Fatal("migration changed reserved funds, withdrawal status, or private evidence")
	}
	if viewed.Valid {
		t.Fatal("migration retained an incomplete historical review authorization")
	}
	var paidViewedAfter sql.NullTime
	if err := db.QueryRow(`SELECT photo_viewed_at FROM withdrawal_evidence_reviews WHERE withdrawal_id=$1`, paid.ID).Scan(&paidViewedAfter); err != nil {
		t.Fatal(err)
	}
	if !paidViewedAfter.Valid || !paidViewedAfter.Time.Equal(paidViewed.Time) {
		t.Fatal("migration erased completed withdrawal access history")
	}
	if fresh, err := c.Security.FreshMFA(ctx, admin.ID, cookie.Value); err != nil || !fresh {
		t.Fatalf("migration invalidated administrator MFA: %v", err)
	}
	if err := repo.ClaimWithdrawal(ctx, withdrawal.ID, admin.ID, services.SecurityHash(cookie.Value)); err == nil {
		t.Fatal("historical review bypassed the new photo checks")
	}
}

// This exact revision has already been applied to an existing installation.
// Pin it so later changes must be shipped in a new numbered migration.
func TestAppliedPhotoReviewMigrationIsUnchanged(t *testing.T) {
	body, err := fs.ReadFile(migrations.Files, "083_private_photo_review_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	checksum := sha256.Sum256(body)
	if fmt.Sprintf("%x", checksum) != "359aa1538cbad6b210a045aac22bf810adfd7e859d548d8d65da3b0e5ac735df" {
		t.Fatal("applied migration 083 changed; add a new migration instead")
	}
}
