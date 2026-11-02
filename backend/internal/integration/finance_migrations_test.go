package integration_test

import (
	"backend/internal/database"
	"backend/internal/testutil"
	"backend/migrations"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFinanceMigrationPreservesExistingBalances(t *testing.T) {
	db, _ := testutil.Database(t, false)
	baseline := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		version, _ := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if version > 68 {
			continue
		}
		body, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		baseline[name] = &fstest.MapFile{Data: body}
	}
	if err = database.ApplyMigrations(db, baseline); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT table_name,string_agg(column_name||':'||data_type,', ' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name IN ('wallets','wallet_transactions','wallet_deposits','wallet_withdrawals','transactions','platform_revenue') GROUP BY table_name ORDER BY table_name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, columns string
		if err = rows.Scan(&name, &columns); err != nil {
			t.Fatal(err)
		}
		t.Log(name, columns)
	}
	rows.Close()
	var userID, walletID int
	if err = db.QueryRow(`INSERT INTO users(full_name,phone,password_hash,role,account_type) VALUES('Historical Buyer','2348012345678','historical-test-hash','buyer','individual') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`INSERT INTO wallets(owner_type,owner_id,currency,balance) VALUES('user',$1,'NGN',1234.56) ON CONFLICT(owner_type,owner_id,currency) DO UPDATE SET balance=1234.56 RETURNING id`, userID).Scan(&walletID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'deposit',1234.56,'historical-deposit')`, walletID); err != nil {
		t.Fatal(err)
	}
	if err = database.RunMigration(db); err != nil {
		t.Fatal(err)
	}
	var balance string
	var credit, debit int64
	if err = db.QueryRow(`SELECT balance::text FROM wallets WHERE id=$1`, walletID).Scan(&balance); err != nil || balance != "1234.56" {
		t.Fatalf("historical balance changed: %s %v", balance, err)
	}
	if err = db.QueryRow(`SELECT COALESCE(SUM(debit_kobo),0),COALESCE(SUM(credit_kobo),0) FROM accounting_postings`).Scan(&debit, &credit); err != nil || debit != credit || credit != 123456 {
		t.Fatalf("opening ledger debit=%d credit=%d: %v", debit, credit, err)
	}
}
