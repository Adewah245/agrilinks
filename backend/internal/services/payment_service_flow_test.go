package services

import (
	"context"
	"errors"
	"testing"

	"backend/internal/models"
)

type fakePaymentStore struct {
	payment          *models.Payment
	createdReference string
	createdCurrency  string
	createdProvider  string
	updatedStatus    string
	finalized        bool
	finalStatus      string
	finalTransaction string
	finalWebhook     *models.WebhookEvent
	paidNow          bool
	webhookProcessed bool
}

func (f *fakePaymentStore) CreatePendingForOrder(
	buyerID,
	orderID int,
	reference,
	currency,
	provider string,
) (*models.Payment, error) {
	f.createdReference = reference
	f.createdCurrency = currency
	f.createdProvider = provider
	return &models.Payment{
		ID:        1,
		OrderID:   int64(orderID),
		BuyerID:   buyerID,
		Reference: reference,
		Amount:    "2500.50",
		Currency:  currency,
		Status:    models.PaymentPending,
		Provider:  provider,
	}, nil
}

func (f *fakePaymentStore) UpdateStatus(reference, status, providerTransactionID string) error {
	f.updatedStatus = status
	return nil
}

func (f *fakePaymentStore) GetByReference(reference string) (*models.Payment, error) {
	return f.payment, nil
}

func (f *fakePaymentStore) FinalizePayment(
	reference,
	status,
	providerTransactionID string,
	webhook *models.WebhookEvent,
) (bool, error) {
	f.finalized = true
	f.finalStatus = status
	f.finalTransaction = providerTransactionID
	f.finalWebhook = webhook
	paidNow := f.paidNow
	if !paidNow && status == models.PaymentSuccessful {
		paidNow = true
	}
	return paidNow, nil
}

func (f *fakePaymentStore) HasWebhookBeenProcessed(transactionID, event string) (bool, error) {
	return f.webhookProcessed, nil
}

func (f *fakePaymentStore) ListByBuyer(buyerID int) ([]models.Payment, error) {
	return nil, nil
}

func (f *fakePaymentStore) GetWallet(ownerType string, ownerID int, currency string) (*models.Wallet, error) {
	return nil, nil
}

func (f *fakePaymentStore) GetPaymentProfile(userID int, currency string) (*models.User, error) {
	return nil, nil
}

func (f *fakePaymentStore) UpsertBankAccount(account *models.UserBankAccount) error {
	return nil
}

type fakePaymentOrders struct {
	order *models.Order
}

func (f *fakePaymentOrders) GetForBuyer(orderID, buyerID int) (*models.Order, error) {
	return f.order, nil
}

func (f *fakePaymentOrders) ListByBuyer(buyerID int) ([]models.Order, error) {
	if f.order == nil {
		return nil, nil
	}
	return []models.Order{*f.order}, nil
}

type fakePaymentUsers struct {
	user *models.User
}

func (f *fakePaymentUsers) GetByID(id int) (*models.User, error) {
	return f.user, nil
}

type fakePaymentGateway struct {
	initializeRequest InitializePaymentRequest
	initializeURL     string
	verifyResult      *VerifiedTransaction
	verifyError       error
	verifyCalls       int
}

func (f *fakePaymentGateway) InitializePayment(_ context.Context, req InitializePaymentRequest) (string, error) {
	f.initializeRequest = req
	return f.initializeURL, nil
}

func (f *fakePaymentGateway) VerifyTransaction(_ context.Context, transactionID string) (*VerifiedTransaction, error) {
	f.verifyCalls++
	return f.verifyResult, f.verifyError
}

type fakePaymentNotifier struct {
	created int
}

func (f *fakePaymentNotifier) CreateNotification(userID int, title, message, notificationType string) error {
	f.created++
	return nil
}

func newPaymentServiceForTest(
	store *fakePaymentStore,
	orders *fakePaymentOrders,
	users *fakePaymentUsers,
	gateway *fakePaymentGateway,
	notifier *fakePaymentNotifier,
) *PaymentService {
	if notifier == nil {
		notifier = &fakePaymentNotifier{}
	}
	return NewPaymentService(store, orders, users, gateway, notifier, "webhook-hash", "NGN")
}

