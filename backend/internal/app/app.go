package app

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/services"
)

// New wires the same application used by the server and integration tests.
// Provider calls are enabled only when their real configuration is present.
func New(db *sql.DB, cfg config.Config) (*Container, error) {
	var hasCutover bool
	if err := db.QueryRow(`SELECT to_regclass('legacy_agro_shield.cutover_state') IS NOT NULL`).Scan(&hasCutover); err != nil {
		return nil, err
	}
	if hasCutover {
		var complete bool
		if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM legacy_agro_shield.cutover_state WHERE completed_at IS NOT NULL)`).Scan(&complete); err != nil {
			return nil, err
		}
		if !complete {
			return nil, fmt.Errorf("the old data transfer is unfinished; complete cmd/legacy-cutover before starting the app")
		}
	}

	var diagnosis services.AIProvider
	var err error
	if cfg.GeminiAPIKey != "" {
		diagnosis, err = services.NewGeminiProvider(cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			return nil, fmt.Errorf("configure diagnosis: %w", err)
		}
	}
	commission, err := services.ParseCommissionBasisPoints(cfg.CommissionPercent)
	if err != nil {
		return nil, err
	}
	var payment services.PaymentGateway
	var walletProvider *services.FlutterwaveClient
	if cfg.FlutterwaveSecretKey != "" {
		client := services.NewFlutterwaveClient(cfg.FlutterwaveSecretKey, cfg.PaymentRedirectURL, cfg.FlutterwavePaymentOptions)
		client.SetBaseURL(cfg.FlutterwaveBaseURL)
		payment = client
		walletProvider = client
	}
	prices := services.NewPriceService(repository.NewPostgresPriceRepository(db), repository.NewPostgresSubmitterRepository(db))
	container := NewContainer(
		repository.NewUserRepository(db), repository.NewLocationRepository(db), repository.NewProductRepository(db),
		repository.NewOrderRepository(db), repository.NewCartRepository(db), repository.NewDiagnosisRepository(db),
		repository.NewNegotiationRepository(db), repository.NewNegotiationMessageRepository(db),
		repository.NewNotificationRepository(db), repository.NewMarketEventRepository(db), repository.NewLearningRepository(db),
		repository.NewPaymentRepository(db, commission), repository.NewDeliveryRepository(db), repository.NewDeliveryPartnerRepository(db),
		repository.NewReviewRepository(db), repository.NewConversationRepository(db), repository.NewConversationMemberRepository(db),
		repository.NewChatMessageRepository(db), diagnosis, prices, cfg.Negotiation, payment, cfg.FlutterwaveSecretHash, cfg.PaymentCurrency,
	)
	var administratorIDs []int
	if strings.TrimSpace(cfg.WalletAdminUserIDs) != "" {
		for _, raw := range strings.Split(cfg.WalletAdminUserIDs, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("WALLET_ADMIN_USER_IDS must be a list of positive user IDs")
			}
			administratorIDs = append(administratorIDs, id)
		}
	}
	container.UserRepo.SetConfiguredAdministrators(administratorIDs)
	container.Config = cfg
	container.Sessions = services.NewSessionService(repository.NewSessionRepository(db), cfg.SessionTTL)
	container.Weather = services.NewWeatherService()
	var delivery services.OTPDelivery
	switch cfg.SMSProvider {
	case "termii":
		delivery, err = services.NewTermiiOTPDelivery(cfg.TermiiAPIKey, cfg.TermiiSenderID, cfg.TermiiBaseURL)
	case "brevo":
		delivery, err = services.NewBrevoOTPDelivery(cfg.BrevoSMTPHost, cfg.BrevoSMTPLogin, cfg.BrevoSMTPKey, cfg.BrevoFromEmail, cfg.BrevoFromName)
	}
	if err != nil {
		return nil, fmt.Errorf("configure password recovery: %w", err)
	}
	container.OTP = services.NewOTPService(repository.NewOTPRepository(db), delivery)
	container.OTP.Configure(cfg.OTPTTL, cfg.OTPMaxAttempts, cfg.OTPResendInterval)
	container.Auth.SetOTP(container.OTP)
	container.Wallet = services.NewWalletService(repository.NewWalletRepository(db, cfg.PaymentCurrency, commission), repository.NewPaymentRepository(db, commission), walletProvider, container.OTP, cfg.PaymentCurrency)
	container.Payment.SetWallet(container.Wallet)
	container.Learning.SetProvider(services.NewLearningProvider(cfg))
	return container, nil
}
