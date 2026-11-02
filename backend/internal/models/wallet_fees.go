package models

import (
	"backend/internal/money"
	"time"
)

type FeePolicy struct {
	ID                                                                           int64
	FundingServiceKobo, WithdrawalLowKobo, WithdrawalMidKobo, WithdrawalHighKobo int64
	EffectiveAt                                                                  time.Time
	WithdrawalProviderBearer                                                     string
	ProviderQuoteTaxBPS                                                          int64
	Reason                                                                       string
	CreatedAt                                                                    time.Time
}
type FeeQuote struct {
	AllocationReference                                                                                                               string
	ID                                                                                                                                string
	UserID                                                                                                                            int
	Kind, Currency                                                                                                                    string
	AmountKobo, ServiceKobo, ProviderKobo, CustomerProviderKobo, WalletDebitKobo, WalletCreditKobo, BankCreditKobo, CheckoutTotalKobo int64
	PaymentMethod, BankCode, AccountNumber, AccountName, BankName                                                                     string
	PolicyID                                                                                                                          int64
	ProviderEvidence                                                                                                                  string
	ExpiresAt                                                                                                                         time.Time
}

func (q FeeQuote) Amount() string        { return money.Format(q.AmountKobo) }
func (q FeeQuote) ServiceFee() string    { return money.Format(q.ServiceKobo) }
func (q FeeQuote) ProviderFee() string   { return money.Format(q.CustomerProviderKobo) }
func (q FeeQuote) ProviderCost() string  { return money.Format(q.ProviderKobo) }
func (q FeeQuote) WalletDebit() string   { return money.Format(q.WalletDebitKobo) }
func (q FeeQuote) WalletCredit() string  { return money.Format(q.WalletCreditKobo) }
func (q FeeQuote) BankCredit() string    { return money.Format(q.BankCreditKobo) }
func (q FeeQuote) CheckoutTotal() string { return money.Format(q.CheckoutTotalKobo) }

type FeeBreakdown struct {
	ServiceKobo, ProviderKobo, CreditKobo, DebitKobo int64
	State                                            string
}

func (f FeeBreakdown) ServiceFee() string   { return money.Format(f.ServiceKobo) }
func (f FeeBreakdown) ProviderFee() string  { return money.Format(f.ProviderKobo) }
func (f FeeBreakdown) WalletCredit() string { return money.Format(f.CreditKobo) }
func (f FeeBreakdown) WalletDebit() string  { return money.Format(f.DebitKobo) }

func (p FeePolicy) WithdrawalFee(n int64) (int64, error) {
	if _, err := money.WithdrawalFee(n); err != nil {
		return 0, err
	}
	if n <= 500000 {
		return p.WithdrawalLowKobo, nil
	}
	if n < 1000000 {
		return p.WithdrawalMidKobo, nil
	}
	return p.WithdrawalHighKobo, nil
}
