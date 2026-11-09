package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var confirmationToken = regexp.MustCompile(`^[a-f0-9]{64}$`)

type WalletConfirmationInput struct{ Kind, Recipient, Amount, Note, PIN, Key, Quote string }

func (s *WalletService) StartConfirmation(ctx context.Context, user *models.User, session string, input WalletConfirmationInput) (string, error) {
	if user == nil || user.ID <= 0 || session == "" {
		return "", repository.ErrWalletConfirmation
	}
	if err := s.verifyPIN(ctx, user.ID, input.PIN); err != nil {
		return "", err
	}
	if !s.security.Available() {
		return "", ErrSecurityUnavailable
	}
	if err := s.security.Limit(ctx, user.ID, "wallet_confirmation_start", 12, time.Minute); err != nil {
		return "", err
	}
	if !s.otp.Available() {
		return "", errors.New("code delivery is not set up yet")
	}
	amount, err := normalizedMoney(input.Amount)
	if err != nil {
		return "", err
	}
	if !requestToken.MatchString(input.Key) || (input.Kind != "transfer" && input.Kind != "withdrawal") {
		return "", repository.ErrWalletConfirmation
	}
	c := &models.WalletConfirmation{UserID: user.ID, SessionHash: SecurityHash(session), Kind: input.Kind, Currency: s.currency, Amount: amount, MaxAttempts: s.otp.maxAttempts}
	if c.MaxAttempts < 1 || c.MaxAttempts > 10 {
		c.MaxAttempts = DefaultOTPMaxAttempts
	}
	ttl := s.otp.ttl
	if ttl <= 0 || ttl > 10*time.Minute {
		ttl = DefaultOTPTTL
	}
	c.ExpiresAt = time.Now().Add(ttl)
	debitAmount := amount
	if input.Kind == "transfer" {
		payee, err := s.ResolveRecipient(ctx, user.ID, input.Recipient)
		if err != nil {
			return "", err
		}
		c.RecipientWallet = payee.WalletNumber
		c.Note = strings.TrimSpace(input.Note)
		if len(c.Note) > 1000 {
			return "", errors.New("use a shorter transfer note")
		}
		c.Reference = fmt.Sprintf("TRANSFER-%d-%s", user.ID, input.Key)
	} else {
		if input.Key != input.Quote {
			return "", repository.ErrWalletConfirmation
		}
		q, err := s.confirmationWithdrawalQuote(ctx, user, input.Quote, amount)
		if err != nil {
			return "", err
		}
		c.QuoteID = q.ID
		c.BankCode = q.BankCode
		c.AccountNumber = q.AccountNumber
		c.Reference = fmt.Sprintf("WITHDRAW-%d-%s", user.ID, input.Key)
		if q.ExpiresAt.Before(c.ExpiresAt) {
			c.ExpiresAt = q.ExpiresAt
		}
		debitAmount = q.WalletDebit()
	}
	available, err := s.repo.AvailableConfirmationBalance(ctx, user.ID, debitAmount)
	if err != nil {
		return "", err
	}
	if !available {
		return "", repository.ErrInsufficientBalance
	}
	token, err := securityToken(32)
	if err != nil {
		return "", err
	}
	c.TokenHash = SecurityHash(token)
	code, err := generateOTPCode(otpCodeDigits)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), s.otp.hashCost)
	if err != nil {
		return "", err
	}
	c.OTPHash = string(hash)
	if err = s.repo.CreateConfirmation(ctx, c, s.otp.resendInterval); err != nil {
		return "", err
	}
	if err = s.otp.delivery.SendOTP(ctx, OTPMessage{UserID: user.ID, Phone: strings.TrimSpace(user.Phone), Email: strings.TrimSpace(user.Email), Purpose: input.Kind, Code: code, ExpiresAt: c.ExpiresAt}); err != nil {
		_ = s.repo.ConfirmationDelivered(context.WithoutCancel(ctx), c.TokenHash, false)
		return "", errors.New("could not deliver your confirmation code; please try again")
	}
	if err = s.repo.ConfirmationDelivered(ctx, c.TokenHash, true); err != nil {
		return "", err
	}
	return token, nil
}
func (s *WalletService) VerifyConfirmation(ctx context.Context, userID int, session, token, code string) error {
	if !confirmationToken.MatchString(token) || session == "" {
		return repository.ErrWalletConfirmation
	}
	if err := s.security.Limit(ctx, userID, "wallet_confirmation_verify", 30, time.Minute); err != nil {
		return err
	}
	return s.repo.VerifyConfirmation(ctx, userID, SecurityHash(session), SecurityHash(token), strings.TrimSpace(code))
}
func (s *WalletService) CompleteConfirmation(ctx context.Context, user *models.User, session, token string, image []byte, e models.WithdrawalEvidence) (*models.WalletConfirmationResult, error) {
	if user == nil || !confirmationToken.MatchString(token) || session == "" {
		return nil, repository.ErrWalletConfirmation
	}
	if !s.security.Available() {
		return nil, ErrSecurityUnavailable
	}
	if err := s.security.Limit(ctx, user.ID, "wallet_confirmation_complete", 30, time.Minute); err != nil {
		return nil, err
	}
	hash, sessionHash := SecurityHash(token), SecurityHash(session)
	c, err := s.repo.Confirmation(ctx, user.ID, sessionHash, hash)
	if err != nil {
		return nil, err
	}
	if c.FinalReference.Valid {
		return s.repo.CompleteConfirmation(ctx, user.ID, sessionHash, hash, nil, e, nil)
	}
	if !c.VerifiedAt.Valid || c.InvalidatedAt.Valid || !time.Now().Before(c.ExpiresAt) {
		return nil, repository.ErrWalletConfirmation
	}
	if err = validateVerificationCapture(token, e); err != nil {
		return nil, err
	}
	encrypted, err := s.security.PrivateImage(image)
	if err != nil {
		return nil, err
	}
	var quote *models.FeeQuote
	if c.Kind == "withdrawal" {
		quote, err = s.confirmationWithdrawalQuote(ctx, user, c.QuoteID, c.Amount)
		if err != nil {
			return nil, err
		}
	}
	result, err := s.repo.CompleteConfirmation(ctx, user.ID, sessionHash, hash, encrypted, e, quote)
	if err == nil {
		s.security.Event(ctx, user.ID, "wallet_"+result.Kind+"_confirmed", "info")
	}
	return result, err
}
func (s *WalletService) confirmationWithdrawalQuote(ctx context.Context, user *models.User, id, amount string) (*models.FeeQuote, error) {
	if !s.ProviderAvailable() {
		return nil, ErrPaymentProvider
	}
	q, err := s.repo.FeeQuote(ctx, user.ID, id, "withdrawal", false)
	if err != nil {
		return nil, err
	}
	if q.Amount() != amount || q.Currency != s.currency {
		return nil, repository.ErrWalletRequestChanged
	}
	profile, err := s.profiles.GetPaymentProfile(user.ID, s.currency)
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.BankCode != q.BankCode || profile.AccountNumber != q.AccountNumber {
		return nil, errors.New("your bank account changed; review a new quote")
	}
	resolved, err := s.provider.ResolveAccount(ctx, q.AccountNumber, q.BankCode)
	if err != nil {
		return nil, err
	}
	if !sameAccountOwner(user.FullName, resolved.AccountName) || !sameAccountOwner(q.AccountName, resolved.AccountName) {
		return nil, errors.New("the payout account no longer matches your registered name")
	}
	policy, err := s.repo.FeePolicy(ctx)
	if err != nil {
		return nil, err
	}
	if policy.ID != q.PolicyID {
		return nil, errors.New("the fee policy changed; review a new quote")
	}
	fee, err := s.provider.WithdrawalFee(ctx, q.Amount(), q.Currency)
	if err != nil {
		return nil, err
	}
	tax, err := money.BasisPoints(fee.FeeKobo, policy.ProviderQuoteTaxBPS)
	if err != nil {
		return nil, err
	}
	total, err := money.Add(fee.FeeKobo, tax)
	if err != nil {
		return nil, err
	}
	if total != q.ProviderKobo {
		return nil, errors.New("provider charges changed; review a new quote before withdrawing")
	}
	return q, nil
}
