package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"backend/internal/models"
)

func (s *WalletService) Revision(ctx context.Context, userID int) (string, error) {
	if userID <= 0 {
		return "", errors.New("please sign in")
	}
	return s.repo.WalletRevision(ctx, userID)
}

// ReconcileFunding only uses provider-verified state. A timeout or missing
// transaction is never treated as proof that money was not sent.
func (s *WalletService) ReconcileFunding(ctx context.Context) error {
	if !s.ProviderAvailable() {
		return nil
	}
	var failures []error
	for _, phase := range []func(context.Context) error{s.reconcileAccountSetups, s.reconcilePendingFunding, s.reconcileActiveFunding} {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Give every queue a turn even when one provider endpoint is slow.
		phaseCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := phase(phaseCtx)
		cancel()
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *WalletService) reconcileAccountSetups(ctx context.Context) error {
	var failures []error
	users, err := s.repo.VirtualAccountUsers(ctx, 20)
	if err != nil {
		return err
	}
	for _, id := range users {
		name, err := s.repo.VirtualAccountOwnerName(ctx, id)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if _, err := s.RefreshVirtualAccount(ctx, id, name); err != nil {
			failures = append(failures, fmt.Errorf("funding account %d: %w", id, err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return errors.Join(failures...)
}

func (s *WalletService) reconcilePendingFunding(ctx context.Context) error {
	var failures []error
	checks, err := s.repo.ClaimPendingFunding(ctx, 20)
	if err != nil {
		return err
	}
	for _, check := range checks {
		var verified *VerifiedTransaction
		var err error
		if check.ProviderID != "" {
			verified, err = s.provider.VerifyTransaction(ctx, check.ProviderID)
		} else {
			verified, err = s.provider.VerifyTransactionByReference(ctx, check.Reference)
		}
		if err == nil {
			if verified == nil || verified.Reference != check.Reference || (check.ProviderID != "" && verified.TransactionID != check.ProviderID) {
				err = ErrReferenceMismatch
			} else {
				_, err = s.ApplyVerifiedDeposit(ctx, verified, nil)
			}
		}
		if err != nil && !errors.Is(err, ErrProviderNotFound) {
			failures = append(failures, fmt.Errorf("pending funding verification: %w", err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return errors.Join(failures...)
}

func (s *WalletService) reconcileActiveFunding(ctx context.Context) error {
	var failures []error
	accounts, err := s.repo.ClaimActiveFunding(ctx, 20)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		verified, err := s.provider.VerifyTransactionByReference(ctx, account.TxRef)
		if err == nil {
			if verified == nil || verified.Reference != account.TxRef {
				err = ErrReferenceMismatch
			} else {
				_, err = s.ApplyVerifiedDeposit(ctx, verified, nil)
			}
		}
		if err != nil && !errors.Is(err, ErrProviderNotFound) {
			failures = append(failures, fmt.Errorf("funding account %d verification: %w", account.UserID, err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return errors.Join(failures...)
}

// RunFundingReconciliation exits with the server context. Each pass is bounded
// and sequential, so a slow provider cannot create overlapping local workers.
func (s *WalletService) RunFundingReconciliation(ctx context.Context) {
	if !s.ProviderAvailable() {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		pass, cancel := context.WithTimeout(ctx, 45*time.Second)
		err := s.ReconcileFunding(pass)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Printf("wallet funding reconciliation: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RecoverFundingTransfer is also the operator path for an older transfer that
// was not returned by Flutterwave's single-result reference lookup.
func (s *WalletService) RecoverFundingTransfer(ctx context.Context, user *models.User, providerID string) error {
	if user == nil || !user.CanAdmin("finance") {
		return errors.New("finance access is required")
	}
	if !s.ProviderAvailable() {
		return ErrVirtualAccountUnavailable
	}
	verified, err := s.provider.VerifyTransaction(ctx, providerID)
	if err != nil {
		return err
	}
	if verified == nil || verified.TransactionID != providerID {
		return ErrReferenceMismatch
	}
	account, err := s.repo.VirtualAccountByReference(ctx, verified.Reference)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("this transaction does not belong to a funding account")
	}
	_, err = s.ApplyVerifiedDeposit(ctx, verified, nil)
	return err
}
