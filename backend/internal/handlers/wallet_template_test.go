package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

func TestWalletTemplateRendersAccountStates(t *testing.T) {
	for _, test := range []struct {
		name      string
		provider  bool
		account   *models.WalletVirtualAccount
		profile   *models.User
		want      string
		forbidden string
	}{
		{name: "payments unavailable", want: "Bank payments are not set up yet.", forbidden: `name="request_key"`},
		{name: "new account", provider: true, profile: &models.User{}, want: "Connect your bank account"},
		{name: "saved payout account", provider: true, profile: &models.User{BankName: "Payout Bank", AccountNumber: "0123456789"}, want: "Your saved account for withdrawals."},
		{name: "active funding account", provider: true, profile: &models.User{}, account: &models.WalletVirtualAccount{Status: models.VirtualAccountActive, BankName: "Funding Bank", AccountNumber: "0987654321"}, want: `id="wallet-funding-number">0987654321`},
		{name: "pending funding account", provider: true, profile: &models.User{}, account: &models.WalletVirtualAccount{Status: models.VirtualAccountPending, AccountNumber: "0987654321"}, want: "Connect your bank account", forbidden: "0987654321"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			page := services.WalletPage{
				User:              &models.User{ID: 1, FullName: "Ada Farmer"},
				Wallet:            &models.Wallet{Balance: "0.00", Currency: "NGN", WalletNumber: "AS1234567890"},
				Profile:           test.profile,
				VirtualAccount:    test.account,
				ProviderAvailable: test.provider,
			}
			if err := render.RenderTemplates(response, "wallet.html", page); err != nil {
				t.Fatalf("wallet account state should render: %v", err)
			}
			body := response.Body.String()
			if !strings.Contains(body, test.want) {
				t.Fatalf("missing account state %q", test.want)
			}
			if test.forbidden != "" && strings.Contains(body, test.forbidden) {
				t.Fatalf("unexpected account details or action %q", test.forbidden)
			}
		})
	}
}

func TestWalletTemplatePreservesMoneyActionForms(t *testing.T) {
	now := time.Now()
	response := httptest.NewRecorder()
	page := services.WalletPage{
		User:              &models.User{ID: 1, FullName: "Ada <script>alert(1)</script>"},
		Wallet:            &models.Wallet{Balance: "12500.50", Currency: "NGN", WalletNumber: "AS1234567890"},
		Profile:           &models.User{},
		ProviderAvailable: true,
		History:           []models.WalletTransaction{{Type: "deposit", Amount: "12500.50", Reference: "FUND-1", CreatedAt: now}},
		Deposits:          []models.WalletDeposit{{Amount: "12500.50", Currency: "NGN", Status: "successful", Reference: "FUND-1", CreatedAt: now}},
		Transfers: []models.Transfer{
			{ID: 101, SenderID: 1, RecipientID: 2, Status: "pending", Currency: "NGN", Amount: "100.00", CreatedAt: now},
			{ID: 102, SenderID: 2, RecipientID: 1, Status: "pending", Currency: "NGN", Amount: "100.00", CreatedAt: now},
			{ID: 103, SenderID: 1, RecipientID: 2, Status: "completed", Currency: "NGN", Amount: "100.00", CreatedAt: now},
		},
		Withdrawals: []models.Withdrawal{
			{ID: 201, Status: "pending", Currency: "NGN", Amount: "100.00", AccountNumber: "0123456789", CreatedAt: now},
			{ID: 202, Status: "paid", Currency: "NGN", Amount: "100.00", AccountNumber: "0123456789", CreatedAt: now},
		},
	}
	if err := render.RenderTemplates(response, "wallet.html", page); err != nil {
		t.Fatalf("populated wallet should render: %v", err)
	}
	body := response.Body.String()
	for _, action := range []string{"deposit", "save_bank", "send_code", "withdraw", "transfer", "cancel_transfer", "cancel_withdrawal"} {
		if !strings.Contains(body, `name="action" value="`+action+`"`) {
			t.Errorf("missing existing money action %s", action)
		}
	}
	for _, field := range []string{"request_key", "bank_code", "account_number", "recipient", "amount", "note", "code", "purpose"} {
		if !strings.Contains(body, `name="`+field+`"`) {
			t.Errorf("missing required wallet form field %s", field)
		}
	}
	for _, id := range []string{"101", "201"} {
		if !strings.Contains(body, `name="id" value="`+id+`"`) {
			t.Errorf("pending outgoing request %s should be cancellable", id)
		}
	}
	for _, id := range []string{"102", "103", "202"} {
		if strings.Contains(body, `name="id" value="`+id+`"`) {
			t.Errorf("incoming or completed request %s should not have a cancellation form", id)
		}
	}
	if strings.Contains(body, "<script>alert(1)</script>") || !strings.Contains(body, "&lt;script&gt;") {
		t.Error("wallet holder names must remain HTML escaped")
	}
}
