package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"backend/internal/models"
)

// PaymentService isolates the payment workflow from HTTP handlers and the
// Flutterwave API. The interfaces below are persistence boundaries rather
// than concrete repositories so the workflow can be tested without a database.
type paymentStore interface {
	CreatePendingForOrder(buyerID, orderID int, reference, currency, provider string) (*models.Payment, error)
	UpdateStatus(reference, status, providerTransactionID string) error
	SaveCheckoutURL(reference, checkoutURL string) error
	GetByReference(reference string) (*models.Payment, error)
	FinalizePayment(reference, status, providerTransactionID string, webhook *models.WebhookEvent) (bool, error)
	HasWebhookBeenProcessed(transactionID, event string) (bool, error)
	ListByBuyer(buyerID int) ([]models.Payment, error)
	GetWallet(ownerType string, ownerID int, currency string) (*models.Wallet, error)
	GetPaymentProfile(userID int, currency string) (*models.User, error)
	UpsertBankAccount(account *models.UserBankAccount) error
}

type paymentOrderStore interface {
	GetForBuyer(orderID, buyerID int) (*models.Order, error)
	ListByBuyer(buyerID int) ([]models.Order, error)
}

type paymentUserStore interface {
	GetByID(id int) (*models.User, error)
}

type paymentNotifier interface {
	CreateNotification(userID int, title, message, notificationType string) error
}

// PaymentService owns payment initiation, provider verification, and the
// mapping from provider state to Agro-Shield transaction and order state.
type PaymentService struct {
	repo       paymentStore
	orders     paymentOrderStore
	users      paymentUserStore
	gateway    PaymentGateway
	notifier   paymentNotifier
	secretHash string
	currency   string
	wallet     *WalletService
}

func NewPaymentService(
	repo paymentStore,
	orders paymentOrderStore,
	users paymentUserStore,
	gateway PaymentGateway,
	notifier paymentNotifier,
	secretHash string,
	currency string,
) *PaymentService {
	return &PaymentService{
		repo:       repo,
		orders:     orders,
		users:      users,
		gateway:    gateway,
		notifier:   notifier,
		secretHash: strings.TrimSpace(secretHash),
		currency:   strings.ToUpper(strings.TrimSpace(currency)),
	}
}

func (s *PaymentService) SetWallet(wallet *WalletService) { s.wallet = wallet }

var (
	ErrUnauthorizedWebhook     = errors.New("unauthorized webhook")
	ErrInvalidWebhook          = errors.New("invalid webhook payload")
	ErrReferenceMismatch       = errors.New("payment reference mismatch")
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderNotAwaitingPayment = errors.New("order is not awaiting payment")
	ErrPaymentEmailRequired    = errors.New("an email address is required for payment")
	ErrPaymentProvider         = errors.New("payment provider is unavailable")
)

type InitiatePaymentResult struct {
	CheckoutURL string
	Reference   string
}

