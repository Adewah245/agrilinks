package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/repository"
)

type ProviderFeeQuote struct {
	FeeKobo, ChargeKobo int64
	Evidence            string
}

func (c *FlutterwaveClient) WithdrawalFee(ctx context.Context, amount, currency string) (*ProviderFeeQuote, error) {
	var response struct {
		Data []struct {
			FeeType  string      `json:"fee_type"`
			Currency string      `json:"currency"`
			Fee      json.Number `json:"fee"`
		} `json:"data"`
	}
	if err := c.walletRequest(ctx, "GET", "/transfers/fee?"+url.Values{"amount": {amount}, "currency": {currency}, "type": {"account"}}.Encode(), nil, &response); err != nil {
		return nil, err
	}
	var found *ProviderFeeQuote
	for _, item := range response.Data {
		if item.Currency != currency {
			continue
		}
		if found != nil || item.FeeType != "value" {
			return nil, errors.New("provider returned an ambiguous transfer fee")
		}
		n, err := money.Parse(item.Fee.String())
		if err != nil {
			return nil, errors.New("provider did not return a valid transfer fee")
		}
		raw, _ := json.Marshal(item)
		found = &ProviderFeeQuote{FeeKobo: n, Evidence: string(raw)}
	}
	if found == nil {
		return nil, errors.New("provider did not quote a fee for this currency")
	}
	return found, nil
}
func (c *FlutterwaveClient) CollectionFee(ctx context.Context, amount, currency, method string) (*ProviderFeeQuote, error) {
	paymentType := "bank_transfer"
	if method == "card" {
		paymentType = "card"
	} else if method != "banktransfer" {
		return nil, errors.New("choose card or bank transfer")
	}
	var response struct {
		Data struct {
			Fee         json.Number `json:"fee"`
			Charge      json.Number `json:"charge_amount"`
			Currency    string      `json:"currency"`
			Merchant    json.Number `json:"merchant_fee"`
			Flutterwave json.Number `json:"flutterwave_fee"`
			StampDuty   json.Number `json:"stamp_duty_fee"`
		} `json:"data"`
	}
	if err := c.walletRequest(ctx, "GET", "/transactions/fee?"+url.Values{"amount": {amount}, "currency": {currency}, "payment_type": {paymentType}}.Encode(), nil, &response); err != nil {
		return nil, err
	}
	fee, err := money.Parse(response.Data.Fee.String())
	if err != nil {
		return nil, errors.New("provider did not return a collection fee")
	}
	total, err := money.Positive(response.Data.Charge.String())
	if err != nil || response.Data.Currency != currency {
		return nil, errors.New("provider did not return a valid collection total")
	}
	principal, err := money.Positive(amount)
	if err != nil || total < principal {
		return nil, errors.New("provider collection total does not match the deposit")
	}
	evidence, _ := json.Marshal(response.Data)
	return &ProviderFeeQuote{FeeKobo: fee, ChargeKobo: total, Evidence: string(evidence)}, nil
}

func sameAccountOwner(profile, bank string) bool {
	normalize := func(s string) string {
		words := strings.Fields(strings.ToUpper(strings.Map(func(r rune) rune {
			if r == '.' || r == ',' || r == '-' {
				return ' '
			}
			return r
		}, s)))
		out := []string{}
		for _, w := range words {
			if w != "MR" && w != "MRS" && w != "MS" && w != "DR" {
				out = append(out, w)
			}
		}
		if len(out) < 2 {
			return ""
		}
		sort.Strings(out)
		return strings.Join(out, " ")
	}
	p := normalize(profile)
	return p != "" && p == normalize(bank)
}