func TestInitiatePaymentUsesConfiguredCurrency(t *testing.T) {
	store := &fakePaymentStore{}
	orders := &fakePaymentOrders{order: &models.Order{ID: 9, BuyerID: 4, Status: "pending", TotalAmount: 2500.5}}
	users := &fakePaymentUsers{user: &models.User{ID: 4, Email: "buyer@example.test", FullName: "Buyer"}}
	gateway := &fakePaymentGateway{initializeURL: "https://checkout.flutterwave.test/session"}
	service := newPaymentServiceForTest(store, orders, users, gateway, &fakePaymentNotifier{})

	result, err := service.Initiate(context.Background(), 4, 9)
	if err != nil {
		t.Fatalf("Initiate() error = %v", err)
	}
	if result.CheckoutURL != gateway.initializeURL {
		t.Fatalf("checkout URL = %q", result.CheckoutURL)
	}
	if store.createdCurrency != "NGN" || store.createdProvider != models.PaymentProviderFlutterwave {
		t.Fatalf("created payment with currency=%q provider=%q", store.createdCurrency, store.createdProvider)
	}
	if gateway.initializeRequest.Currency != "NGN" || gateway.initializeRequest.Amount != "2500.50" {
		t.Fatalf("unexpected gateway request: %#v", gateway.initializeRequest)
	}
	if gateway.initializeRequest.Reference != store.createdReference {
		t.Fatalf("gateway reference = %q, stored reference = %q", gateway.initializeRequest.Reference, store.createdReference)
	}
}

func TestInitiateRejectsOrderThatIsNotPending(t *testing.T) {
	service := newPaymentServiceForTest(
		&fakePaymentStore{},
		&fakePaymentOrders{order: &models.Order{ID: 9, BuyerID: 4, Status: "confirmed", TotalAmount: 2500.5}},
		&fakePaymentUsers{user: &models.User{ID: 4, Email: "buyer@example.test"}},
		&fakePaymentGateway{},
		&fakePaymentNotifier{},
	)

	if _, err := service.Initiate(context.Background(), 4, 9); !errors.Is(err, ErrOrderNotAwaitingPayment) {
		t.Fatalf("error = %v, want ErrOrderNotAwaitingPayment", err)
	}
}

func TestConfirmCallbackFinalizesVerifiedPayment(t *testing.T) {
	store := &fakePaymentStore{payment: &models.Payment{
		OrderID: 9, BuyerID: 4, Reference: "ORDER-9-ref", Amount: "2500.50", Currency: "NGN", Status: models.PaymentPending,
	}}
	gateway := &fakePaymentGateway{verifyResult: &VerifiedTransaction{
		TransactionID: "123", Reference: "ORDER-9-ref", Status: "successful", Amount: 2500.5, Currency: "NGN",
	}}
	notifier := &fakePaymentNotifier{}
	service := newPaymentServiceForTest(store, &fakePaymentOrders{}, &fakePaymentUsers{}, gateway, notifier)

	status, err := service.ConfirmCallback(context.Background(), "123", "ORDER-9-ref")
	if err != nil {
		t.Fatalf("ConfirmCallback() error = %v", err)
	}
	if status != models.PaymentSuccessful || store.finalStatus != models.PaymentSuccessful {
		t.Fatalf("status = %q, finalized = %q", status, store.finalStatus)
	}
	if store.finalTransaction != "123" || store.finalWebhook != nil {
		t.Fatalf("unexpected finalization: txn=%q webhook=%#v", store.finalTransaction, store.finalWebhook)
	}
	if notifier.created != 1 {
		t.Fatalf("notifications = %d, want 1", notifier.created)
	}
}

