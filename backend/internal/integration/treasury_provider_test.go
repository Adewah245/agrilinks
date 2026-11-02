package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/services"
	"backend/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

type simulatedTreasury struct {
	mu          sync.Mutex
	calls       int
	outcome     string
	submitError error
	v           services.VerifiedTransfer
}

func (p *simulatedTreasury) WithdrawalFee(context.Context, string, string) (*services.ProviderFeeQuote, error) {
	return &services.ProviderFeeQuote{FeeKobo: 500, Evidence: `{"source":"simulated_provider_fee"}`}, nil
}
func (p *simulatedTreasury) ResolveAccount(_ context.Context, number, bank string) (*services.ResolvedAccount, error) {
	return &services.ResolvedAccount{AccountNumber: number, AccountName: "AgriLinks Limited"}, nil
}
func (p *simulatedTreasury) CreateWalletTransfer(_ context.Context, ref, amount, currency, bank, number, name string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.v = services.VerifiedTransfer{ID: json.Number(fmt.Sprint(88000 + p.calls)), Reference: ref, Amount: json.Number(amount), Currency: currency, AccountNumber: number, BankCode: bank, Fee: "5.00", Status: p.outcome}
	if p.submitError != nil {
		return "", p.submitError
	}
	return p.v.ID.String(), nil
}
func (p *simulatedTreasury) VerifyWalletTransfer(_ context.Context, id string) (*services.VerifiedTransfer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := p.v
	if v.ID.String() != id {
		return nil, services.ErrProviderNotFound
	}
	return &v, nil
}
func (p *simulatedTreasury) FindTransfer(ctx context.Context, ref string) (*services.VerifiedTransfer, error) {
	p.mu.Lock()
	v := p.v
	p.mu.Unlock()
	if v.Reference != ref {
		return nil, services.ErrProviderNotFound
	}
	return &v, nil
}

