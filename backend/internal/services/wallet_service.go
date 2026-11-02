package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/money"
	"backend/internal/repository"
)

type WalletService struct {
	repo     *repository.WalletRepository
	profiles *repository.PaymentRepository
	provider *FlutterwaveClient
	otp      *OTPService
	currency string
	pins     *TransactionPINService
}

func NewWalletService(repo *repository.WalletRepository, profiles *repository.PaymentRepository, provider *FlutterwaveClient, otp *OTPService, currency string) *WalletService {
	return &WalletService{repo: repo, profiles: profiles, provider: provider, otp: otp, currency: currency}
}

func (s *WalletService) SetOTP(otp *OTPService)  { s.otp = otp }
func (s *WalletService) ProviderAvailable() bool { return s != nil && s.provider != nil }

func (s *WalletService) SetPINService(pins *TransactionPINService) { s.pins = pins }
func (s *WalletService) SetupPIN(ctx context.Context, user *models.User, password, pin, confirmation string) error {
	if s.pins == nil {
		return errors.New("transaction PIN service is unavailable")
	}
	return s.pins.Setup(ctx, user, password, pin, confirmation)
}
func (s *WalletService) verifyPIN(ctx context.Context, userID int, pin string) error {
	if s.pins == nil {
		return errors.New("transaction PIN service is unavailable")
	}
	return s.pins.Verify(ctx, userID, pin)
}

type WalletPage struct {
	Quote, FundingIntent     *models.FeeQuote
	FeeHistory               map[string]models.FeeBreakdown
	Escrows                  []models.Escrow
	HasTransactionPIN        bool
	FailedAction, RequestKey string
	User                     *models.User
	Wallet                   *models.Wallet
	History                  []models.WalletTransaction
	Deposits                 []models.WalletDeposit
	Withdrawals              []models.Withdrawal
	Transfers                []models.Transfer
	VirtualAccount           *models.WalletVirtualAccount
	Profile                  *models.User
	ProviderAvailable        bool
	FundingProfileReady      bool
	Revision                 string
	Error, Notice            string
}

