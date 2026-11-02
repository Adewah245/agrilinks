package integration_test

import (
	"database/sql"
	"fmt"
	"testing"
)

// Test-only opening money uses balanced journals and the same cached projection.
func seedAccountingWallet(t *testing.T, db *sql.DB, userID int, kobo int64) {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var walletID, journalID int64
	if err = tx.QueryRow(`UPDATE wallets SET balance=$2::numeric/100 WHERE owner_type='user' AND owner_id=$1 AND currency='NGN' RETURNING id`, userID, kobo).Scan(&walletID); err != nil {
		t.Fatal(err)
	}
	code := fmt.Sprintf("wallet:%d", walletID)
	if _, err = tx.Exec(`INSERT INTO accounting_accounts(code,name,category,wallet_id) VALUES($1,'Test customer wallet','liability',$2) ON CONFLICT DO NOTHING`, code, walletID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(`INSERT INTO accounting_journals(reference,transaction_type,fingerprint,evidence) VALUES($1,'test_opening','test-fixture','{"source":"disposable_test_fixture"}') RETURNING id`, "test-opening:"+code).Scan(&journalID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO accounting_postings(journal_id,line,account_code,currency,debit_kobo,credit_kobo) VALUES($1,1,'opening_suspense','NGN',$3,0),($1,2,$2,'NGN',0,$3)`, journalID, code, kobo); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