func TestTreasuryProviderExecutionAuthenticationAndRecovery(t *testing.T) {
	db, _ := testutil.Database(t, true)
	ctx := context.Background()
	c := application(t, db, testConfig(t))
	customer := fixtureUser(t, c, 1, "buyer")
	initiator := fixtureUser(t, c, 2, "buyer")
	approver := fixtureUser(t, c, 3, "buyer")
	support := fixtureUser(t, c, 4, "buyer")
	for _, entry := range []struct {
		u    *models.User
		role string
	}{{initiator, "finance_admin"}, {approver, "super_admin"}, {support, "customer_care"}} {
		if _, err := db.Exec(`UPDATE users SET staff_role=$2 WHERE id=$1`, entry.u.ID, entry.role); err != nil {
			t.Fatal(err)
		}
		entry.u.StaffRole = entry.role
		hash, _ := bcrypt.GenerateFromPassword([]byte("582491"), bcrypt.MinCost)
		if err := c.UserRepo.CreateTransactionPIN(ctx, entry.u.ID, string(hash)); err != nil {
			t.Fatal(err)
		}
	}
	capture := &captureOTP{}
	otp := services.NewOTPService(repository.NewOTPRepository(db), capture)
	otp.Configure(services.DefaultOTPTTL, 5, 0)
	provider := &simulatedTreasury{outcome: "successful"}
	repo := repository.NewFinanceRepository(db)
	service := services.NewFinanceService(repo, provider, c.UserRepo, services.NewTransactionPINService(c.UserRepo), otp)
	c.Finance = service
	c.Payment.SetFinance(service)
	if err := service.Authenticate(ctx, initiator, "test password with spaces", "582491", "123456"); err == nil {
		t.Fatal("unissued OTP accepted")
	}
	if err := service.SendCode(ctx, initiator); err != nil {
		t.Fatal(err)
	}
	if err := service.Authenticate(ctx, initiator, "wrong password", "582491", capture.message.Code); err == nil {
		t.Fatal("wrong administrator password accepted")
	}
	if err := service.Authenticate(ctx, initiator, "test password with spaces", "582491", capture.message.Code); err != nil {
		t.Fatal(err)
	}
	if err := service.Authenticate(ctx, initiator, "test password with spaces", "582491", capture.message.Code); err == nil {
		t.Fatal("finance OTP reused")
	}
	if err := service.SendCode(ctx, support); err == nil {
		t.Fatal("support requested treasury code")
	}
	bank := repository.ChangePayload{CompanyName: "AgriLinks Limited", BankName: "Test Bank", BankCode: "001", AccountNumber: "0123456789", Evidence: "Verified test business account ownership"}
	if err := service.VerifyCompanyBank(ctx, &bank); err != nil {
		t.Fatal(err)
	}
	changeID, err := repo.ProposeChange(ctx, initiator, strings.Repeat("a", 32), "company_bank", "Business settlement destination", bank)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.DecideChange(ctx, approver, changeID, true, "Independent business account verification"); err != nil {
		t.Fatal(err)
	}
	wallet := repository.NewWalletRepository(db, "NGN", 0)
	policy, err := wallet.FeePolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 4; n++ {
		key := fmt.Sprintf("%032x", n)
		q := &models.FeeQuote{ID: key, UserID: customer.ID, Kind: "deposit", Currency: "NGN", AmountKobo: 1000000, WalletCreditKobo: 1000000, ServiceKobo: 5000, CheckoutTotalKobo: 1005000, PolicyID: policy.ID, ProviderEvidence: `{"source":"simulated_collection_quote"}`}
		if err = wallet.SaveFeeQuote(ctx, q); err != nil {
			t.Fatal(err)
		}
		ref := fmt.Sprintf("WALLET-%d-%s", customer.ID, key)
		if _, err = wallet.CreateDeposit(ctx, customer.ID, ref, "10050.00"); err != nil {
			t.Fatal(err)
		}
		zero := int64(0)
		if _, err = wallet.FinalizeDeposit(ctx, ref, fmt.Sprint(50000+n), "10050.00", "NGN", "successful", nil, repository.DepositCosts{ProviderKobo: &zero}); err != nil {
			t.Fatal(err)
		}
		batch := models.SettlementBatch{Provider: "flutterwave", Reference: fmt.Sprintf("batch-%d", n), Currency: "NGN", Destination: "provider_balance", Source: "simulated_provider_statement", Evidence: `{"record":"test"}`, StatementReference: fmt.Sprintf("statement-%d", n), Reason: "Provider collection settlement", GrossKobo: 1005000, NetKobo: 1005000, StatementKobo: 1005000, Items: []models.SettlementItem{{Reference: ref, GrossKobo: 1005000, NetKobo: 1005000}}}
		id, err := repo.ImportSettlement(ctx, initiator, batch)
		if err != nil {
			t.Fatal(err)
		}
		if err = repo.ReconcileSettlement(ctx, approver, id, "Provider and settled balance matched"); err != nil {
			t.Fatal(err)
		}
	}
	requestApproved := func() *models.CompanyWithdrawal {
		t.Helper()
		q, err := service.Quote(ctx, initiator, "20")
		if err != nil {
			t.Fatal(err)
		}
		w, err := repo.RequestCompanyWithdrawal(ctx, initiator, q.ID, "Transfer settled company earnings")
		if err != nil {
			t.Fatal(err)
		}
		if err = repo.ApproveCompanyWithdrawal(ctx, approver, w.ID, true, "Independent approval of source and destination"); err != nil {
			t.Fatal(err)
		}
		return w
	}
	w := requestApproved()
	if err = service.Execute(ctx, initiator, w.ID); err == nil {
		t.Fatal("finance requester executed payment without senior permission")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = service.Execute(ctx, approver, w.ID) }()
	}
	wg.Wait()
	provider.mu.Lock()
	calls := provider.calls
	provider.mu.Unlock()
	if calls != 1 {
		t.Fatalf("provider executions=%d", calls)
	}
	state, err := repo.CompanyWithdrawal(ctx, w.ID)
	if err != nil || state.Status != "processing" {
		t.Fatalf("provider acceptance bypassed bank verification: %+v %v", state, err)
	}
	if err = service.ConfirmBankReceipt(ctx, initiator, w.ID, "business-001", "20", "Self bank confirmation"); err == nil {
		t.Fatal("requester confirmed own settlement")
	}
	if err = service.ConfirmBankReceipt(ctx, approver, w.ID, "business-001", "19", "Wrong bank amount"); err == nil {
		t.Fatal("mismatched bank settlement accepted")
	}
	if err = service.ConfirmBankReceipt(ctx, approver, w.ID, "business-001", "20", "Bank statement confirms exact payout"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err = service.TransferWebhook(ctx, state.ProviderID, state.Reference); err != nil {
			t.Fatal(err)
		}
	}
	state, _ = repo.CompanyWithdrawal(ctx, w.ID)
	if state.Status != "successful" {
		t.Fatal("verified company payout not successful")
	}
	t.Run("ambiguous submission recovers without second payout", func(t *testing.T) {
		provider.mu.Lock()
		provider.submitError = errors.New("simulated network timeout")
		provider.mu.Unlock()
		w := requestApproved()
		if err := service.Execute(ctx, approver, w.ID); err == nil {
			t.Fatal("ambiguous submission reported complete")
		}
		if err := service.Check(ctx, w.ID); err != nil {
			t.Fatal(err)
		}
		if err := service.Execute(ctx, approver, w.ID); err == nil {
			t.Fatal("ambiguous submission retried")
		}
		if err := service.ConfirmBankReceipt(ctx, approver, w.ID, "business-002", "20", "Statement confirms recovered transfer"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("definitive refusal releases reservation", func(t *testing.T) {
		provider.mu.Lock()
		provider.submitError = services.ErrProviderRefused
		provider.mu.Unlock()
		before, sourceBefore, err := repo.Available(ctx)
		if err != nil {
			t.Fatal(err)
		}
		w := requestApproved()
		if err := service.Execute(ctx, approver, w.ID); err == nil {
			t.Fatal("refusal reported success")
		}
		after, sourceAfter, err := repo.Available(ctx)
		if err != nil || after != before || sourceAfter != sourceBefore {
			t.Fatalf("failed company transfer retained reserve: %d/%d %d/%d %v", before, after, sourceBefore, sourceAfter, err)
		}
	})
	handler := routes.NewHandler(c)
	response := request(handler, "GET", "/admin/revenue", nil, session(t, c, approver))
	requireStatus(t, response, 200)
	if !strings.Contains(response.Body.String(), "Completed company transfers") || !strings.Contains(response.Body.String(), "company-bank:") {
		t.Fatal("dashboard omitted real ledger records")
	}
	requireStatus(t, request(handler, "GET", "/admin/revenue", nil, session(t, c, support)), 403)
	var debit, credit int64
	if err = db.QueryRow(`SELECT SUM(debit_kobo),SUM(credit_kobo) FROM accounting_postings`).Scan(&debit, &credit); err != nil || debit != credit {
		t.Fatalf("unbalanced treasury debits=%d credits=%d: %v", debit, credit, err)
	}
	t.Logf("simulated provider workflow balanced: debit=%d credit=%d kobo", debit, credit)
}