func (s *WalletService) Page(ctx context.Context, user *models.User) (*WalletPage, error) {
	if user == nil || user.ID <= 0 {
		return nil, errors.New("please sign in")
	}
	p := &WalletPage{User: user, ProviderAvailable: s.ProviderAvailable(), FundingProfileReady: virtualAccountProfileReady(user)}
	var err error
	if s.pins != nil {
		p.HasTransactionPIN, err = s.pins.HasPIN(ctx, user.ID)
		if err != nil {
			return nil, err
		}
	}
	// Read the revision before page data. A payment committed during rendering
	// will then be detected by polling rather than leaving a stale balance with
	// the new revision. A newly created wallet triggers one refresh in the same way.
	p.Revision, err = s.Revision(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		p.Revision = "new"
	} else if err != nil {
		return nil, err
	}
	if p.Wallet, err = s.repo.Wallet(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.History, err = s.repo.History(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.Deposits, err = s.repo.Deposits(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.Withdrawals, err = s.repo.Withdrawals(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.Transfers, err = s.repo.Transfers(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.VirtualAccount, err = s.repo.VirtualAccount(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.VirtualAccount != nil && p.VirtualAccount.Currency != s.currency {
		p.VirtualAccount = nil
	}
	if p.Profile, err = s.profiles.GetPaymentProfile(user.ID, s.currency); err != nil {
		return nil, err
	}
	if p.FeeHistory, err = s.repo.FeeHistory(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.FundingIntent, err = s.repo.ActiveFundingIntent(ctx, user.ID); err != nil {
		return nil, err
	}
	if p.Escrows, err = s.repo.ProtectedOrders(ctx, user.ID); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *WalletService) ReviewQuote(ctx context.Context, userID int, key, kind string) (*models.FeeQuote, error) {
	return s.repo.FeeQuote(ctx, userID, key, kind, false)
}
func (s *WalletService) AcceptFundingIntent(ctx context.Context, userID int, key string) error {
	return s.repo.AcceptFundingIntent(ctx, userID, key)
}

var decimalAmount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)
var accountNumber = regexp.MustCompile(`^[0-9]{10}$`)
var requestToken = regexp.MustCompile(`^[a-f0-9]{32}$`)

// Money is read as digits and counted in minor units, never float arithmetic.
func moneyMinorUnits(value string) (int64, error) { return money.Positive(value) }

func normalizedMoney(value string) (string, error) {
	n, err := moneyMinorUnits(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%02d", n/100, n%100), nil
}

func ParseCommissionBasisPoints(value string) (int64, error) {
	if strings.TrimSpace(value) == "" || (decimalAmount.MatchString(strings.TrimSpace(value)) && strings.Trim(strings.TrimSpace(value), "0.") == "") {
		return 0, nil
	}
	// A percentage with two decimal places becomes basis points.
	n, err := moneyMinorUnits(value)
	if err != nil || n > 10000 {
		return 0, errors.New("COMMISSION_PERCENT must be between 0 and 100 with at most two decimal places")
	}
	return n, nil
}

func newWalletReference(prefix string, userID int) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%s", prefix, userID, hex.EncodeToString(b)), nil
}

func (s *WalletService) StartDeposit(ctx context.Context, user *models.User, amount, key string) (*InitiatePaymentResult, error) {
	if !s.ProviderAvailable() {
		return nil, ErrPaymentProvider
	}
	if user == nil || user.ID <= 0 || strings.TrimSpace(user.Email) == "" {
		return nil, ErrPaymentEmailRequired
	}
	if !requestToken.MatchString(key) {
		return nil, errors.New("reload the wallet page before adding money")
	}
	amount, err := normalizedMoney(amount)
	if err != nil {
		return nil, err
	}
	reference := fmt.Sprintf("WALLET-%d-%s", user.ID, key)
	q, err := s.repo.FeeQuote(ctx, user.ID, key, "deposit", true)
	if err != nil {
		return nil, err
	}
	if q.Amount() != amount {
		return nil, repository.ErrWalletRequestChanged
	}
	exists, err := s.repo.DepositRequestExists(ctx, user.ID, reference)
	if err != nil {
		return nil, err
	}
	if !exists {
		if _, err = s.repo.FeeQuote(ctx, user.ID, key, "deposit", false); err != nil {
			return nil, err
		}
		fee, err := s.provider.CollectionFee(ctx, money.Format(q.AmountKobo+q.ServiceKobo), s.currency, q.PaymentMethod)
		if err != nil {
			return nil, err
		}
		if fee.FeeKobo != q.ProviderKobo || fee.ChargeKobo != q.CheckoutTotalKobo {
			return nil, errors.New("provider charges changed; review a new fee quote before confirming")
		}
	}

	d, err := s.repo.CreateDeposit(ctx, user.ID, reference, money.Format(q.AmountKobo+q.ServiceKobo))
	if err != nil {
		return nil, err
	}
	reference = d.Reference
	if d.Status != "pending" {
		return nil, errors.New("this funding request has already been handled")
	}
	if d.CheckoutURL != "" {
		return &InitiatePaymentResult{Reference: reference, CheckoutURL: d.CheckoutURL}, nil
	}
	claimed, err := s.repo.ClaimDepositCheckout(ctx, reference)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errors.New("this payment is already being prepared; check funding history before trying again")
	}
	link, err := s.provider.InitializePayment(ctx, InitializePaymentRequest{Reference: reference, Amount: d.Amount, Currency: d.Currency, CustomerEmail: user.Email, CustomerName: user.FullName, Description: "Add money to your AgriLinks wallet", PaymentOptions: q.PaymentMethod})
	if err != nil {
		return nil, err
	}
	if err = s.repo.SaveDepositURL(ctx, reference, link); err != nil {
		return nil, err
	}
	return &InitiatePaymentResult{Reference: reference, CheckoutURL: link}, nil
}

func (s *WalletService) ApplyVerifiedDeposit(ctx context.Context, v *VerifiedTransaction, event *models.WebhookEvent) (string, error) {
	if v == nil || v.TransactionID == "" {
		return "", errors.New("the provider returned an invalid funding amount")
	}
	amount, err := normalizedMoney(v.Amount.String())
	if err != nil {
		return "", errors.New("the provider returned an invalid funding amount")
	}
	status := verifiedPaymentStatus(v.Status)
	if strings.EqualFold(strings.TrimSpace(v.Status), "reversed") || strings.EqualFold(strings.TrimSpace(v.Status), "refunded") {
		status = "reversed"
	}
	costs, err := providerDepositCosts(v)
	if err != nil {
		return "", err
	}
	_, err = s.repo.FinalizeDeposit(ctx, v.Reference, v.TransactionID, amount, strings.ToUpper(v.Currency), status, event, costs)
	return status, err
}

func (s *WalletService) PayOrder(ctx context.Context, userID, orderID int, pin string) error {
	if userID <= 0 || orderID <= 0 {
		return errors.New("choose an order")
	}
	if err := s.verifyPIN(ctx, userID, pin); err != nil {
		return err
	}
	return s.repo.PayOrder(ctx, userID, orderID)
}

func (s *WalletService) SendCode(ctx context.Context, user *models.User, purpose string) error {
	if user == nil || (purpose != "withdrawal" && purpose != "transfer") {
		return errors.New("choose a wallet action")
	}
	if s.otp == nil || !s.otp.Available() {
		return errors.New("code delivery is not set up yet")
	}
	return s.otp.Issue(ctx, user.ID, user.Phone, user.Email, purpose)
}

func (s *WalletService) RequestTransfer(ctx context.Context, user *models.User, recipient, amount, note, code, pin, key string) error {
	if user == nil || s.otp == nil {
		return errors.New("please sign in")
	}
	amount, err := normalizedMoney(amount)
	if err != nil {
		return err
	}
	recipient = strings.TrimSpace(recipient)
	note = strings.TrimSpace(note)
	if recipient == "" || len(note) > 1000 {
		return errors.New("enter a recipient wallet number and a shorter note")
	}
	if err = s.verifyPIN(ctx, user.ID, pin); err != nil {
		return err
	}
	if !requestToken.MatchString(key) {
		return errors.New("reload the wallet page before sending money")
	}
	reference := fmt.Sprintf("TRANSFER-%d-%s", user.ID, key)
	if found, err := s.repo.TransferMatches(ctx, user.ID, recipient, amount, reference, note); err != nil || found {
		return err
	}
	if err = s.otp.Verify(user.ID, "transfer", code); err != nil {
		return err
	}
	return s.repo.RequestTransfer(ctx, user.ID, recipient, amount, reference, note)
}

func (s *WalletService) DecideTransfer(ctx context.Context, user *models.User, id int64, decision string) error {
	if user == nil || id <= 0 {
		return errors.New("choose a transfer")
	}
	return s.repo.DecideTransfer(ctx, user.ID, user.CanAdmin("finance"), id, decision)
}

func (s *WalletService) Banks(ctx context.Context) ([]Bank, error) {
	if !s.ProviderAvailable() {
		return nil, ErrPaymentProvider
	}
	return s.provider.ListBanks(ctx)
}

func (s *WalletService) SaveBank(ctx context.Context, userID int, bankCode, number, pin string) error {
	if err := s.verifyPIN(ctx, userID, pin); err != nil {
		return err
	}
	if !s.ProviderAvailable() {
		return ErrPaymentProvider
	}
	if !accountNumber.MatchString(number) {
		return errors.New("enter the ten digits of your bank account number")
	}
	banks, err := s.provider.ListBanks(ctx)
	if err != nil {
		return err
	}
	name := ""
	for _, bank := range banks {
		if bank.Code == bankCode {
			name = bank.Name
			break
		}
	}
	if name == "" {
		return errors.New("choose a bank from the list")
	}
	account, err := s.provider.ResolveAccount(ctx, number, bankCode)
	if err != nil {
		return err
	}
	return s.profiles.UpsertBankAccount(&models.UserBankAccount{UserID: userID, BankName: name, BankCode: bankCode, AccountName: account.AccountName, AccountNumber: number})
}

func (s *WalletService) RequestWithdrawal(ctx context.Context, user *models.User, amount, code, pin, key string) error {
	if user == nil || user.ID <= 0 {
		return errors.New("please sign in")
	}
	if !s.ProviderAvailable() {
		return ErrPaymentProvider
	}
	if !requestToken.MatchString(key) {
		return errors.New("review the withdrawal fee quote first")
	}
	n, err := money.Positive(amount)
	if err != nil {
		return err
	}
	if err = s.verifyPIN(ctx, user.ID, pin); err != nil {
		return err
	}
	q, err := s.repo.FeeQuote(ctx, user.ID, key, "withdrawal", true)
	if err != nil {
		return err
	}
	if q.AmountKobo != n {
		return repository.ErrWalletRequestChanged
	}
	reference := fmt.Sprintf("WITHDRAW-%d-%s", user.ID, key)
	if found, err := s.repo.WithdrawalMatches(ctx, user.ID, q.Amount(), reference, q.BankCode, q.AccountNumber); err != nil {
		return err
	} else if found {
		x, err := s.repo.WithdrawalByReference(ctx, reference)
		if err != nil {
			return err
		}
		if x.Status == "pending" {
			return s.submitWithdrawal(ctx, x, 0)
		}
		return nil
	}
	if _, err = s.repo.FeeQuote(ctx, user.ID, key, "withdrawal", false); err != nil {
		return err
	}
	profile, err := s.profiles.GetPaymentProfile(user.ID, s.currency)
	if err != nil {
		return err
	}
	if profile == nil || profile.BankCode != q.BankCode || profile.AccountNumber != q.AccountNumber {
		return errors.New("your bank account changed; review a new quote")
	}
	resolved, err := s.provider.ResolveAccount(ctx, q.AccountNumber, q.BankCode)
	if err != nil {
		return err
	}
	if !sameAccountOwner(user.FullName, resolved.AccountName) || !sameAccountOwner(q.AccountName, resolved.AccountName) {
		return errors.New("the payout account no longer matches your registered name")
	}
	policy, err := s.repo.FeePolicy(ctx)
	if err != nil {
		return err
	}
	if policy.ID != q.PolicyID {
		return errors.New("the fee policy changed; review a new quote")
	}
	fee, err := s.provider.WithdrawalFee(ctx, q.Amount(), q.Currency)
	if err != nil {
		return err
	}
	tax, err := money.BasisPoints(fee.FeeKobo, policy.ProviderQuoteTaxBPS)
	if err != nil {
		return err
	}
	totalFee, err := money.Add(fee.FeeKobo, tax)
	if err != nil {
		return err
	}
	if totalFee != q.ProviderKobo {
		return errors.New("provider charges changed; review a new quote before withdrawing")
	}
	if s.otp == nil {
		return errors.New("withdrawal code verification is unavailable")
	}
	if err = s.otp.Verify(user.ID, "withdrawal", code); err != nil {
		return err
	}
	account := &models.UserBankAccount{UserID: user.ID, BankName: q.BankName, BankCode: q.BankCode, AccountName: q.AccountName, AccountNumber: q.AccountNumber}
	if err = s.repo.RequestWithdrawal(ctx, user.ID, q.Amount(), reference, account, q); err != nil {
		return err
	}
	x, err := s.repo.WithdrawalByReference(ctx, reference)
	if err != nil {
		return err
	}
	return s.submitWithdrawal(ctx, x, 0)
}

func (s *WalletService) submitWithdrawal(ctx context.Context, x *models.Withdrawal, actorID int) error {
	if err := s.repo.ClaimWithdrawal(ctx, x.ID, actorID); err != nil {
		return err
	}
	providerID, err := s.provider.CreateWalletTransfer(ctx, x.Reference, x.Amount, x.Currency, x.BankCode, x.AccountNumber, x.AccountName)
	if err != nil {
		if errors.Is(err, ErrProviderRefused) {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if markErr := s.repo.MarkWithdrawalRefused(cleanup, x.ID); markErr != nil {
				return markErr
			}
			if refundErr := s.repo.FinishWithdrawal(cleanup, x.ID, 0, false, "failed", true); refundErr != nil {
				return refundErr
			}
			return errors.New("the provider rejected the payout; the amount and all wallet fees were returned")
		}
		return errors.New("the payout is awaiting provider verification; do not start another withdrawal")
	}
	if err = s.repo.SaveTransferID(ctx, x.ID, providerID); err != nil {
		return err
	}
	x.TransferID = sql.NullString{String: providerID, Valid: true}
	return s.checkWithdrawal(ctx, x)
}

func (s *WalletService) DecideWithdrawal(ctx context.Context, user *models.User, id int64, decision string) error {
	if user == nil || id <= 0 {
		return errors.New("choose a withdrawal")
	}
	if decision != "approve" && decision != "check" {
		return s.repo.FinishWithdrawal(ctx, id, user.ID, user.CanAdmin("finance"), decision, false)
	}
	if !user.CanAdmin("finance") {
		return errors.New("only an administrator can approve withdrawals")
	}
	if !s.ProviderAvailable() {
		return ErrPaymentProvider
	}
	x, err := s.repo.Withdrawal(ctx, id)
	if err != nil {
		return err
	}
	if decision == "check" {
		return s.checkWithdrawal(ctx, x)
	}
	resolved, err := s.provider.ResolveAccount(ctx, x.AccountNumber, x.BankCode)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.Join(strings.Fields(resolved.AccountName), " "), strings.Join(strings.Fields(x.AccountName), " ")) {
		return errors.New("the bank account name no longer matches this withdrawal")
	}
	return s.submitWithdrawal(ctx, x, user.ID)
}

func (s *WalletService) checkWithdrawal(ctx context.Context, x *models.Withdrawal) error {
	if !x.TransferID.Valid || x.TransferID.String == "" {
		return errors.New("the provider has not returned a transfer number; check this reference in the provider account before taking further action")
	}
	v, err := s.provider.VerifyWalletTransfer(ctx, x.TransferID.String)
	if err != nil {
		return err
	}
	amount, err := normalizedMoney(v.Amount.String())
	if err != nil {
		return err
	}
	if v.Reference != x.Reference || v.Currency != x.Currency || amount != x.Amount || (v.AccountNumber != "" && v.AccountNumber != x.AccountNumber) || (v.BankCode != "" && v.BankCode != x.BankCode) {
		return errors.New("the provider payout does not match this withdrawal")
	}
	if err = s.repo.SaveTransferID(ctx, x.ID, v.ID.String()); err != nil {
		return err
	}
	var actual *int64
	if v.Fee != "" {
		fee, err := money.Parse(v.Fee.String())
		if err != nil {
			return err
		}
		actual = &fee
	}
	switch strings.ToLower(v.Status) {
	case "successful", "success":
		return s.repo.FinishWithdrawal(ctx, x.ID, 0, false, "paid", true, actual)
	case "reversed":
		return s.repo.FinishWithdrawal(ctx, x.ID, 0, false, "reversed", true, actual)
	case "failed":
		return s.repo.FinishWithdrawal(ctx, x.ID, 0, false, "failed", true, actual)
	default:
		return nil
	}
}

func (s *WalletService) TransferWebhook(ctx context.Context, providerID, reference string) error {
	x, err := s.repo.WithdrawalByReference(ctx, reference)
	if err != nil {
		return err
	}
	if x.TransferID.Valid && x.TransferID.String != providerID {
		return errors.New("payout number did not match")
	}
	x.TransferID.Valid = true
	x.TransferID.String = providerID
	return s.checkWithdrawal(ctx, x)
}

type WalletAdminPage struct {
	User          *models.User
	Withdrawals   []models.Withdrawal
	Transfers     []models.Transfer
	Payments      []models.Payment
	Error, Notice string
}

func (s *WalletService) AdminPage(ctx context.Context, user *models.User) (*WalletAdminPage, error) {
	if user == nil || !user.CanAdmin("finance") {
		return nil, errors.New("administrator access is required")
	}
	p := &WalletAdminPage{User: user}
	var err error
	if p.Withdrawals, err = s.repo.Withdrawals(ctx, 0); err != nil {
		return nil, err
	}
	if p.Transfers, err = s.repo.Transfers(ctx, 0); err != nil {
		return nil, err
	}
	if p.Payments, err = s.repo.PaymentExceptions(ctx); err != nil {
		return nil, err
	}
	return p, nil
}
