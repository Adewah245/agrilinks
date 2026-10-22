package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

type WalletService struct {
	repo     *repository.WalletRepository
	profiles *repository.PaymentRepository
	provider *FlutterwaveClient
	otp      *OTPService
	currency string
}

func NewWalletService(repo *repository.WalletRepository, profiles *repository.PaymentRepository, provider *FlutterwaveClient, otp *OTPService, currency string) *WalletService {
	return &WalletService{repo: repo, profiles: profiles, provider: provider, otp: otp, currency: currency}
}

func (s *WalletService) SetOTP(otp *OTPService)  { s.otp = otp }
func (s *WalletService) ProviderAvailable() bool { return s != nil && s.provider != nil }

type WalletPage struct {
	User              *models.User
	Wallet            *models.Wallet
	History           []models.WalletTransaction
	Deposits          []models.WalletDeposit
	Withdrawals       []models.Withdrawal
	Transfers         []models.Transfer
	VirtualAccount    *models.WalletVirtualAccount
	Profile           *models.User
	ProviderAvailable bool
	Error, Notice     string
}

func (s *WalletService) Page(ctx context.Context, user *models.User) (*WalletPage, error) {
	if user == nil || user.ID <= 0 {
		return nil, errors.New("please sign in")
	}
	p := &WalletPage{User: user, ProviderAvailable: s.ProviderAvailable()}
	var err error
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
	if p.Profile, err = s.profiles.GetPaymentProfile(user.ID, s.currency); err != nil {
		return nil, err
	}
	return p, nil
}

var decimalAmount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)
var accountNumber = regexp.MustCompile(`^[0-9]{10}$`)
var requestToken = regexp.MustCompile(`^[a-f0-9]{32}$`)

// Money is read as digits and counted in minor units, never float arithmetic.
func moneyMinorUnits(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if !decimalAmount.MatchString(value) {
		return 0, errors.New("enter an amount with no more than two decimal places")
	}
	parts := strings.Split(value, ".")
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	n, err := strconv.ParseInt(strings.TrimLeft(parts[0]+fraction, "0"), 10, 64)
	if err != nil || n <= 0 {
		return 0, errors.New("enter a positive amount within the supported range")
	}
	return n, nil
}

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
	d, err := s.repo.CreateDeposit(ctx, user.ID, reference, amount)
	if err != nil {
		return nil, err
	}
	if d.Status != "pending" {
		return nil, errors.New("this funding request has already been handled")
	}
	if d.CheckoutURL != "" {
		return &InitiatePaymentResult{Reference: reference, CheckoutURL: d.CheckoutURL}, nil
	}
	link, err := s.provider.InitializePayment(ctx, InitializePaymentRequest{Reference: reference, Amount: d.Amount, Currency: d.Currency, CustomerEmail: user.Email, CustomerName: user.FullName, Description: "Add money to your Agro-Shield wallet"})
	if err != nil {
		return nil, err
	}
	if err = s.repo.SaveDepositURL(ctx, reference, link); err != nil {
		return nil, err
	}
	return &InitiatePaymentResult{Reference: reference, CheckoutURL: link}, nil
}

func (s *WalletService) ApplyVerifiedDeposit(ctx context.Context, v *VerifiedTransaction, event *models.WebhookEvent) (string, error) {
	if v == nil || v.TransactionID == "" || math.IsNaN(v.Amount) || math.IsInf(v.Amount, 0) || v.Amount <= 0 {
		return "", errors.New("the provider returned an invalid funding amount")
	}
	status := verifiedPaymentStatus(v.Status)
	_, err := s.repo.FinalizeDeposit(ctx, v.Reference, v.TransactionID, strconv.FormatFloat(v.Amount, 'f', 2, 64), strings.ToUpper(v.Currency), status, event)
	return status, err
}

