package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/testutil"
)

type financeFixture struct {
	t       *testing.T
	db      *sql.DB
	wallet  *WalletRepository
	finance *FinanceRepository
	counter int
	ctx     context.Context
}

func newFinanceFixture(t *testing.T) *financeFixture {
	db, _ := testutil.Database(t, true)
	return &financeFixture{t: t, db: db, wallet: NewWalletRepository(db, "NGN", 250), finance: NewFinanceRepository(db), ctx: context.Background()}
}
func (f *financeFixture) user(role string) *models.User {
	f.t.Helper()
	f.counter++
	u := &models.User{FullName: fmt.Sprintf("Finance Test %d", f.counter), StaffRole: role, AccountStatus: "active"}
	err := f.db.QueryRow(`INSERT INTO users(full_name,phone,password_hash,role,account_type,staff_role,account_status) VALUES($1,$2,'test-only-hash','buyer','individual',$3,'active') RETURNING id`, u.FullName, fmt.Sprintf("234801%07d", f.counter), role).Scan(&u.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err = f.wallet.Wallet(f.ctx, u.ID); err != nil {
		f.t.Fatal(err)
	}
	return u
}
func (f *financeFixture) key() string { f.counter++; return fmt.Sprintf("%032x", f.counter) }
func (f *financeFixture) quote(u *models.User, kind string, amount, provider int64) *models.FeeQuote {
	f.t.Helper()
	p, err := f.wallet.FeePolicy(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	q := &models.FeeQuote{ID: f.key(), UserID: u.ID, Kind: kind, Currency: "NGN", AmountKobo: amount, PolicyID: p.ID, ProviderKobo: provider, CustomerProviderKobo: provider, ProviderEvidence: `{"source":"simulated_provider_quote"}`, BankCode: "001", BankName: "Test bank", AccountName: u.FullName, AccountNumber: "0123456789"}
	if kind == "deposit" {
		q.ServiceKobo = p.FundingServiceKobo
		q.WalletCreditKobo = amount
		q.CheckoutTotalKobo = amount + q.ServiceKobo + provider
		q.PaymentMethod = "banktransfer"
	} else {
		q.ServiceKobo, err = p.WithdrawalFee(amount)
		if err != nil {
			f.t.Fatal(err)
		}
		q.BankCreditKobo = amount
		q.WalletDebitKobo = amount + q.ServiceKobo + provider
	}
	if err = f.wallet.SaveFeeQuote(f.ctx, q); err != nil {
		f.t.Fatal(err)
	}
	return q
}
func (f *financeFixture) fund(u *models.User, amount, provider int64) (string, *models.FeeQuote) {
	f.t.Helper()
	q := f.quote(u, "deposit", amount, provider)
	ref := fmt.Sprintf("WALLET-%d-%s", u.ID, q.ID)
	gross := money.Format(amount + q.ServiceKobo)
	if _, err := f.wallet.CreateDeposit(f.ctx, u.ID, ref, gross); err != nil {
		f.t.Fatal(err)
	}
	charged := q.CheckoutTotalKobo
	if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, fmt.Sprint(f.counter), gross, "NGN", "successful", nil, DepositCosts{ProviderKobo: &provider, ChargedKobo: &charged}); err != nil {
		f.t.Fatal(err)
	}
	return ref, q
}
func (f *financeFixture) balance(u *models.User, want int64) {
	f.t.Helper()
	var n int64
	if err := f.db.QueryRow(`SELECT (balance*100)::bigint FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, u.ID).Scan(&n); err != nil || n != want {
		f.t.Fatalf("wallet %d balance=%d want=%d: %v", u.ID, n, want, err)
	}
}
func (f *financeFixture) account(code string) int64 {
	f.t.Helper()
	n, err := financeBalance(f.ctx, f.db, code)
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}
func (f *financeFixture) balanced() {
	f.t.Helper()
	var d, c int64
	if err := f.db.QueryRow(`SELECT COALESCE(SUM(debit_kobo),0),COALESCE(SUM(credit_kobo),0) FROM accounting_postings`).Scan(&d, &c); err != nil || d != c {
		f.t.Fatalf("ledger debits=%d credits=%d: %v", d, c, err)
	}
	var bad int
	if err := f.db.QueryRow(`SELECT count(*) FROM wallets w WHERE (w.balance*100)::numeric<>COALESCE((SELECT SUM(p.credit_kobo::numeric-p.debit_kobo::numeric) FROM accounting_postings p WHERE p.account_code='wallet:'||w.id AND p.currency=w.currency),0)`).Scan(&bad); err != nil || bad != 0 {
		f.t.Fatalf("wallet projection mismatches=%d: %v", bad, err)
	}
	f.t.Logf("balanced ledger: debit=%d credit=%d kobo; wallet mismatches=0", d, c)
}
func (f *financeFixture) request(u *models.User, q *models.FeeQuote, ref string) error {
	// Accounting fixtures supply private evidence; HTTP capture/authorization
	// boundaries are exercised separately by the integration security suite.
	_, err := f.db.Exec(`WITH photo AS (INSERT INTO private_verification_photos(user_id,purpose,encrypted_image) VALUES($1,'withdrawal',$2) RETURNING id) INSERT INTO withdrawal_evidence(token_hash,user_id,session_hash,quote_id,reference,expires_at,photo_id,latitude,longitude,accuracy_m,captured_at,received_at) SELECT $3,$1,$3,$4,$3,CURRENT_TIMESTAMP+interval '10 minutes',id,7.7,8.5,10,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM photo ON CONFLICT(token_hash) DO NOTHING`, u.ID, []byte("test-only-encrypted-photo"), ref, q.ID)
	if err != nil {
		return err
	}
	return f.wallet.RequestWithdrawal(f.ctx, u.ID, q.Amount(), ref, &models.UserBankAccount{BankName: q.BankName, BankCode: q.BankCode, AccountName: q.AccountName, AccountNumber: q.AccountNumber}, q, models.WithdrawalAuthorization{TokenHash: ref, SessionHash: ref})
}

func (f *financeFixture) reviewedClaim(id int64) error {
	admin := f.user("finance_admin")
	session := f.key() + f.key()
	security := NewSecurityRepository(f.db)
	if err := NewSessionRepository(f.db).Create(f.ctx, session, admin.ID, time.Now().Add(time.Hour)); err != nil {
		return err
	}
	if err := security.BeginMFA(f.ctx, admin.ID, session, []byte("test-only-encrypted-secret"), false); err != nil {
		return err
	}
	m, err := security.MFA(f.ctx, admin.ID)
	if err != nil {
		return err
	}
	if err = security.ConfirmMFA(f.ctx, admin.ID, session, m, 1, nil); err != nil {
		return err
	}
	if err = security.ReviewEvidence(f.ctx, id, admin.ID, session); err != nil {
		return err
	}
	evidence, err := security.Evidence(f.ctx, id)
	if err != nil {
		return err
	}
	if err = security.MarkPhotoViewed(f.ctx, admin.ID, session, evidence.PhotoID); err != nil {
		return err
	}
	return f.wallet.ClaimWithdrawal(f.ctx, id, admin.ID, session)
}

func TestFinancialFundingAndWithdrawalAccounting(t *testing.T) {
	f := newFinanceFixture(t)
	u := f.user("")
	ref, q := f.fund(u, 10000000, 1500)
	f.balance(u, 10000000)
	if f.account("funding_fee_revenue") != -5000 || f.account("provider_expense") != 1500 || f.account("provider_cost_recovery") != -1500 {
		t.Fatal("funding destinations incorrect")
	}
	for i := 0; i < 4; i++ {
		if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, fmt.Sprint(f.counter), money.Format(q.AmountKobo+q.ServiceKobo), "NGN", "successful", nil); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(u, 10000000)
	if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, fmt.Sprint(f.counter), money.Format(q.AmountKobo+q.ServiceKobo), "NGN", "pending", nil); err != nil {
		t.Fatal(err)
	}
	remaining := int64(10000000)
	for _, tt := range []struct{ amount, fee int64 }{{100, 5000}, {500000, 5000}, {500100, 20000}, {999900, 20000}, {1000000, 30000}} {
		t.Run(fmt.Sprintf("tier_%d", tt.amount), func(t *testing.T) {
			q := f.quote(u, "withdrawal", tt.amount, 1000)
			if q.ServiceKobo != tt.fee || q.BankCreditKobo != tt.amount {
				t.Fatalf("wrong tier quote: %+v", q)
			}
			ref := "WITHDRAW-" + q.ID
			if err := f.request(u, q, ref); err != nil {
				t.Fatal(err)
			}
			remaining -= q.WalletDebitKobo
			f.balance(u, remaining)
			if err := f.request(u, q, ref); err != nil {
				t.Fatal(err)
			}
			x, err := f.wallet.WithdrawalByReference(f.ctx, ref)
			if err != nil {
				t.Fatal(err)
			}
			if err = f.reviewedClaim(x.ID); err != nil {
				t.Fatal(err)
			}
			if err = f.wallet.SaveTransferID(f.ctx, x.ID, fmt.Sprint(10000+x.ID)); err != nil {
				t.Fatal(err)
			}
			actual := int64(500)
			for i := 0; i < 3; i++ {
				if err = f.wallet.FinishWithdrawal(f.ctx, x.ID, 0, false, "paid", true, &actual); err != nil {
					t.Fatal(err)
				}
			}
			remaining += 500
			f.balance(u, remaining)
			f.balanced()
		})
	}
	t.Run("complete debit required", func(t *testing.T) {
		q := f.quote(u, "withdrawal", remaining/100*100, 1000)
		if err := f.request(u, q, "insufficient"); err == nil {
			t.Fatal("payout accepted without money for complete debit")
		}
		f.balance(u, remaining)
	})
	t.Run("failed payout returns all customer fees", func(t *testing.T) {
		q := f.quote(u, "withdrawal", 10000, 1000)
		if err := f.request(u, q, "failed"); err != nil {
			t.Fatal(err)
		}
		x, _ := f.wallet.WithdrawalByReference(f.ctx, "failed")
		if err := f.reviewedClaim(x.ID); err != nil {
			t.Fatal(err)
		}
		actual := int64(250)
		if err := f.wallet.FinishWithdrawal(f.ctx, x.ID, 0, false, "failed", true, &actual); err != nil {
			t.Fatal(err)
		}
		f.balance(u, remaining)
		f.balanced()
	})
	t.Run("payout reversal returns fees once", func(t *testing.T) {
		q := f.quote(u, "withdrawal", 10000, 1000)
		if err := f.request(u, q, "reverse-payout"); err != nil {
			t.Fatal(err)
		}
		x, _ := f.wallet.WithdrawalByReference(f.ctx, "reverse-payout")
		if err := f.reviewedClaim(x.ID); err != nil {
			t.Fatal(err)
		}
		actual := int64(1000)
		if err := f.wallet.FinishWithdrawal(f.ctx, x.ID, 0, false, "paid", true, &actual); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			if err := f.wallet.FinishWithdrawal(f.ctx, x.ID, 0, false, "reversed", true, &actual); err != nil {
				t.Fatal(err)
			}
		}
		f.balance(u, remaining)
		f.balanced()
	})
	t.Run("deposit reversal creates debt for spent money", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, fmt.Sprint(2), money.Format(q.AmountKobo+q.ServiceKobo), "NGN", "reversed", nil); err != nil {
				t.Fatal(err)
			}
		}
		f.balance(u, 0)
		if f.account("funding_fee_revenue") != 0 {
			t.Fatal("reversed funding retained service revenue")
		}
		var debt int64
		if err := f.db.QueryRow(`SELECT recovery_debt_kobo FROM wallets WHERE owner_id=$1 AND owner_type='user'`, u.ID).Scan(&debt); err != nil || debt != 10000000-remaining {
			t.Fatalf("debt %d: %v", debt, err)
		}
		f.fund(u, debt+50000, 0)
		f.balance(u, 50000)
		f.balanced()
	})
}

func TestAccountingConcurrencyImmutabilityAndRollback(t *testing.T) {
	f := newFinanceFixture(t)
	u := f.user("")
	f.fund(u, 1000000, 0)
	q := f.quote(u, "withdrawal", 500000, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := f.request(u, q, "same-request"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	f.balance(u, 494000)
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM wallet_withdrawals WHERE reference='same-request'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate requests=%d: %v", count, err)
	}
	x, _ := f.wallet.WithdrawalByReference(f.ctx, "same-request")
	if err := f.wallet.FinishWithdrawal(f.ctx, x.ID, u.ID, false, "cancelled", false); err != nil {
		t.Fatal(err)
	}
	f.balance(u, 1000000)
	qs := []*models.FeeQuote{}
	for i := 0; i < 12; i++ {
		qs = append(qs, f.quote(u, "withdrawal", 500000, 1000))
	}
	var won atomic.Int32
	for i, q := range qs {
		wg.Add(1)
		go func(i int, q *models.FeeQuote) {
			defer wg.Done()
			if f.request(u, q, fmt.Sprintf("race-%d", i)) == nil {
				won.Add(1)
			}
		}(i, q)
	}
	wg.Wait()
	if won.Load() != 1 {
		t.Fatalf("distinct concurrent requests accepted=%d", won.Load())
	}
	f.balance(u, 494000)
	f.balanced()
	t.Run("posted journals cannot change", func(t *testing.T) {
		for _, query := range []string{`UPDATE accounting_postings SET debit_kobo=debit_kobo WHERE journal_id=(SELECT min(id) FROM accounting_journals)`, `DELETE FROM accounting_journals`, `UPDATE accounting_accounts SET category='revenue' WHERE code LIKE 'wallet:%'`, `INSERT INTO accounting_postings SELECT min(id),99,'provider_clearing','NGN',1,0 FROM accounting_journals`} {
			if _, err := f.db.Exec(query); err == nil {
				t.Fatalf("history mutation accepted: %s", query)
			}
		}
	})
	t.Run("unbalanced transaction rejected", func(t *testing.T) {
		tx, err := f.db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err = postJournal(tx, "unbalanced-test", "test", "NGN", 0, "{}", debit("provider_clearing", 100), credit("other_platform_revenue", 99)); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err == nil {
			t.Fatal("unbalanced journal committed")
		}
	})
	t.Run("database failure rolls back money and fees", func(t *testing.T) {
		v := f.user("")
		q := f.quote(v, "deposit", 100000, 0)
		ref := fmt.Sprintf("WALLET-%d-%s", v.ID, q.ID)
		if _, err := f.wallet.CreateDeposit(f.ctx, v.ID, ref, "1050.00"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.db.Exec(`CREATE FUNCTION fail_test_posting() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated database failure'; END $$; CREATE TRIGGER fail_test_posting BEFORE INSERT ON accounting_postings FOR EACH ROW EXECUTE FUNCTION fail_test_posting()`); err != nil {
			t.Fatal(err)
		}
		if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, "99881", "1050.00", "NGN", "successful", nil); err == nil {
			t.Fatal("failure was hidden")
		}
		if _, err := f.db.Exec(`DROP TRIGGER fail_test_posting ON accounting_postings; DROP FUNCTION fail_test_posting()`); err != nil {
			t.Fatal(err)
		}
		f.balance(v, 0)
		var n int
		if err := f.db.QueryRow(`SELECT count(*) FROM wallet_fee_records WHERE reference=$1`, ref).Scan(&n); err != nil || n != 0 {
			t.Fatalf("fee survived rollback: %d %v", n, err)
		}
		if _, err := f.wallet.FinalizeDeposit(f.ctx, ref, "99881", "1050.00", "NGN", "successful", nil); err != nil {
			t.Fatal(err)
		}
		f.balance(v, 100000)
		f.balanced()
	})
}

