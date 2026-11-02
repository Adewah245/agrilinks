package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// SettlementProvider isolates treasury from any provider's transport and fee
// schedule. Tests inject a simulated provider; production uses the configured
// Flutterwave client. Statement imports use the same normalized repository API.
type SettlementProvider interface {
	WithdrawalFee(context.Context, string, string) (*ProviderFeeQuote, error)
	ResolveAccount(context.Context, string, string) (*ResolvedAccount, error)
	CreateWalletTransfer(context.Context, string, string, string, string, string, string) (string, error)
	VerifyWalletTransfer(context.Context, string) (*VerifiedTransfer, error)
	FindTransfer(context.Context, string) (*VerifiedTransfer, error)
}

func (c *FlutterwaveClient) FindTransfer(ctx context.Context, reference string) (*VerifiedTransfer, error) {
	var response struct {
		Data []VerifiedTransfer `json:"data"`
	}
	if err := c.walletRequest(ctx, "GET", "/transfers?"+url.Values{"reference": {reference}}.Encode(), nil, &response); err != nil {
		return nil, err
	}
	var found *VerifiedTransfer
	for _, v := range response.Data {
		if v.Reference == reference {
			if found != nil {
				return nil, errors.New("provider returned multiple transfers for this reference")
			}
			copy := v
			found = &copy
		}
	}
	if found == nil {
		return nil, ErrProviderNotFound
	}
	return c.VerifyWalletTransfer(ctx, found.ID.String())
}

type FinanceService struct {
	Repo     *repository.FinanceRepository
	provider SettlementProvider
	users    *repository.UserRepository
	pins     *TransactionPINService
	otp      *OTPService
}