func (s *WalletService) PayOrder(ctx context.Context, userID, orderID int) error {
	if userID <= 0 || orderID <= 0 {
		return errors.New("choose an order")
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

func (s *WalletService) RequestTransfer(ctx context.Context, user *models.User, recipient, amount, note, code string) error {
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
	if err = s.otp.Verify(user.ID, "transfer", code); err != nil {
		return err
	}
	reference, err := newWalletReference("TRANSFER", user.ID)
	if err != nil {
		return err
	}
	return s.repo.RequestTransfer(ctx, user.ID, recipient, amount, reference, note)
}

func (s *WalletService) DecideTransfer(ctx context.Context, user *models.User, id int64, decision string) error {
	if user == nil || id <= 0 {
		return errors.New("choose a transfer")
	}
	return s.repo.DecideTransfer(ctx, user.ID, user.IsAdministrator(), id, decision)
}

func (s *WalletService) Banks(ctx context.Context) ([]Bank, error) {
	if !s.ProviderAvailable() {
		return nil, ErrPaymentProvider
	}
	return s.provider.ListBanks(ctx)
}

func (s *WalletService) SaveBank(ctx context.Context, userID int, bankCode, number string) error {
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

func (s *WalletService) RequestWithdrawal(ctx context.Context, user *models.User, amount, code string) error {
	if user == nil || s.otp == nil {
		return errors.New("please sign in")
	}
	if !s.ProviderAvailable() {
		return ErrPaymentProvider
	}
	if s.currency != "NGN" {
		return errors.New("bank withdrawals currently support Nigerian naira")
	}
	amount, err := normalizedMoney(amount)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(amount, ".00") {
		return errors.New("enter a whole naira amount for a bank withdrawal")
	}
	p, err := s.profiles.GetPaymentProfile(user.ID, s.currency)
	if err != nil {
		return err
	}
	if p == nil || p.AccountNumber == "" {
		return errors.New("save your bank account first")
	}
	resolved, err := s.provider.ResolveAccount(ctx, p.AccountNumber, p.BankCode)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.Join(strings.Fields(resolved.AccountName), " "), strings.Join(strings.Fields(p.AccountName), " ")) {
		return errors.New("your bank details have changed; save the account again")
	}
	if err = s.otp.Verify(user.ID, "withdrawal", code); err != nil {
		return err
	}
	reference, err := newWalletReference("WITHDRAW", user.ID)
	if err != nil {
		return err
	}
	return s.repo.RequestWithdrawal(ctx, user.ID, amount, reference, &models.UserBankAccount{UserID: user.ID, BankName: p.BankName, BankCode: p.BankCode, AccountName: resolved.AccountName, AccountNumber: p.AccountNumber})
}

func (s *WalletService) DecideWithdrawal(ctx context.Context, user *models.User, id int64, decision string) error {
	if user == nil || id <= 0 {
		return errors.New("choose a withdrawal")
	}
	if decision != "approve" && decision != "check" {
		return s.repo.FinishWithdrawal(ctx, id, user.ID, user.IsAdministrator(), decision, false)
	}
	if !user.IsAdministrator() {
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
	if err = s.repo.ClaimWithdrawal(ctx, id, user.ID); err != nil {
		return err
	}
	providerID, err := s.provider.CreateWalletTransfer(ctx, x.Reference, x.Amount, x.Currency, x.BankCode, x.AccountNumber, x.AccountName)
	if err != nil {
		return fmt.Errorf("the payout needs a provider check before it can be retried: %w", err)
	}
	if err = s.repo.SaveTransferID(ctx, id, providerID); err != nil {
		return err
	}
	x.TransferID.Valid = true
	x.TransferID.String = providerID
	return s.checkWithdrawal(ctx, x)
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
	if v.Reference != x.Reference || v.Currency != x.Currency || amount != x.Amount {
		return errors.New("the provider payout does not match this withdrawal")
	}
	if err = s.repo.SaveTransferID(ctx, x.ID, v.ID.String()); err != nil {
		return err
	}
	switch strings.ToLower(v.Status) {
	case "successful", "success":
		return s.repo.FinishWithdrawal(ctx, x.ID, 0, false, "paid", true)
	case "failed":
		return s.repo.FinishWithdrawal(ctx, x.ID, 0, false, "failed", true)
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
	if user == nil || !user.IsAdministrator() {
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