func TestCompanyTreasurySettlementAndPermissions(t *testing.T) {
	f := newFinanceFixture(t)
	customer := f.user("")
	requester := f.user("finance_admin")
	approver := f.user("super_admin")
	support := f.user("customer_care")
	ref, q := f.fund(customer, 1000000, 0)
	if a, _, err := f.finance.Available(f.ctx); err != nil || a != 0 {
		t.Fatalf("unsettled revenue became cash: %d %v", a, err)
	}
	bankChange := ChangePayload{CompanyName: "AgriLinks Limited", BankName: "Test business bank", BankCode: "001", AccountNumber: "1234567890", AccountName: "AgriLinks Limited", Evidence: "Simulated business account verification"}
	id, err := f.finance.ProposeChange(f.ctx, requester, f.key(), "company_bank", "Company verified settlement destination", bankChange)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.DecideChange(f.ctx, requester, id, true, "Attempt self approval"); err == nil {
		t.Fatal("bank self approval accepted")
	}
	if err = f.finance.DecideChange(f.ctx, support, id, true, "Attempt support approval"); err == nil {
		t.Fatal("support approved bank destination")
	}
	if err = f.finance.DecideChange(f.ctx, approver, id, true, "Independently verified company ownership"); err != nil {
		t.Fatal(err)
	}
	p, err := f.finance.TransferPolicy(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	batch := models.SettlementBatch{Provider: "flutterwave", Reference: "test-batch", Currency: "NGN", Destination: "provider_balance", Source: "simulated_provider_statement", Evidence: `{"document":"test-settlement"}`, StatementReference: "bank-001", Reason: "Compare provider and bank records", GrossKobo: q.CheckoutTotalKobo, NetKobo: q.CheckoutTotalKobo, StatementKobo: q.CheckoutTotalKobo, Items: []models.SettlementItem{{Reference: ref, GrossKobo: q.CheckoutTotalKobo, NetKobo: q.CheckoutTotalKobo}}}
	settlementID, err := f.finance.ImportSettlement(f.ctx, requester, batch)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ReconcileSettlement(f.ctx, requester, settlementID, "Self reconciliation must fail"); err == nil {
		t.Fatal("self reconciliation accepted")
	}
	if err = f.finance.ReconcileSettlement(f.ctx, approver, settlementID, "Matched provider receipt and settled balance"); err != nil {
		t.Fatal(err)
	}
	if a, source, err := f.finance.Available(f.ctx); err != nil || a != 5000 || source != 5000 {
		t.Fatalf("available=%d source=%d: %v", a, source, err)
	}
	if _, err = f.finance.ImportSettlement(f.ctx, requester, batch); err == nil {
		t.Fatal("duplicate settlement accepted")
	}
	cq := &models.CompanyQuote{ID: f.key(), InitiatedBy: requester.ID, AmountKobo: 3000, ProviderKobo: 1000, TotalKobo: 4000, CompanyAccountID: p.CompanyAccountID.Int64, PolicyID: p.ID, ProviderEvidence: `{"source":"simulated_transfer_fee"}`}
	if err = f.finance.SaveCompanyQuote(f.ctx, cq); err != nil {
		t.Fatal(err)
	}
	w, err := f.finance.RequestCompanyWithdrawal(f.ctx, requester, cq.ID, "Transfer earned fees to company business account")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.finance.RequestCompanyWithdrawal(f.ctx, requester, cq.ID, "Transfer earned fees to company business account"); err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ApproveCompanyWithdrawal(f.ctx, requester, w.ID, true, "Self approval blocked"); err == nil {
		t.Fatal("self approval accepted")
	}
	if err = f.finance.ApproveCompanyWithdrawal(f.ctx, support, w.ID, true, "Support approval blocked"); err == nil {
		t.Fatal("support approval accepted")
	}
	if err = f.finance.ClaimCompanyWithdrawal(f.ctx, w.ID, approver); err == nil {
		t.Fatal("unapproved execution accepted")
	}
	if err = f.finance.ApproveCompanyWithdrawal(f.ctx, approver, w.ID, true, "All obligations and destination verified"); err != nil {
		t.Fatal(err)
	}
	state, _ := f.finance.CompanyWithdrawal(f.ctx, w.ID)
	if state.Status != "approved" {
		t.Fatal("approval did not remain separate from payment")
	}
	if err = f.finance.ClaimCompanyWithdrawal(f.ctx, w.ID, approver); err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ClaimCompanyWithdrawal(f.ctx, w.ID, approver); err == nil {
		t.Fatal("duplicate execution claim accepted")
	}
	if err = f.finance.SaveCompanyTransferID(f.ctx, w.ID, "90001"); err != nil {
		t.Fatal(err)
	}
	if err = f.finance.FinishCompanyWithdrawal(f.ctx, w.ID, "successful", 1000, `{"source":"simulated_provider"}`); err == nil {
		t.Fatal("success without bank evidence accepted")
	}
	proof := CompanySettlementProof{Actor: approver, StatementReference: "business-credit-001", ReceivedKobo: 3000, Evidence: `{"source":"simulated_bank_statement"}`}
	if err = f.finance.FinishCompanyWithdrawal(f.ctx, w.ID, "successful", 500, `{"source":"simulated_provider"}`, proof); err != nil {
		t.Fatal(err)
	}
	if f.account(fmt.Sprintf("company-bank:%d", p.CompanyAccountID.Int64)) != 3000 || f.account("provider_expense") != 500 {
		t.Fatal("company transfer destination or actual costs incorrect")
	}
	if a, source, err := f.finance.Available(f.ctx); err != nil || a != 4500 || source != 1500 {
		t.Fatalf("remaining company cash=%d source=%d: %v", a, source, err)
	}
	f.balance(customer, 1000000)
	f.balanced()
	reserveID, err := f.finance.ProposeChange(f.ctx, requester, f.key(), "reserve", "Reserve for future chargebacks", ChangePayload{Account: "chargeback_reserve", AmountKobo: 4500})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.DecideChange(f.ctx, approver, reserveID, true, "Approved reserve requirement"); err != nil {
		t.Fatal(err)
	}
	if a, source, err := f.finance.Available(f.ctx); err != nil || a != 0 || source != 0 {
		t.Fatalf("reserved funds available=%d %d: %v", a, source, err)
	}
	cq.ID = f.key()
	if err = f.finance.SaveCompanyQuote(f.ctx, cq); err != nil {
		t.Fatal(err)
	}
	if _, err = f.finance.RequestCompanyWithdrawal(f.ctx, requester, cq.ID, "Try withdrawing restricted reserves"); err == nil {
		t.Fatal("reserved obligations withdrawn")
	}
	page, err := f.finance.Page(f.ctx, approver, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Accounts) == 0 || len(page.Metrics) < 20 || len(page.Withdrawals) != 1 {
		t.Fatal("dashboard not backed by records")
	}
	if _, err = f.finance.Page(f.ctx, support, "", ""); err == nil {
		t.Fatal("support read company finances")
	}
	var csv strings.Builder
	if err = f.finance.Export(f.ctx, approver, &csv, "", ""); err != nil || !strings.Contains(csv.String(), "company_bank_transfer") {
		t.Fatalf("accounting export failed: %v", err)
	}
	policyID, err := f.finance.ProposeChange(f.ctx, requester, f.key(), "fee_policy", "Scheduled approved fee policy", ChangePayload{FundingServiceKobo: 5000, WithdrawalLowKobo: 5000, WithdrawalMidKobo: 20000, WithdrawalHighKobo: 30000, ProviderBearer: "platform", EffectiveAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.DecideChange(f.ctx, approver, policyID, true, "Provider bearer verified"); err != nil {
		t.Fatal(err)
	}
	current, err := f.wallet.FeePolicy(f.ctx)
	if err != nil || current.WithdrawalProviderBearer != "customer" {
		t.Fatal("future fee policy applied early")
	}
	f.balanced()
}

func TestSettlementCorrectionsDeductionsAndFundingAllocation(t *testing.T) {
	f := newFinanceFixture(t)
	u := f.user("")
	initiator := f.user("finance_admin")
	approver := f.user("super_admin")
	ref, q := f.fund(u, 100000, 1000)
	batch := models.SettlementBatch{Provider: "flutterwave", Reference: "corrected-batch", Currency: "NGN", Destination: "provider_balance", Source: "simulated_statement", Evidence: `{"source":"provider_and_bank_statement"}`, StatementReference: "statement-corrected", Reason: "Reconcile actual provider deduction", GrossKobo: 106000, DeductionsKobo: 1000, NetKobo: 105000, StatementKobo: 104999, Items: []models.SettlementItem{{Reference: ref, GrossKobo: 106000, DeductionsKobo: 1000, NetKobo: 105000}}}
	id, err := f.finance.ImportSettlement(f.ctx, initiator, batch)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ReconcileSettlement(f.ctx, approver, id, "Statement amount does not match"); err == nil {
		t.Fatal("mismatched settlement became cash")
	}
	if f.account("provider_settled_cash") != 0 {
		t.Fatal("mismatch credited company cash")
	}
	batch.StatementKobo = 105000
	correctedID, err := f.finance.ImportSettlement(f.ctx, initiator, batch)
	if err != nil || correctedID != id {
		t.Fatalf("correction failed: %d %v", correctedID, err)
	}
	if err = f.finance.ReconcileSettlement(f.ctx, approver, id, "Corrected statement independently matched"); err != nil {
		t.Fatal(err)
	}
	if f.account("provider_expense") != 1000 {
		t.Fatal("provider deduction charged twice")
	}
	var versions int
	if err = f.db.QueryRow(`SELECT count(*) FROM settlement_import_revisions WHERE batch_id=$1`, id).Scan(&versions); err != nil || versions != 2 {
		t.Fatalf("import versions=%d %v", versions, err)
	}
	withdrawal := f.quote(u, "withdrawal", 10000, 1000)
	if err = f.request(u, withdrawal, "settled-payout"); err != nil {
		t.Fatal(err)
	}
	x, _ := f.wallet.WithdrawalByReference(f.ctx, "settled-payout")
	if err = f.reviewedClaim(x.ID); err != nil {
		t.Fatal(err)
	}
	actual := int64(700)
	if err = f.wallet.FinishWithdrawal(f.ctx, x.ID, 0, false, "paid", true, &actual); err != nil {
		t.Fatal(err)
	}
	outgoing := models.SettlementBatch{Direction: "customer_payout", Provider: "flutterwave", Reference: "payout-batch", Currency: "NGN", Destination: "provider_balance", Source: "simulated_statement", Evidence: `{"source":"settled_provider_payout"}`, StatementReference: "provider-debit-001", Reason: "Reconcile bank payout plus actual cost", GrossKobo: 10000, DeductionsKobo: 700, NetKobo: 10700, StatementKobo: 10700, Items: []models.SettlementItem{{Reference: "settled-payout", GrossKobo: 10000, DeductionsKobo: 700, NetKobo: 10700}}}
	id, err = f.finance.ImportSettlement(f.ctx, initiator, outgoing)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ReconcileSettlement(f.ctx, approver, id, "Payout and deduction matched statement"); err != nil {
		t.Fatal(err)
	}
	if f.account("provider_clearing") != 0 || f.account("provider_expense") != 1700 {
		t.Fatal("outbound settlement duplicated costs or failed to clear")
	}
	f.balanced()
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = postJournal(tx, "provider-invoice-1", "provider_cost", "NGN", 0, `{"source":"verified_provider_invoice"}`, debit("provider_expense", 100), credit("provider_payable", 100)); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	outgoing.Direction = "provider_payable"
	outgoing.Reference = "invoice-settlement"
	outgoing.StatementReference = "provider-invoice-debit"
	outgoing.GrossKobo = 100
	outgoing.DeductionsKobo = 0
	outgoing.NetKobo = 100
	outgoing.StatementKobo = 100
	outgoing.Items = []models.SettlementItem{{Reference: "provider-invoice-1", GrossKobo: 100, NetKobo: 100}}
	id, err = f.finance.ImportSettlement(f.ctx, initiator, outgoing)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.finance.ReconcileSettlement(f.ctx, approver, id, "Verified provider payable settled"); err != nil {
		t.Fatal(err)
	}
	if f.account("provider_payable") != 0 {
		t.Fatal("provider payable not cleared")
	}
	v := f.user("")
	if _, err = f.wallet.CreateDeposit(f.ctx, v.ID, "unmatched-funding", "123.45"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.wallet.FinalizeDeposit(f.ctx, "unmatched-funding", "99922", "123.45", "NGN", "successful", nil); err != nil {
		t.Fatal(err)
	}
	f.balance(v, 0)
	if f.account("funding_unmatched") != -12345 {
		t.Fatal("unmatched funds lost")
	}
	allocation := &models.FeeQuote{ID: f.key(), UserID: v.ID, Kind: "funding_allocation", Currency: "NGN", AmountKobo: 7345, ServiceKobo: 5000, WalletCreditKobo: 7345, CheckoutTotalKobo: 12345, PolicyID: q.PolicyID, AllocationReference: "unmatched-funding", ProviderEvidence: `{"source":"customer_confirmed_existing_funds"}`}
	if err = f.wallet.SaveFeeQuote(f.ctx, allocation); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err = f.wallet.AllocateFunding(f.ctx, v.ID, allocation.ID); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(v, 7345)
	if f.account("funding_unmatched") != 0 {
		t.Fatal("allocation liability not cleared")
	}
	if _, err = f.wallet.FinalizeDeposit(f.ctx, "unmatched-funding", "99922", "123.45", "NGN", "reversed", nil); err != nil {
		t.Fatal(err)
	}
	f.balance(v, 0)
	f.balanced()
}