func NewFinanceService(repo *repository.FinanceRepository, provider SettlementProvider, users *repository.UserRepository, pins *TransactionPINService, otp *OTPService) *FinanceService {
	return &FinanceService{Repo: repo, provider: provider, users: users, pins: pins, otp: otp}
}
func financialKey() (string, error) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(key[:]), nil
}
func (s *FinanceService) SendCode(ctx context.Context, user *models.User) error {
	if user == nil || !user.CanAdmin("treasury") {
		return repository.ErrAdminForbidden
	}
	if s.otp == nil {
		return errors.New("administrator code delivery is unavailable")
	}
	return s.otp.Issue(ctx, user.ID, user.Phone, user.Email, "finance_action")
}
func (s *FinanceService) Authenticate(ctx context.Context, user *models.User, password, pin, code string) error {
	if user == nil || !user.CanAdmin("treasury") || s.users == nil || s.pins == nil || s.otp == nil {
		return repository.ErrAdminForbidden
	}
	if err := s.pins.Verify(ctx, user.ID, pin); err != nil {
		return err
	}
	current, err := s.users.GetByID(user.ID)
	if err != nil {
		return err
	}
	if current == nil || !current.CanAdmin("treasury") || bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(password)) != nil {
		return errors.New("current administrator password is required")
	}
	return s.otp.Verify(user.ID, "finance_action", code)
}
func (s *FinanceService) Quote(ctx context.Context, user *models.User, amount string) (*models.CompanyQuote, error) {
	if user == nil || !user.CanAdmin("treasury") {
		return nil, repository.ErrAdminForbidden
	}
	if s.provider == nil {
		return nil, ErrPaymentProvider
	}
	n, err := money.Positive(amount)
	if err != nil {
		return nil, err
	}
	p, err := s.Repo.TransferPolicy(ctx)
	if err != nil {
		return nil, err
	}
	if !p.CompanyAccountID.Valid {
		return nil, errors.New("a company business bank account needs two-administrator approval first")
	}
	if n > p.PerTransferLimit {
		return nil, errors.New("amount exceeds the approved transfer limit")
	}
	bank, err := s.Repo.CompanyAccount(ctx, p.CompanyAccountID.Int64)
	if err != nil {
		return nil, err
	}
	resolved, err := s.provider.ResolveAccount(ctx, bank.AccountNumber, bank.BankCode)
	if err != nil {
		return nil, err
	}
	if !equalAccountNames(resolved.AccountName, bank.AccountName) {
		return nil, errors.New("provider bank resolution no longer matches the approved business account")
	}
	fee, err := s.provider.WithdrawalFee(ctx, money.Format(n), "NGN")
	if err != nil {
		return nil, err
	}
	key, err := financialKey()
	if err != nil {
		return nil, err
	}
	total, err := money.Add(n, fee.FeeKobo)
	if err != nil {
		return nil, err
	}
	_, available, err := s.Repo.Available(ctx)
	if err != nil {
		return nil, err
	}
	if total > available {
		return nil, errors.New("insufficient reconciled, settled and unrestricted provider funds")
	}
	q := &models.CompanyQuote{ID: key, InitiatedBy: user.ID, AmountKobo: n, ProviderKobo: fee.FeeKobo, TotalKobo: total, CompanyAccountID: bank.ID, PolicyID: p.ID, ProviderEvidence: fee.Evidence}
	if err = s.Repo.SaveCompanyQuote(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}
func equalAccountNames(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}
func (s *FinanceService) VerifyCompanyBank(ctx context.Context, p *repository.ChangePayload) error {
	if s.provider == nil {
		return ErrPaymentProvider
	}
	if p.CompanyName == "" || len(p.Evidence) < 5 {
		return errors.New("company legal name and business-account ownership evidence are required")
	}
	account, err := s.provider.ResolveAccount(ctx, p.AccountNumber, p.BankCode)
	if err != nil {
		return err
	}
	if !equalAccountNames(p.CompanyName, account.AccountName) {
		return errors.New("the resolved bank account must match the company's legal business name")
	}
	p.AccountName = account.AccountName
	return nil
}
func (s *FinanceService) DecideChange(ctx context.Context, user *models.User, id int64, approve bool, reason string) error {
	c, err := s.Repo.Change(ctx, id)
	if err != nil {
		return err
	}
	if approve && c.Kind == "company_bank" {
		var p repository.ChangePayload
		if err = json.Unmarshal([]byte(c.Payload), &p); err != nil {
			return err
		}
		previous := p.AccountName
		if err = s.VerifyCompanyBank(ctx, &p); err != nil {
			return err
		}
		if !equalAccountNames(previous, p.AccountName) {
			return errors.New("the bank destination has changed since proposal")
		}
	}
	return s.Repo.DecideChange(ctx, user, id, approve, reason)
}
func (s *FinanceService) Execute(ctx context.Context, user *models.User, id int64) error {
	if user == nil || !user.CanAdmin("treasury_approve") {
		return repository.ErrAdminForbidden
	}
	if s.provider == nil {
		return ErrPaymentProvider
	}
	w, err := s.Repo.CompanyWithdrawal(ctx, id)
	if err != nil {
		return err
	}
	if w.Status != "approved" {
		return errors.New("only an approved transfer can be executed")
	}
	bank, err := s.Repo.CompanyAccount(ctx, w.CompanyAccountID)
	if err != nil {
		return err
	}
	resolved, err := s.provider.ResolveAccount(ctx, bank.AccountNumber, bank.BankCode)
	if err != nil {
		return err
	}
	if !equalAccountNames(resolved.AccountName, bank.AccountName) {
		return errors.New("the company bank resolution changed")
	}
	fee, err := s.provider.WithdrawalFee(ctx, w.Amount(), w.Currency)
	if err != nil {
		return err
	}
	if fee.FeeKobo > w.ProviderKobo {
		return errors.New("provider charges increased; cancel this approved request and obtain a fresh approval")
	}
	if err = s.Repo.ClaimCompanyWithdrawal(ctx, id, user); err != nil {
		return err
	}
	providerID, err := s.provider.CreateWalletTransfer(ctx, w.Reference, w.Amount(), w.Currency, bank.BankCode, bank.AccountNumber, bank.AccountName)
	if err != nil {
		if errors.Is(err, ErrProviderRefused) {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if e := s.Repo.FinishCompanyWithdrawal(cleanup, id, "failed", 0, `{"source":"definitive_provider_refusal"}`); e != nil {
				return e
			}
		}
		return errors.New("provider submission needs verification; do not submit a second transfer")
	}
	if err = s.Repo.SaveCompanyTransferID(ctx, id, providerID); err != nil {
		return err
	}
	return s.Check(ctx, id)
}
func (s *FinanceService) verified(ctx context.Context, w *models.CompanyWithdrawal) (*VerifiedTransfer, error) {
	if s.provider == nil {
		return nil, ErrPaymentProvider
	}
	var v *VerifiedTransfer
	var err error
	if w.ProviderID == "" {
		v, err = s.provider.FindTransfer(ctx, w.Reference)
	} else {
		v, err = s.provider.VerifyWalletTransfer(ctx, w.ProviderID)
	}
	if err != nil {
		return nil, err
	}
	n, err := money.Positive(v.Amount.String())
	if err != nil {
		return nil, err
	}
	bank, err := s.Repo.CompanyAccount(ctx, w.CompanyAccountID)
	if err != nil {
		return nil, err
	}
	if n != w.AmountKobo || v.Currency != w.Currency || v.Reference != w.Reference || v.AccountNumber != bank.AccountNumber || v.BankCode != bank.BankCode {
		return nil, errors.New("provider transfer identity, amount or bank destination mismatch")
	}
	if err = s.Repo.SaveCompanyTransferID(ctx, w.ID, v.ID.String()); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *FinanceService) Check(ctx context.Context, id int64) error {
	w, err := s.Repo.CompanyWithdrawal(ctx, id)
	if err != nil {
		return err
	}
	if w.Status != "processing" && w.Status != "successful" {
		return nil
	}
	v, err := s.verified(ctx, w)
	if err != nil {
		return err
	}
	fee, err := money.Parse(v.Fee.String())
	if err != nil {
		return errors.New("provider has not supplied the actual transfer cost")
	}
	evidence, _ := json.Marshal(v)
	switch strings.ToLower(v.Status) {
	case "failed":
		if w.Status == "successful" {
			return errors.New("provider outcome conflicts with confirmed bank settlement")
		}
		return s.Repo.FinishCompanyWithdrawal(ctx, id, "failed", fee, string(evidence))
	case "reversed":
		return s.Repo.FinishCompanyWithdrawal(ctx, id, "reversed", fee, string(evidence))
	case "successful", "success":
		return nil
	default:
		return nil
	}
}
func (s *FinanceService) ConfirmBankReceipt(ctx context.Context, user *models.User, id int64, statement, amount, reason string) error {
	if user == nil || !user.CanAdmin("treasury_approve") {
		return repository.ErrAdminForbidden
	}
	if len(strings.TrimSpace(reason)) < 5 {
		return errors.New("record the bank settlement evidence")
	}
	w, err := s.Repo.CompanyWithdrawal(ctx, id)
	if err != nil {
		return err
	}
	v, err := s.verified(ctx, w)
	if err != nil {
		return err
	}
	if v.Status != "successful" && v.Status != "success" {
		return errors.New("the provider has not confirmed a successful payout")
	}
	fee, err := money.Parse(v.Fee.String())
	if err != nil {
		return err
	}
	n, err := money.Positive(amount)
	if err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"source": "provider_verification_and_bank_statement", "provider": v, "statement_reference": statement, "reason": reason})
	return s.Repo.FinishCompanyWithdrawal(ctx, id, "successful", fee, string(evidence), repository.CompanySettlementProof{Actor: user, StatementReference: statement, ReceivedKobo: n, Evidence: string(evidence)})
}
func (s *FinanceService) TransferWebhook(ctx context.Context, id, reference string) error {
	w, err := s.Repo.CompanyWithdrawalByReference(ctx, reference)
	if err != nil {
		return err
	}
	if w.ProviderID != "" && w.ProviderID != id {
		return errors.New("company provider transfer mismatch")
	}
	v, err := s.provider.VerifyWalletTransfer(ctx, id)
	if err != nil {
		return err
	}
	if v.Reference != reference {
		return errors.New("company provider reference mismatch")
	}
	if err = s.Repo.SaveCompanyTransferID(ctx, w.ID, id); err != nil {
		return err
	}
	return s.Check(ctx, w.ID)
}
func (s *FinanceService) RunReconciliation(ctx context.Context) {
	if s == nil || s.provider == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := s.Repo.ProcessingCompanyWithdrawals(ctx)
			if err != nil {
				log.Printf("company reconciliation query: %v", err)
				continue
			}
			for _, w := range rows {
				if err = s.Check(ctx, w.ID); err != nil {
					log.Printf("company reconciliation %s: %v", w.Reference, err)
				}
			}
		}
	}
}
func (s *FinanceService) NewRequestKey() string {
	key, err := financialKey()
	if err != nil {
		log.Print(fmt.Errorf("finance request key: %w", err))
	}
	return key
}
