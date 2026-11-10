package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"backend/internal/models"
)

var (
	ErrBankAccountNumber           = errors.New("enter the ten digits of your bank account number")
	ErrBankSelection               = errors.New("choose a bank from the list")
	ErrBankVerificationUnavailable = errors.New("bank verification is temporarily unavailable; please try again")
	ErrBankVerificationRequired    = errors.New("check the account name before saving")
	ErrBankAccountOwnerMismatch    = errors.New("the bank account name does not match your registered name; use an account in your name or correct your profile")
	ErrBankAccountNameChanged      = errors.New("the bank returned a different account name; check the name again before saving")
)

// BankAccountVerification contains only details returned by the selected bank.
// A name lookup does not save an account or authorize a withdrawal.
type BankAccountVerification struct {
	BankCode       string `json:"bank_code"`
	BankName       string `json:"bank_name"`
	AccountNumber  string `json:"account_number"`
	AccountName    string `json:"account_name"`
	MatchesProfile bool   `json:"matches_profile"`
}

func (s *WalletService) ResolveBankAccount(ctx context.Context, user *models.User, bankCode, number string) (*BankAccountVerification, error) {
	if user == nil || user.ID <= 0 {
		return nil, errors.New("please sign in")
	}
	bankCode, number = strings.TrimSpace(bankCode), strings.TrimSpace(number)
	if !accountNumber.MatchString(number) {
		return nil, ErrBankAccountNumber
	}
	if bankCode == "" || len(bankCode) > 64 {
		return nil, ErrBankSelection
	}
	if !s.ProviderAvailable() {
		return nil, ErrBankVerificationUnavailable
	}
	banks, err := s.provider.ListBanks(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBankVerificationUnavailable, err)
	}
	bankName := ""
	for _, bank := range banks {
		if bank.Code == bankCode {
			bankName = bank.Name
			break
		}
	}
	if bankName == "" {
		return nil, ErrBankSelection
	}
	account, err := s.provider.ResolveAccount(ctx, number, bankCode)
	if errors.Is(err, ErrBankAccountUnconfirmed) || errors.Is(err, ErrProviderRefused) || errors.Is(err, ErrProviderNotFound) {
		return nil, ErrBankAccountUnconfirmed
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBankVerificationUnavailable, err)
	}
	return &BankAccountVerification{
		BankCode: bankCode, BankName: bankName,
		AccountNumber: account.AccountNumber, AccountName: account.AccountName,
		MatchesProfile: sameAccountOwner(user.FullName, account.AccountName),
	}, nil
}

func (s *WalletService) SaveBank(ctx context.Context, userID int, bankCode, number, confirmedName, pin string) error {
	if err := s.verifyPIN(ctx, userID, pin); err != nil {
		return err
	}
	if strings.TrimSpace(confirmedName) == "" {
		return ErrBankVerificationRequired
	}
	user, err := s.profiles.GetPaymentProfile(userID, s.currency)
	if err != nil {
		return err
	}
	// Resolve again at save time. Never persist a name supplied by the browser
	// or reuse a lookup for another bank, number, or registered account holder.
	account, err := s.ResolveBankAccount(ctx, user, bankCode, number)
	if err != nil {
		return err
	}
	if !account.MatchesProfile {
		return ErrBankAccountOwnerMismatch
	}
	if !equalAccountNames(confirmedName, account.AccountName) {
		return ErrBankAccountNameChanged
	}
	return s.profiles.UpsertBankAccount(&models.UserBankAccount{
		UserID: userID, BankName: account.BankName, BankCode: account.BankCode,
		AccountName: account.AccountName, AccountNumber: account.AccountNumber,
	})
}
