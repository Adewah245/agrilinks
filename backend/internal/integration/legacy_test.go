package integration_test

import (
	"os"
	"testing"

	"backend/internal/database"
	"backend/internal/legacycutover"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestLegacyTransferPreservesRecordsAndBalances(t *testing.T) {
	db, _ := testutil.Database(t, false)
	for _, name := range []string{"testdata/legacy_schema.sql", "testdata/legacy_records.sql"} {
		script, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(script)); err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("old password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE farmers SET password_hash=$1`, string(hash)); err != nil {
		t.Fatal(err)
	}
	if err = database.RunMigration(db); err == nil {
		t.Fatal("ordinary migration accepted an old database")
	}
	if err = legacycutover.Preflight(db); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err = db.QueryRow(`SELECT to_regclass('public.users') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatal("preflight changed the database")
	}
	if _, err = db.Exec(`UPDATE wallet_withdrawals SET bank_name=''`); err != nil {
		t.Fatal(err)
	}
	if err = legacycutover.Run(db); err == nil {
		t.Fatal("transfer accepted an invalid withdrawal")
	}
	if err = db.QueryRow(`SELECT to_regclass('public.farmers') IS NOT NULL`).Scan(&exists); err != nil || !exists {
		t.Fatal("failed preflight moved old tables")
	}
	if _, err = db.Exec(`UPDATE wallet_withdrawals SET bank_name='Test bank'`); err != nil {
		t.Fatal(err)
	}
	if err = legacycutover.Run(db); err != nil {
		t.Fatal(err)
	}
	if err = legacycutover.Verify(db); err != nil {
		t.Fatal(err)
	}
	checks := map[string]int{
		`SELECT count(*) FROM users`: 2,
		`SELECT count(*) FROM products WHERE id=41 AND owner_id=11 AND quantity=10 AND price=90`:                                  1,
		`SELECT count(*) FROM order_items WHERE order_id=51 AND product_id=41 AND quantity=2 AND total_price=180`:                 1,
		`SELECT count(*) FROM chat_messages WHERE conversation_id=101`:                                                            2,
		`SELECT count(*) FROM conversation_members WHERE conversation_id=101 AND user_id=11 AND member_role='admin'`:              1,
		`SELECT count(*) FROM wallets WHERE owner_type='user' AND owner_id=11 AND balance=1234.56 AND wallet_number='8000000011'`: 1,
		`SELECT count(*) FROM wallet_transactions`:                                                                                2,
		`SELECT count(*) FROM transactions WHERE payment_reference='old-order-payment' AND amount=180 AND status='successful'`:    1,
		`SELECT count(*) FROM deliveries WHERE order_id=51 AND status='delivered'`:                                                1,
		`SELECT count(*) FROM product_reviews WHERE product_id=41 AND order_id=51`:                                                1,
		`SELECT count(*) FROM diagnoses WHERE id=121 AND symptoms='Leaves turning yellow'`:                                        1,
		`SELECT count(*) FROM pg_constraint WHERE conrelid='public.deliveries'::regclass AND conname='fk_deliveries_order'`:       1,
	}
	for query, want := range checks {
		var got int
		if err = db.QueryRow(query).Scan(&got); err != nil || got != want {
			t.Fatalf("%s: got %d want %d (%v)", query, got, want, err)
		}
	}
	c := application(t, db, testConfig(t))
	if _, err = c.Auth.Login("08010000011", "old password"); err != nil {
		t.Fatalf("old account cannot sign in: %v", err)
	}
	if _, err = db.Exec(`UPDATE wallets SET balance=balance+100 WHERE owner_type='user' AND owner_id=11`); err != nil {
		t.Fatal(err)
	}
	if err = legacycutover.Run(db); err != nil {
		t.Fatal(err)
	}
	var balance string
	if err = db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=11`).Scan(&balance); err != nil || balance != "1334.56" {
		t.Fatalf("repeated transfer overwrote current money: %s %v", balance, err)
	}
	if err = database.RunMigration(db); err != nil {
		t.Fatal(err)
	}
}
