// finance-preview renders synthetic financial pages for browser checks. It has
// no database connection, payment credentials, or payment execution capability.
package main

import (
	"database/sql"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/render"
	"backend/internal/services"
)

func main() {
	out := os.Getenv("FINANCE_PREVIEW_DIR")
	if out == "" {
		out = "/tmp/agrilinks-finance-preview"
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		log.Fatal(err)
	}
	save := func(name, template string, data any) {
		response := httptest.NewRecorder()
		if err := render.RenderPage(response, template, data); err != nil {
			log.Fatal(err)
		}
		html := response.Body.String()
		if err := os.WriteFile(filepath.Join(out, name+".html"), []byte(html), 0600); err != nil {
			log.Fatal(err)
		}
	}
	now := time.Now()
	user := &models.User{ID: 1, FullName: "Ada Okafor", StaffRole: "super_admin"}
	key := strings.Repeat("a", 32)
	wallet := services.WalletPage{User: user, Wallet: &models.Wallet{Balance: "125000.50", Currency: "NGN", WalletNumber: "AS1234567890"}, Profile: &models.User{BankName: "Test Bank", BankCode: "001", AccountNumber: "0123456789", AccountName: "Ada Okafor"}, HasTransactionPIN: true, ProviderAvailable: true, FundingProfileReady: true, Revision: "preview", VirtualAccount: &models.WalletVirtualAccount{Status: "active", AccountNumber: "1234567890", BankName: "Test Funding Bank", AccountName: "Ada Okafor"}}
	wallet.Quote = &models.FeeQuote{ID: key, Kind: "withdrawal", AmountKobo: 1000000, ServiceKobo: 30000, CustomerProviderKobo: 2500, WalletDebitKobo: 1032500, BankCreditKobo: 1000000, BankName: "Test Bank", AccountName: "Ada Okafor", AccountNumber: "0123456789", ExpiresAt: now.Add(10 * time.Minute)}
	save("withdrawal", "wallet.html", wallet)
	wallet.Quote = &models.FeeQuote{ID: key, Kind: "deposit", AmountKobo: 1000000, ServiceKobo: 5000, CustomerProviderKobo: 14000, WalletCreditKobo: 1000000, CheckoutTotalKobo: 1019000, ExpiresAt: now.Add(10 * time.Minute)}
	save("funding", "wallet.html", wallet)
	wallet.Quote = nil
	wallet.HasTransactionPIN = false
	save("enrollment", "wallet.html", wallet)
	finance := models.FinancePage{User: user, CanRequest: true, CanApprove: true, CSRF: key, ReportFrom: "2026-11-01", ReportTo: "2026-11-02", CompanyAccount: &models.CompanyAccount{CompanyName: "AgriLinks Limited", BankName: "Test Business Bank", AccountName: "AgriLinks Limited", AccountNumber: "0123456789"}, Metrics: []models.FinanceMetric{{Label: "Gross revenue", Value: "₦30000.00", Detail: "Posted fee revenue"}, {Label: "Available company funds", Value: "₦24000.00", Detail: "Settled cash after obligations and reserves"}, {Label: "Customer wallet liabilities", Value: "₦250000.00", Detail: "Protected customer funds"}}, Accounts: []models.FinanceAccount{{Code: "provider_settled_cash", Name: "Reconciled provider cash", Category: "asset", DebitKobo: 28000000, BalanceKobo: 28000000}, {Code: "wallet:1", Name: "Customer wallet 1", Category: "liability", CreditKobo: 25000000, BalanceKobo: -25000000}}, Withdrawals: []models.CompanyWithdrawal{{ID: 1, Reference: "COMPANY-" + key, AmountKobo: 1000000, ProviderKobo: 2500, TotalKobo: 1002500, Status: "pending", InitiatedBy: 2, Reason: "Transfer settled company service fees"}}, Quote: &models.CompanyQuote{ID: key, AmountKobo: 1000000, ProviderKobo: 2500, TotalKobo: 1002500, ExpiresAt: now.Add(10 * time.Minute)}}
	save("revenue", "platform_revenue.html", finance)
	escrow := models.EscrowPage{User: user, CanReview: true, CSRF: key, Escrow: &models.Escrow{OrderID: 1001, BuyerName: "Ada Okafor", SellerName: "Benue Farm Cooperative", GrossKobo: 1000000, CommissionKobo: 25000, Currency: "NGN", State: "held", PaymentReference: "ORDER-1001", Products: "Premium maize", DeliveredAt: sql.NullTime{Time: now, Valid: true}, BuyerConfirmedAt: sql.NullTime{Time: now, Valid: true}}, Items: []models.EscrowItem{{Product: "Premium maize", Quantity: "100", UnitPrice: "100.00", Total: "10000.00"}}}
	save("escrow", "purchase_protection.html", escrow)
	save("admin-security", "admin-security.html", map[string]any{"Enabled": false, "Verified": false, "Available": true, "CSRF": key, "Destination": "/admin/finance"})
	save("admin-security-enabled", "admin-security.html", map[string]any{"Enabled": true, "Verified": false, "Available": true, "CSRF": key, "Destination": "/admin/finance"})
	save("withdrawal-review", "withdrawal-review.html", models.WithdrawalReviewPage{
		Withdrawal:          &models.Withdrawal{ID: 10, UserID: 2, FullName: "Preview Member", Amount: "10000.00", Currency: "NGN", Status: "pending", BankName: "Test Bank", AccountNumber: "0123456789", AccountName: "Preview Member"},
		Evidence:            &models.WithdrawalEvidence{WithdrawalID: 10, UserID: 2, PhotoID: 101, Latitude: 7.7, Longitude: 8.5, AccuracyM: 12, CapturedAt: now, ReceivedAt: now},
		RegistrationPhotoID: 102, CSRF: key,
	})
	// A visibly synthetic placeholder tests private image loading without any
	// real identity photograph or private verification storage.
	photo := image.NewRGBA(image.Rect(0, 0, 240, 240))
	draw.Draw(photo, photo.Bounds(), image.NewUniform(color.RGBA{R: 220, G: 240, B: 230, A: 255}), image.Point{}, draw.Src)
	draw.Draw(photo, image.Rect(70, 40, 170, 140), image.NewUniform(color.RGBA{R: 40, G: 100, B: 80, A: 255}), image.Point{}, draw.Src)
	f, err := os.Create(filepath.Join(out, "private-photo.jpg"))
	if err != nil {
		log.Fatal(err)
	}
	if err := jpeg.Encode(f, photo, nil); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	log.Print("Synthetic finance previews written to ", out)
}