func TestConfirmCallbackRejectsAmountMismatch(t *testing.T) {
	store := &fakePaymentStore{payment: &models.Payment{
		OrderID: 9, BuyerID: 4, Reference: "ORDER-9-ref", Amount: "2500.50", Currency: "NGN", Status: models.PaymentPending,
	}}
	gateway := &fakePaymentGateway{verifyResult: &VerifiedTransaction{
		TransactionID: "123", Reference: "ORDER-9-ref", Status: "successful", Amount: 1, Currency: "NGN",
	}}
	service := newPaymentServiceForTest(store, &fakePaymentOrders{}, &fakePaymentUsers{}, gateway, &fakePaymentNotifier{})

	if _, err := service.ConfirmCallback(context.Background(), "123", "ORDER-9-ref"); !errors.Is(err, ErrReferenceMismatch) {
		t.Fatalf("error = %v, want ErrReferenceMismatch", err)
	}
	if store.finalized {
		t.Fatal("mismatched payment was finalized")
	}
}

func TestProcessWebhookRejectsInvalidHashBeforeVerification(t *testing.T) {
	gateway := &fakePaymentGateway{}
	service := newPaymentServiceForTest(&fakePaymentStore{}, &fakePaymentOrders{}, &fakePaymentUsers{}, gateway, nil)

	err := service.ProcessWebhook(context.Background(), []byte(`{"event":"charge.completed","data":{"id":123,"tx_ref":"ref"}}`), "wrong")
	if !errors.Is(err, ErrUnauthorizedWebhook) {
		t.Fatalf("error = %v, want ErrUnauthorizedWebhook", err)
	}
	if gateway.verifyCalls != 0 {
		t.Fatalf("provider verification called %d times", gateway.verifyCalls)
	}
}

func TestProcessWebhookFinalizesAndRecordsEvent(t *testing.T) {
	body := []byte(`{"event":"charge.completed","data":{"id":123,"tx_ref":"ORDER-9-ref"}}`)
	store := &fakePaymentStore{payment: &models.Payment{
		OrderID: 9, BuyerID: 4, Reference: "ORDER-9-ref", Amount: "2500.50", Currency: "NGN", Status: models.PaymentPending,
	}}
	gateway := &fakePaymentGateway{verifyResult: &VerifiedTransaction{
		TransactionID: "123", Reference: "ORDER-9-ref", Status: "successful", Amount: 2500.5, Currency: "NGN",
	}}
	service := newPaymentServiceForTest(store, &fakePaymentOrders{}, &fakePaymentUsers{}, gateway, &fakePaymentNotifier{})

	if err := service.ProcessWebhook(context.Background(), body, "webhook-hash"); err != nil {
		t.Fatalf("ProcessWebhook() error = %v", err)
	}
	if !store.finalized || store.finalStatus != models.PaymentSuccessful || store.finalTransaction != "123" {
		t.Fatalf("unexpected finalization: %#v", store)
	}
	if store.finalWebhook == nil ||
		store.finalWebhook.Provider != models.PaymentProviderFlutterwave ||
		store.finalWebhook.TransactionID != "123" ||
		store.finalWebhook.Event != "charge.completed" ||
		string(store.finalWebhook.Payload) != string(body) {
		t.Fatalf("unexpected webhook record: %#v", store.finalWebhook)
	}
}

func TestProcessWebhookSkipsAlreadyProcessedEvent(t *testing.T) {
	gateway := &fakePaymentGateway{}
	store := &fakePaymentStore{webhookProcessed: true}
	service := newPaymentServiceForTest(store, &fakePaymentOrders{}, &fakePaymentUsers{}, gateway, nil)

	err := service.ProcessWebhook(context.Background(), []byte(`{"event":"charge.completed","data":{"id":123,"tx_ref":"ORDER-9-ref"}}`), "webhook-hash")
	if err != nil {
		t.Fatalf("ProcessWebhook() error = %v", err)
	}
	if gateway.verifyCalls != 0 || store.finalized {
		t.Fatalf("processed event was handled again: verifyCalls=%d finalized=%v", gateway.verifyCalls, store.finalized)
	}
}
