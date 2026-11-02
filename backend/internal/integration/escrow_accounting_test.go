package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"backend/internal/repository"
	"backend/internal/testutil"
)

func TestEscrowDisputesRefundsChargebacksAndRaces(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	buyer := fixtureUser(t, c, 1, "buyer")
	seller := fixtureUser(t, c, 2, "farmer")
	admin := fixtureUser(t, c, 3, "buyer")
	outsider := fixtureUser(t, c, 4, "buyer")
	if _, err := db.Exec(`UPDATE users SET staff_role='finance_admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	admin.StaffRole = "finance_admin"
	repo := repository.NewPaymentRepository(db, 250)
	escrow := repository.NewEscrowRepository(db)
	create := func(key string) (int, string, string) {
		t.Helper()
		product := fixtureProduct(t, db, seller.ID, "Protected maize "+key, 10, 100)
		if err := c.Order.PlaceOrder(buyer.ID, product, 2, key); err != nil {
			t.Fatal(err)
		}
		var order int
		if err := db.QueryRow(`SELECT id FROM orders WHERE checkout_key=$1`, key).Scan(&order); err != nil {
			t.Fatal(err)
		}
		ref := "PROTECTED-" + key
		pid := fmt.Sprint(70000 + order)
		if _, err := repo.CreatePendingForOrder(buyer.ID, order, ref, "NGN", "flutterwave"); err != nil {
			t.Fatal(err)
		}
		zero := int64(0)
		for i := 0; i < 3; i++ {
			if _, err := repo.FinalizePayment(ref, "successful", pid, nil, repository.DepositCosts{ProviderKobo: &zero}); err != nil {
				t.Fatal(err)
			}
		}
		return order, ref, pid
	}
	assertLedger := func() {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM (SELECT journal_id,currency FROM accounting_postings GROUP BY journal_id,currency HAVING SUM(debit_kobo::numeric)<>SUM(credit_kobo::numeric)) bad`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("unbalanced journals %d: %v", n, err)
		}
		if err := db.QueryRow(`SELECT count(*) FROM wallets w WHERE w.balance*100<>COALESCE((SELECT SUM(credit_kobo::numeric-debit_kobo::numeric) FROM accounting_postings WHERE account_code='wallet:'||w.id AND currency=w.currency),0)`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("wallet mismatches %d: %v", n, err)
		}
	}
	t.Run("checks disputes and single release", func(t *testing.T) {
		id, ref, pid := create("release-checks")
		if err := escrow.Act(ctx, admin, id, "release", "Premature review"); err == nil {
			t.Fatal("released before delivery")
		}
		if err := escrow.Act(ctx, outsider, id, "seller_delivered", "Forged delivery"); err == nil {
			t.Fatal("outsider delivered order")
		}
		if err := escrow.Act(ctx, seller, id, "seller_delivered", "Delivery evidence received"); err != nil {
			t.Fatal(err)
		}
		if err := escrow.Act(ctx, buyer, id, "confirm_delivery", "All goods received"); err != nil {
			t.Fatal(err)
		}
		if err := escrow.Act(ctx, buyer, id, "dispute", "Goods fail the agreed specification"); err != nil {
			t.Fatal(err)
		}
		if err := escrow.Act(ctx, admin, id, "release", "Must remain frozen"); err == nil {
			t.Fatal("disputed escrow released")
		}
		var frozen int64
		if err := db.QueryRow(`SELECT COALESCE(SUM(credit_kobo-debit_kobo),0) FROM accounting_postings WHERE account_code=$1`, fmt.Sprintf("disputed-escrow:%d", id)).Scan(&frozen); err != nil || frozen != 20000 {
			t.Fatalf("disputed liability=%d %v", frozen, err)
		}
		if err := escrow.Act(ctx, admin, id, "resolve_dispute", "Replacement delivery agreed and completed"); err != nil {
			t.Fatal(err)
		}
		if err := escrow.Act(ctx, admin, id, "release", "Needs renewed buyer confirmation"); err == nil {
			t.Fatal("release reused pre-dispute confirmation")
		}
		if err := escrow.Act(ctx, buyer, id, "confirm_delivery", "Replacement goods received"); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := escrow.Act(ctx, admin, id, "release", "Agreement and delivery verified"); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
		var releases int
		if err := db.QueryRow(`SELECT count(*) FROM escrow_events WHERE order_id=$1 AND action='release'`, id).Scan(&releases); err != nil || releases != 1 {
			t.Fatalf("release decisions=%d %v", releases, err)
		}
		var sellerCredit string
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, seller.ID).Scan(&sellerCredit); err != nil || sellerCredit != "195.00" {
			t.Fatalf("seller received=%s %v", sellerCredit, err)
		}
		for i := 0; i < 3; i++ {
			if _, err := repo.FinalizePayment(ref, "reversed", pid, nil); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := repo.FinalizePayment(ref, "successful", pid, nil); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, seller.ID).Scan(&sellerCredit); err != nil || sellerCredit != "0.00" {
			t.Fatalf("seller retained chargeback=%s %v", sellerCredit, err)
		}
		assertLedger()
	})
	t.Run("refund and external reversal cannot pay buyer twice", func(t *testing.T) {
		id, ref, pid := create("refund-checks")
		if err := escrow.Act(ctx, buyer, id, "dispute", "Order was not delivered"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			if err := escrow.Act(ctx, admin, id, "refund", "Verified non-delivery; refund buyer"); err != nil {
				t.Fatal(err)
			}
		}
		var balance string
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_id=$1 AND owner_type='user'`, buyer.ID).Scan(&balance); err != nil || balance != "200.00" {
			t.Fatalf("buyer refund=%s %v", balance, err)
		}
		if _, err := repo.FinalizePayment(ref, "reversed", pid, nil); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_id=$1 AND owner_type='user'`, buyer.ID).Scan(&balance); err != nil || balance != "0.00" {
			t.Fatalf("double refund=%s %v", balance, err)
		}
		assertLedger()
	})
	t.Run("held provider chargeback prevents every release", func(t *testing.T) {
		id, ref, pid := create("held-reversal")
		if _, err := repo.FinalizePayment(ref, "reversed", pid, nil); err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"release", "refund", "resolve_dispute"} {
			if err := escrow.Act(ctx, admin, id, action, "Attempt after provider chargeback"); err == nil {
				t.Fatalf("%s accepted after chargeback", action)
			}
		}
		assertLedger()
	})
}