// Initiate creates a pending transaction and asks Flutterwave for checkout.
func (s *PaymentService) Initiate(
	ctx context.Context,
	buyerID,
	orderID int,
) (*InitiatePaymentResult, error) {
	if s == nil || s.repo == nil || s.orders == nil || s.users == nil {
		return nil, errors.New("payment service is not configured")
	}
	if buyerID <= 0 || orderID <= 0 {
		return nil, errors.New("invalid payment request")
	}
	if s.gateway == nil {
		return nil, ErrPaymentProvider
	}
	if s.currency == "" {
		return nil, errors.New("payment currency is not configured")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	order, err := s.orders.GetForBuyer(orderID, buyerID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.Status != "pending" {
		return nil, ErrOrderNotAwaitingPayment
	}
	if order.TotalAmount <= 0 {
		return nil, errors.New("order total must be greater than zero")
	}

	buyer, err := s.users.GetByID(buyerID)
	if err != nil {
		return nil, err
	}
	if buyer == nil || strings.TrimSpace(buyer.Email) == "" {
		return nil, ErrPaymentEmailRequired
	}

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("create payment reference: %w", err)
	}
	reference := "ORDER-" + strconv.Itoa(orderID) + "-" + hex.EncodeToString(random)

	payment, err := s.repo.CreatePendingForOrder(
		buyerID,
		orderID,
		reference,
		s.currency,
		models.PaymentProviderFlutterwave,
	)
	if err != nil {
		return nil, err
	}

	if payment.CheckoutURL != "" {
		return &InitiatePaymentResult{CheckoutURL: payment.CheckoutURL, Reference: payment.Reference}, nil
	}
	checkoutURL, err := s.gateway.InitializePayment(ctx, InitializePaymentRequest{
		Reference:     payment.Reference,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		CustomerEmail: buyer.Email,
		CustomerName:  buyer.FullName,
		Description:   fmt.Sprintf("Payment for order %d", orderID),
	})
	if err != nil {
		// A timeout does not prove the provider rejected the payment. Keep
		// this reference for retries until a verified result settles it.
		return nil, fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}

	if err := s.repo.SaveCheckoutURL(payment.Reference, checkoutURL); err != nil {
		return nil, err
	}
	return &InitiatePaymentResult{
		CheckoutURL: checkoutURL,
		Reference:   payment.Reference,
	}, nil
}

// ConfirmCallback verifies the browser redirect with Flutterwave. The webhook
// remains authoritative, but verification here gives the buyer immediate
// feedback instead of trusting query-string values from the redirect.
func (s *PaymentService) ConfirmCallback(
	ctx context.Context,
	transactionID,
	reference string,
) (string, error) {
	if s == nil || s.repo == nil || s.gateway == nil {
		return "", ErrPaymentProvider
	}

	transactionID = strings.TrimSpace(transactionID)
	reference = strings.TrimSpace(reference)
	if transactionID == "" || reference == "" {
		return "", errors.New("transaction ID and reference are required")
	}

	verified, err := s.gateway.VerifyTransaction(ctx, transactionID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}
	if verified == nil || verified.Reference != reference || verified.TransactionID != transactionID {
		return "", ErrReferenceMismatch
	}

	return s.applyVerifiedPayment(verified, nil)
}

// ProcessWebhook authenticates, verifies, and durably processes one provider
// event. The provider transaction is verified against Flutterwave before any
// local payment or order state changes.
func (s *PaymentService) ProcessWebhook(
	ctx context.Context,
	body []byte,
	receivedHash string,
) error {
	if s == nil || s.repo == nil || s.gateway == nil {
		return errors.New("payment service is not configured")
	}
	if s.secretHash == "" ||
		subtle.ConstantTimeCompare([]byte(receivedHash), []byte(s.secretHash)) != 1 {
		return ErrUnauthorizedWebhook
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	var payload struct {
		Event string `json:"event"`
		Data  struct {
			ID        int64  `json:"id"`
			TxRef     string `json:"tx_ref"`
			Reference string `json:"reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrInvalidWebhook, err)
	}
	if payload.Event == "" || payload.Data.ID <= 0 {
		return fmt.Errorf("%w: event data is incomplete", ErrInvalidWebhook)
	}
	if payload.Event == "transfer.completed" && s.wallet != nil {
		return s.wallet.TransferWebhook(ctx, strconv.FormatInt(payload.Data.ID, 10), payload.Data.Reference)
	}
	if payload.Event != "charge.completed" {
		return nil
	}

	transactionID := strconv.FormatInt(payload.Data.ID, 10)
	processed, err := s.repo.HasWebhookBeenProcessed(transactionID, payload.Event)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	verified, err := s.gateway.VerifyTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}
	if verified == nil || verified.Reference != payload.Data.TxRef || verified.TransactionID != transactionID {
		return ErrReferenceMismatch
	}

	_, err = s.applyVerifiedPayment(verified, &models.WebhookEvent{
		Provider:      models.PaymentProviderFlutterwave,
		TransactionID: transactionID,
		Event:         payload.Event,
		Payload:       body,
	})
	return err
}

func (s *PaymentService) applyVerifiedPayment(
	verified *VerifiedTransaction,
	webhook *models.WebhookEvent,
) (string, error) {
	if verified == nil {
		return "", errors.New("verified payment is required")
	}

	payment, err := s.repo.GetByReference(verified.Reference)
	if err != nil {
		return "", err
	}
	if payment == nil {
		if s.wallet != nil {
			return s.wallet.ApplyVerifiedDeposit(context.Background(), verified, webhook)
		}
		return "", errors.New("payment not found")
	}

	status := verifiedPaymentStatus(verified.Status)
	if !strings.EqualFold(payment.Currency, verified.Currency) {
		return "", ErrReferenceMismatch
	}
	if status == models.PaymentSuccessful && !sameMoney(payment.Amount, verified.Amount) {
		return "", ErrReferenceMismatch
	}

	paidNow, err := s.repo.FinalizePayment(
		verified.Reference,
		status,
		verified.TransactionID,
		webhook,
	)
	if err != nil {
		return "", err
	}

	if paidNow && s.notifier != nil {
		_ = s.notifier.CreateNotification(
			payment.BuyerID,
			"Payment confirmed",
			fmt.Sprintf("Payment for order %d has been confirmed.", payment.OrderID),
			"payment",
		)
	}

	return status, nil
}

func verifiedPaymentStatus(providerStatus string) string {
	switch strings.ToLower(strings.TrimSpace(providerStatus)) {
	case "successful", "success":
		return models.PaymentSuccessful
	case "failed", "cancelled", "canceled":
		return models.PaymentFailed
	default:
		return models.PaymentPending
	}
}

func sameMoney(expected string, actual float64) bool {
	expectedAmount, err := strconv.ParseFloat(strings.TrimSpace(expected), 64)
	if err != nil || math.IsNaN(actual) || math.IsInf(actual, 0) || math.IsNaN(expectedAmount) || math.IsInf(expectedAmount, 0) {
		return false
	}
	return strconv.FormatFloat(expectedAmount, 'f', 2, 64) ==
		strconv.FormatFloat(actual, 'f', 2, 64)
}

// PayableOrders returns only orders that are still waiting for payment.
func (s *PaymentService) PayableOrders(buyerID int) ([]models.Order, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	orders, err := s.orders.ListByBuyer(buyerID)
	if err != nil {
		return nil, err
	}

	payable := make([]models.Order, 0, len(orders))
	positions := map[int]int{}
	for _, order := range orders {
		if order.Status != "pending" {
			continue
		}
		item := models.OrderItem{ProductID: order.ProductID, ProductName: order.ProductName, Quantity: order.Quantity, TotalPrice: order.TotalPrice}
		if index, ok := positions[order.ID]; ok {
			payable[index].Items = append(payable[index].Items, item)
			payable[index].ProductName += ", " + order.ProductName
		} else {
			positions[order.ID] = len(payable)
			order.Items = []models.OrderItem{item}
			payable = append(payable, order)
		}
	}
	return payable, nil
}

func (s *PaymentService) Currency() string {
	if s == nil {
		return ""
	}
	return s.currency
}

func (s *PaymentService) MyPayments(buyerID int) ([]models.Payment, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	return s.repo.ListByBuyer(buyerID)
}

func (s *PaymentService) MyWallet(ownerID int) (*models.Wallet, error) {
	if ownerID <= 0 {
		return nil, errors.New("invalid wallet owner")
	}
	if s.currency == "" {
		return nil, errors.New("wallet currency is not configured")
	}
	return s.repo.GetWallet("user", ownerID, s.currency)
}

// PaymentProfile loads the private bank-account and wallet-number details
// used by the transfer flow.
func (s *PaymentService) PaymentProfile(userID int) (*models.User, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if s.currency == "" {
		return nil, errors.New("wallet currency is not configured")
	}
	return s.repo.GetPaymentProfile(userID, s.currency)
}

// UpdateBankAccount validates and saves a user's payout account.
func (s *PaymentService) UpdateBankAccount(
	userID int,
	bankName string,
	bankCode string,
	accountName string,
	accountNumber string,
) error {
	if userID <= 0 {
		return errors.New("invalid user")
	}

	bankName = strings.TrimSpace(bankName)
	bankCode = strings.TrimSpace(bankCode)
	accountName = strings.TrimSpace(accountName)
	accountNumber = strings.TrimSpace(accountNumber)

	if bankName == "" || bankCode == "" || accountName == "" || accountNumber == "" {
		return errors.New("bank name, bank code, account name, and account number are required")
	}

	return s.repo.UpsertBankAccount(&models.UserBankAccount{
		UserID:        userID,
		BankName:      bankName,
		BankCode:      bankCode,
		AccountName:   accountName,
		AccountNumber: accountNumber,
	})
}
