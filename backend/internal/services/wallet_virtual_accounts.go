package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"backend/internal/models"
)

const VirtualAccountConsentVersion = "2026-10-v1"
const virtualAccountRetryAfter = 2 * time.Minute
const virtualAccountCheckAfter = 30 * time.Second

var identityNumber = regexp.MustCompile(`^[0-9]{11}$`)
var (
	ErrVirtualAccountConsent     = errors.New("agree to share your details with Flutterwave before creating a funding account")
	ErrVirtualAccountIdentity    = errors.New("choose BVN or NIN and enter its 11 digits")
	ErrVirtualAccountProfile     = errors.New("add your full name and a valid email address to your profile first")
	ErrVirtualAccountRejected    = errors.New("Flutterwave could not create the account with those details. Check your identity details and try again")
	ErrVirtualAccountUnavailable = errors.New("funding accounts are temporarily unavailable. Please try again later")
)

type VirtualAccountSetup struct {
	IdentityKind, IdentityNumber string
	Consent                      bool
}

func virtualAccountProfileReady(user *models.User) bool {
	if user == nil || strings.TrimSpace(user.FullName) == "" {
		return false
	}
	email := strings.TrimSpace(user.Email)
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func (s *WalletService) SetupVirtualAccount(ctx context.Context, user *models.User, setup VirtualAccountSetup) (*models.WalletVirtualAccount, error) {
	if user == nil || user.ID <= 0 {
		return nil, errors.New("please sign in")
	}
	if !setup.Consent {
		return nil, ErrVirtualAccountConsent
	}
	kind, number := strings.ToLower(strings.TrimSpace(setup.IdentityKind)), strings.ReplaceAll(strings.TrimSpace(setup.IdentityNumber), " ", "")
	if (kind != "bvn" && kind != "nin") || !identityNumber.MatchString(number) {
		return nil, ErrVirtualAccountIdentity
	}
	name, email := strings.Join(strings.Fields(user.FullName), " "), strings.TrimSpace(user.Email)
	if !virtualAccountProfileReady(user) {
		return nil, ErrVirtualAccountProfile
	}
	if !s.ProviderAvailable() {
		return nil, ErrVirtualAccountUnavailable
	}
	if s.currency != "NGN" {
		return nil, errors.New("permanent funding accounts currently support NGN")
	}
	if _, err := s.repo.Wallet(ctx, user.ID); err != nil {
		return nil, err
	}
	a, claimed, created, err := s.repo.ClaimVirtualAccount(ctx, user.ID, fmt.Sprintf("AGRO-VA-%d", user.ID), kind, VirtualAccountConsentVersion, virtualAccountRetryAfter)
	if err != nil || !claimed {
		return a, err
	}
	if !created && a.Status != models.VirtualAccountFailed {
		reference := a.OrderRef
		if reference == "" {
			reference = a.TxRef
		}
		existing, lookupErr := s.provider.GetPermanentVirtualAccount(ctx, reference)
		if lookupErr == nil {
			if err := s.saveVirtualAccountDetails(ctx, a, existing, name); err != nil {
				return nil, err
			}
			return s.repo.VirtualAccount(ctx, user.ID)
		}
		// The lookup API is keyed by order_ref. A missing result for our own
		// tx_ref cannot prove that a timed-out creation did not open an account.
		// Keep that uncertain attempt pending rather than creating a duplicate.
		if !errors.Is(lookupErr, ErrProviderNotFound) || a.OrderRef == "" {
			return s.repo.VirtualAccount(ctx, user.ID)
		}
	}
	names := strings.Fields(name)
	last := ""
	if len(names) > 1 {
		last = strings.Join(names[1:], " ")
	}
	details, err := s.provider.CreatePermanentVirtualAccount(ctx, CreateVirtualAccountRequest{Email: email, TxRef: a.TxRef, Phone: user.Phone, FirstName: names[0], LastName: last, Narration: "AgroShield-" + name, IdentityKind: kind, IdentityNumber: number})
	if err != nil {
		if errors.Is(err, ErrProviderAccessDenied) {
			if saveErr := s.repo.FailVirtualAccount(ctx, user.ID, "Provider did not permit account creation"); saveErr != nil {
				return nil, saveErr
			}
			return nil, ErrVirtualAccountUnavailable
		}
		if errors.Is(err, ErrProviderRefused) {
			if saveErr := s.repo.FailVirtualAccount(ctx, user.ID, "Provider refused account creation"); saveErr != nil {
				return nil, saveErr
			}
			return nil, ErrVirtualAccountRejected
		}
		// A timeout or unreadable response may still have opened an account.
		return s.repo.VirtualAccount(ctx, user.ID)
	}
	if err := s.saveVirtualAccountDetails(ctx, a, details, name); err != nil {
		return nil, err
	}
	return s.repo.VirtualAccount(ctx, user.ID)
}

func (s *WalletService) saveVirtualAccountDetails(ctx context.Context, a *models.WalletVirtualAccount, details *VirtualAccountDetails, name string) error {
	if details == nil {
		return nil
	}
	if details.TxRef != "" && details.TxRef != a.TxRef {
		return errors.New("the provider funding reference did not match")
	}
	if a.OrderRef != "" && details.OrderRef != "" && a.OrderRef != details.OrderRef {
		return ErrReferenceMismatch
	}
	if a.AccountNumber != "" && details.AccountNumber != "" && a.AccountNumber != strings.TrimSpace(details.AccountNumber) {
		return ErrReferenceMismatch
	}
	currency := strings.ToUpper(strings.TrimSpace(details.Currency))
	if currency == "" {
		currency = "NGN"
	}
	if currency != "NGN" {
		return errors.New("the provider funding currency did not match")
	}
	if details.OrderRef != "" {
		a.OrderRef = details.OrderRef
	}
	if details.ProviderRef != "" {
		a.ProviderRef = details.ProviderRef
	}
	if details.AccountNumber != "" {
		a.AccountNumber = strings.TrimSpace(details.AccountNumber)
	}
	if details.BankName != "" {
		a.BankName = strings.TrimSpace(details.BankName)
	}
	if details.AccountName != "" {
		a.AccountName = strings.TrimSpace(details.AccountName)
	}
	if a.AccountName == "" && name != "" {
		a.AccountName = "AgroShield-" + name
	}
	a.Currency, a.Status, a.FailureReason = currency, models.VirtualAccountPending, ""
	status := strings.ToLower(strings.TrimSpace(details.AccountStatus))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(details.Status))
	}
	usable := status == "" || status == "active" || status == "successful" || status == "success"
	if usable && accountNumber.MatchString(a.AccountNumber) && a.BankName != "" && a.AccountName != "" {
		a.Status = models.VirtualAccountActive
	}
	if a.AccountNumber != "" && !accountNumber.MatchString(a.AccountNumber) {
		a.AccountNumber = ""
	}
	return s.repo.SaveVirtualAccount(ctx, a)
}

func (s *WalletService) RefreshVirtualAccount(ctx context.Context, userID int, name string) (*models.WalletVirtualAccount, error) {
	if userID <= 0 {
		return nil, errors.New("please sign in")
	}
	a, err := s.repo.VirtualAccount(ctx, userID)
	if err != nil || a == nil || a.Status != models.VirtualAccountPending {
		return a, err
	}
	if !s.ProviderAvailable() {
		return a, ErrVirtualAccountUnavailable
	}
	claimed, err := s.repo.ClaimVirtualAccountCheck(ctx, userID, virtualAccountCheckAfter)
	if err != nil || !claimed {
		return a, err
	}
	reference := a.OrderRef
	if reference == "" {
		reference = a.TxRef
	}
	details, err := s.provider.GetPermanentVirtualAccount(ctx, reference)
	if errors.Is(err, ErrProviderNotFound) {
		return a, nil
	}
	if err != nil {
		return a, ErrVirtualAccountUnavailable
	}
	if err := s.saveVirtualAccountDetails(ctx, a, details, name); err != nil {
		return nil, err
	}
	return s.repo.VirtualAccount(ctx, userID)
}