func (s *WalletService) Quote(ctx context.Context, user *models.User, kind, amount, method string) (*models.FeeQuote, error) {
	if user == nil || user.ID <= 0 {
		return nil, errors.New("please sign in")
	}
	if s.currency != "NGN" {
		return nil, errors.New("wallet fee quotes currently support NGN")
	}
	n, err := money.Positive(amount)
	if err != nil {
		return nil, err
	}
	policy, err := s.repo.FeePolicy(ctx)
	if err != nil {
		return nil, err
	}
	var key [16]byte
	if _, err = rand.Read(key[:]); err != nil {
		return nil, err
	}
	q := &models.FeeQuote{ID: hex.EncodeToString(key[:]), UserID: user.ID, Kind: kind, Currency: s.currency, AmountKobo: n, PolicyID: policy.ID, ProviderEvidence: "{}"}
	switch kind {
	case "withdrawal":
		if !s.ProviderAvailable() {
			return nil, ErrPaymentProvider
		}
		q.ServiceKobo, err = policy.WithdrawalFee(n)
		if err != nil {
			return nil, err
		}
		profile, err := s.profiles.GetPaymentProfile(user.ID, s.currency)
		if err != nil {
			return nil, err
		}
		if profile == nil || profile.AccountNumber == "" {
			return nil, errors.New("save your own bank account before withdrawing")
		}
		resolved, err := s.provider.ResolveAccount(ctx, profile.AccountNumber, profile.BankCode)
		if err != nil {
			return nil, err
		}
		if !sameAccountOwner(user.FullName, resolved.AccountName) || !strings.EqualFold(strings.Join(strings.Fields(profile.AccountName), " "), strings.Join(strings.Fields(resolved.AccountName), " ")) {
			return nil, errors.New("withdrawals must go to a verified bank account matching your registered name")
		}
		fee, err := s.provider.WithdrawalFee(ctx, money.Format(n), s.currency)
		if err != nil {
			return nil, errors.New("provider charges could not be confirmed; please try the fee quote again later")
		}
		tax, err := money.BasisPoints(fee.FeeKobo, policy.ProviderQuoteTaxBPS)
		if err != nil {
			return nil, err
		}
		q.ProviderKobo, err = money.Add(fee.FeeKobo, tax)
		if err != nil {
			return nil, err
		}
		if policy.WithdrawalProviderBearer == "customer" {
			q.CustomerProviderKobo = q.ProviderKobo
		}
		q.WalletDebitKobo, err = money.Add(n, q.ServiceKobo, q.CustomerProviderKobo)
		if err != nil {
			return nil, err
		}
		q.BankCreditKobo = n
		q.BankCode = profile.BankCode
		q.BankName = profile.BankName
		q.AccountNumber = profile.AccountNumber
		q.AccountName = resolved.AccountName
		q.ProviderEvidence = fee.Evidence
	case "deposit", "funding_preview":

		q.ServiceKobo = policy.FundingServiceKobo
		q.WalletCreditKobo = n
		q.CheckoutTotalKobo, err = money.Add(n, q.ServiceKobo)
		if err != nil {
			return nil, err
		}
		q.PaymentMethod = method
		if kind == "deposit" {
			if !s.ProviderAvailable() {
				return nil, ErrPaymentProvider
			}
			fee, err := s.provider.CollectionFee(ctx, money.Format(q.CheckoutTotalKobo), s.currency, method)
			if err != nil {
				return nil, errors.New("provider charges could not be confirmed; please try the fee quote again later")
			}
			q.ProviderKobo = fee.FeeKobo
			q.CheckoutTotalKobo = fee.ChargeKobo
			q.CustomerProviderKobo = fee.ChargeKobo - n - q.ServiceKobo
			q.ProviderEvidence = fee.Evidence
		} else {
			q.ProviderEvidence = `{"mode":"merchant_collection_costs_absorbed","external_sender_bank_fees_not_controlled":true}`
		}
	default:
		return nil, errors.New("choose a funding or withdrawal quote")
	}
	if err = s.repo.SaveFeeQuote(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *WalletService) FeePolicy(ctx context.Context) (*models.FeePolicy, error) {
	return s.repo.FeePolicy(ctx)
}
func providerDepositCosts(v *VerifiedTransaction) (repository.DepositCosts, error) {
	var costs repository.DepositCosts
	if v.AppFee != "" {
		n, err := money.Parse(v.AppFee.String())
		if err != nil {
			return costs, err
		}
		costs.ProviderKobo = &n
	}
	if v.ChargedAmount != "" {
		n, err := money.Positive(v.ChargedAmount.String())
		if err != nil {
			return costs, err
		}
		costs.ChargedKobo = &n
	}
	return costs, nil
}

func (s *WalletService) QuoteUnallocatedFunding(ctx context.Context, user *models.User, reference string) (*models.FeeQuote, error) {
	if user == nil {
		return nil, errors.New("please sign in")
	}
	held, err := s.repo.UnallocatedFunding(ctx, user.ID, reference)
	if err != nil {
		return nil, err
	}
	policy, err := s.repo.FeePolicy(ctx)
	if err != nil {
		return nil, err
	}
	if held <= policy.FundingServiceKobo {
		return nil, errors.New("this receipt is too small to cover the funding fee; contact support for return to the source account")
	}
	key, err := financialKey()
	if err != nil {
		return nil, err
	}
	q := &models.FeeQuote{ID: key, UserID: user.ID, Kind: "funding_allocation", Currency: "NGN", AmountKobo: held - policy.FundingServiceKobo, WalletCreditKobo: held - policy.FundingServiceKobo, ServiceKobo: policy.FundingServiceKobo, CheckoutTotalKobo: held, PolicyID: policy.ID, AllocationReference: reference, ProviderEvidence: `{"source":"customer_accepted_allocation_of_existing_receipt","provider_costs_absorbed":true}`}
	if err = s.repo.SaveAllocationQuote(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}
func (s *WalletService) AllocateFunding(ctx context.Context, userID int, key string) error {
	return s.repo.AllocateFunding(ctx, userID, key)
}
