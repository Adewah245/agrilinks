package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultNegotiationMaxRounds = 5
	defaultNegotiationWindow    = 24 * time.Hour
	defaultPaymentCurrency      = "NGN"
)

// Config holds all application settings in one place.
type Config struct {
	// Welcome email — Brevo SMTP
	BrevoSMTPHost  string
	BrevoSMTPLogin string
	BrevoSMTPKey   string
	BrevoFromEmail string
	BrevoFromName  string

	// Application
	Port                  string
	AppEnv                string
	FrontendDir           string
	SessionTTL            time.Duration
	RequestTimeout        time.Duration
	ShutdownTimeout       time.Duration
	MaxRequestBytes       int64
	AuthRequestsPerMinute int
	AIRequestsPerMinute   int
	Negotiation           NegotiationPolicy

	// TrustProxyHeaders must only be enabled when a trusted reverse proxy
	// overwrites X-Forwarded-For. TrustedProxyCIDRs are the networks from which
	// those headers are accepted; forwarded values are ignored for any other
	// peer, so a directly reachable client cannot choose its own rate-limit key.
	TrustProxyHeaders bool
	TrustedProxyCIDRs []*net.IPNet

	// Database
	DatabaseURL string

	// Diagnosis AI — Gemini only
	GeminiAPIKey string
	GeminiModel  string

	// Learning AI — Groq primary, OpenRouter backup
	GroqAPIKey        string
	OpenRouterAPIKey  string
	GroqModel         string
	OpenRouterModel   string
	GroqBaseURL       string
	OpenRouterBaseURL string
	AIRequestTimeout  time.Duration
	AIMaxTokens       int

	// Payments — Flutterwave
	FlutterwaveSecretKey  string
	FlutterwaveSecretHash string
	FlutterwaveBaseURL    string
	PaymentRedirectURL    string
	PaymentCurrency       string

	// FlutterwavePaymentOptions is the comma-separated list of funding methods
	// offered on the hosted checkout, e.g. "card,banktransfer". Sent explicitly
	// so which methods appear is decided by this file and not by whatever the
	// Flutterwave dashboard happens to be set to.
	FlutterwavePaymentOptions string

	// CommissionPercent is the platform's cut of marketplace sales, e.g. "2.5".
	// Empty or "0" means no commission, so leaving it unset changes nothing.
	CommissionPercent string

	// Wallet administration
	WalletAdminUserIDs string

	// WalletStaleDepositHours is how long a funding attempt may stay unsettled
	// before it is listed on the admin page for an operator to investigate. It is
	// a reporting threshold only — nothing is credited or expired automatically.
	WalletStaleDepositHours string

	// OTP delivery — SMS
	//
	// SMSProvider selects real OTP delivery: "termii" or "brevo".
	// "disabled" leaves password recovery unavailable until configured.
	SMSProvider       string
	TermiiAPIKey      string
	TermiiSenderID    string
	TermiiBaseURL     string
	OTPTTL            time.Duration
	OTPMaxAttempts    int
	OTPResendInterval time.Duration
	USSDToken         string
}

// NegotiationPolicy controls how long a negotiation remains active and
// how many offers may be exchanged.
type NegotiationPolicy struct {
	MaxRounds int
	Window    time.Duration
}

// FlutterwaveConfig holds the payment boundary settings.
type FlutterwaveConfig struct {
	SecretKey      string
	SecretHash     string
	RedirectURL    string
	Currency       string
	PaymentOptions string
}

// LoadDatabaseURL reads only the database URL.
// It lets database tools run without requiring AI API keys.
func LoadDatabaseURL() (string, error) {
	// Load .env when available.
	_ = godotenv.Load()

	// Read the database URL and remove extra spaces.
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))

	// Stop if the database URL is missing.
	if databaseURL == "" {
		return "", fmt.Errorf("DATABASE_URL must not be empty")
	}

	return databaseURL, nil
}

// LoadNegotiationPolicy reads the marketplace negotiation policy.
//
// Defaults match the product rules documented in README.md.
func LoadNegotiationPolicy() (NegotiationPolicy, error) {
	_ = godotenv.Load()

	policy := NegotiationPolicy{
		MaxRounds: defaultNegotiationMaxRounds,
		Window:    defaultNegotiationWindow,
	}

	if raw := strings.TrimSpace(os.Getenv("NEGOTIATION_MAX_ROUNDS")); raw != "" {
		maxRounds, err := strconv.Atoi(raw)
		if err != nil || maxRounds <= 0 {
			return NegotiationPolicy{}, fmt.Errorf(
				"NEGOTIATION_MAX_ROUNDS must be a positive integer",
			)
		}
		policy.MaxRounds = maxRounds
	}

	if raw := strings.TrimSpace(os.Getenv("NEGOTIATION_WINDOW")); raw != "" {
		window, err := time.ParseDuration(raw)
		if err != nil || window <= 0 {
			return NegotiationPolicy{}, fmt.Errorf(
				"NEGOTIATION_WINDOW must be a positive duration such as 24h",
			)
		}
		policy.Window = window
	}

	return policy, nil
}

// LoadFlutterwaveConfig reads the payment-provider settings required to
// initialize checkout and authenticate webhooks.
func LoadFlutterwaveConfig() (FlutterwaveConfig, error) {
	_ = godotenv.Load()

	cfg := FlutterwaveConfig{
		SecretKey:      strings.TrimSpace(os.Getenv("FLUTTERWAVE_SECRET_KEY")),
		SecretHash:     strings.TrimSpace(os.Getenv("FLUTTERWAVE_SECRET_HASH")),
		RedirectURL:    strings.TrimSpace(os.Getenv("PAYMENT_REDIRECT_URL")),
		Currency:       strings.ToUpper(strings.TrimSpace(os.Getenv("PAYMENT_CURRENCY"))),
		PaymentOptions: strings.TrimSpace(os.Getenv("FLUTTERWAVE_PAYMENT_OPTIONS")),
	}

	if cfg.SecretKey == "" {
		return FlutterwaveConfig{}, fmt.Errorf("FLUTTERWAVE_SECRET_KEY must not be empty")
	}

	if cfg.SecretHash == "" {
		return FlutterwaveConfig{}, fmt.Errorf("FLUTTERWAVE_SECRET_HASH must not be empty")
	}

	if cfg.PaymentOptions == "" {
		cfg.PaymentOptions = "card,banktransfer"
	}

	if cfg.Currency == "" {
		cfg.Currency = defaultPaymentCurrency
	}
	if len(cfg.Currency) != 3 {
		return FlutterwaveConfig{}, fmt.Errorf("PAYMENT_CURRENCY must be a three-letter currency code")
	}

	redirect, err := url.Parse(cfg.RedirectURL)
	if err != nil || (redirect.Scheme != "http" && redirect.Scheme != "https") || redirect.Host == "" {
		return FlutterwaveConfig{}, fmt.Errorf("PAYMENT_REDIRECT_URL must be an absolute http(s) URL")
	}

	return cfg, nil
}

// Load reads the application settings from environment variables.
func Load() (Config, error) {
	// Load .env when available.
	// Deployment environments can provide variables directly.
	_ = godotenv.Load()

	cfg := Config{
		// Welcome email — Brevo SMTP
		BrevoSMTPHost:  strings.TrimSpace(os.Getenv("BREVO_SMTP_HOST")),
		BrevoSMTPLogin: strings.TrimSpace(os.Getenv("BREVO_SMTP_LOGIN")),
		BrevoSMTPKey:   strings.TrimSpace(os.Getenv("BREVO_SMTP_KEY")),
		BrevoFromEmail: strings.TrimSpace(os.Getenv("BREVO_FROM_EMAIL")),
		BrevoFromName:  strings.TrimSpace(os.Getenv("BREVO_FROM_NAME")),

		// Application
		Port:        strings.TrimSpace(os.Getenv("PORT")),
		AppEnv:      strings.TrimSpace(os.Getenv("APP_ENV")),
		FrontendDir: strings.TrimSpace(os.Getenv("FRONTEND_DIR")),

		// Database
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),

		// AI providers
		GeminiAPIKey:      strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:       strings.TrimSpace(os.Getenv("GEMINI_MODEL")),
		GroqAPIKey:        strings.TrimSpace(os.Getenv("GROQ_API_KEY")),
		OpenRouterAPIKey:  strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")),
		GroqModel:         strings.TrimSpace(os.Getenv("GROQ_MODEL")),
		OpenRouterModel:   strings.TrimSpace(os.Getenv("OPENROUTER_MODEL")),
		GroqBaseURL:       envString("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		OpenRouterBaseURL: envString("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),

		// Payments
		FlutterwaveSecretKey:  strings.TrimSpace(os.Getenv("FLUTTERWAVE_SECRET_KEY")),
		FlutterwaveSecretHash: strings.TrimSpace(os.Getenv("FLUTTERWAVE_SECRET_HASH")),
		FlutterwaveBaseURL:    envString("FLUTTERWAVE_BASE_URL", "https://api.flutterwave.com/v3"),
		PaymentRedirectURL:    strings.TrimSpace(os.Getenv("PAYMENT_REDIRECT_URL")),
		PaymentCurrency:       strings.ToUpper(strings.TrimSpace(os.Getenv("PAYMENT_CURRENCY"))),

		FlutterwavePaymentOptions: strings.TrimSpace(os.Getenv("FLUTTERWAVE_PAYMENT_OPTIONS")),

		// Wallet administration
		WalletAdminUserIDs:      strings.TrimSpace(os.Getenv("WALLET_ADMIN_USER_IDS")),
		WalletStaleDepositHours: strings.TrimSpace(os.Getenv("WALLET_STALE_DEPOSIT_HOURS")),
		CommissionPercent:       strings.TrimSpace(os.Getenv("COMMISSION_PERCENT")),

		// SMS / OTP
		SMSProvider:    strings.TrimSpace(os.Getenv("SMS_PROVIDER")),
		TermiiAPIKey:   strings.TrimSpace(os.Getenv("TERMII_API_KEY")),
		TermiiSenderID: strings.TrimSpace(os.Getenv("TERMII_SENDER_ID")),
		TermiiBaseURL:  strings.TrimSpace(os.Getenv("TERMII_BASE_URL")),
		USSDToken:      strings.TrimSpace(os.Getenv("USSD_TOKEN")),
	}
	var err error
	for _, field := range []struct {
		name     string
		target   *time.Duration
		fallback time.Duration
	}{
		{"SESSION_TTL", &cfg.SessionTTL, 24 * time.Hour},
		{"HTTP_REQUEST_TIMEOUT", &cfg.RequestTimeout, 60 * time.Second},
		{"HTTP_SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout, 15 * time.Second},
		{"AI_REQUEST_TIMEOUT", &cfg.AIRequestTimeout, 30 * time.Second},
		{"OTP_TTL", &cfg.OTPTTL, 10 * time.Minute},
		{"OTP_RESEND_INTERVAL", &cfg.OTPResendInterval, time.Minute},
	} {
		*field.target, err = positiveDuration(field.name, field.fallback)
		if err != nil {
			return Config{}, err
		}
	}
	for _, field := range []struct {
		name     string
		target   *int
		fallback int
	}{
		{"AUTH_REQUESTS_PER_MINUTE", &cfg.AuthRequestsPerMinute, 20},
		{"AI_REQUESTS_PER_MINUTE", &cfg.AIRequestsPerMinute, 10},
		{"AI_MAX_TOKENS", &cfg.AIMaxTokens, 1200},
		{"OTP_MAX_ATTEMPTS", &cfg.OTPMaxAttempts, 5},
	} {
		*field.target, err = positiveInt(field.name, field.fallback)
		if err != nil {
			return Config{}, err
		}
	}
	maxBytes, err := positiveInt("HTTP_MAX_REQUEST_BYTES", 64<<20)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxRequestBytes = int64(maxBytes)
	cfg.Negotiation, err = LoadNegotiationPolicy()
	if err != nil {
		return Config{}, err
	}

	rawTrustProxy := strings.TrimSpace(os.Getenv("TRUST_PROXY_HEADERS"))
	rawTrustedCIDRs := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))

	if rawTrustProxy != "" {
		trust, err := strconv.ParseBool(rawTrustProxy)
		if err != nil {
			return Config{}, fmt.Errorf("TRUST_PROXY_HEADERS must be true or false: %w", err)
		}
		cfg.TrustProxyHeaders = trust
	}

	if cfg.TrustProxyHeaders {
		if rawTrustedCIDRs == "" {
			return Config{}, fmt.Errorf("TRUSTED_PROXY_CIDRS must list the proxy networks when TRUST_PROXY_HEADERS is true")
		}

		for _, item := range strings.Split(rawTrustedCIDRs, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}

			_, network, err := net.ParseCIDR(item)
			if err != nil {
				return Config{}, fmt.Errorf("TRUSTED_PROXY_CIDRS contains an invalid network %q: %w", item, err)
			}

			cfg.TrustedProxyCIDRs = append(cfg.TrustedProxyCIDRs, network)
		}

		if len(cfg.TrustedProxyCIDRs) == 0 {
			return Config{}, fmt.Errorf("TRUSTED_PROXY_CIDRS must contain at least one proxy network")
		}
	} else if rawTrustedCIDRs != "" {
		return Config{}, fmt.Errorf("TRUSTED_PROXY_CIDRS requires TRUST_PROXY_HEADERS=true")
	}

	// Card and bank transfer are the two funding routes we advertise.
	if cfg.FlutterwavePaymentOptions == "" {
		cfg.FlutterwavePaymentOptions = "card,banktransfer"
	}

	// Safe defaults for basic application settings.
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}

	if cfg.SMSProvider == "" {
		cfg.SMSProvider = "disabled"
	}

	if cfg.TermiiBaseURL == "" {
		cfg.TermiiBaseURL = "https://api.ng.termii.com"
	}

	// Wallet reporting threshold.
	if cfg.WalletStaleDepositHours == "" {
		cfg.WalletStaleDepositHours = "24"
	}

	// Database access is required to start the application.
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must not be empty")
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	for _, provider := range []struct{ key, model, setting string }{
		{cfg.GeminiAPIKey, cfg.GeminiModel, "GEMINI_MODEL"},
		{cfg.GroqAPIKey, cfg.GroqModel, "GROQ_MODEL"},
		{cfg.OpenRouterAPIKey, cfg.OpenRouterModel, "OPENROUTER_MODEL"},
	} {
		if provider.key != "" && provider.model == "" {
			return Config{}, fmt.Errorf("%s is required when its provider key is configured", provider.setting)
		}
	}
	for name, value := range map[string]string{"GROQ_BASE_URL": cfg.GroqBaseURL, "OPENROUTER_BASE_URL": cfg.OpenRouterBaseURL, "FLUTTERWAVE_BASE_URL": cfg.FlutterwaveBaseURL, "TERMII_BASE_URL": cfg.TermiiBaseURL} {
		if err := validateHTTPURL(name, value); err != nil {
			return Config{}, err
		}
	}
	if cfg.PaymentCurrency == "" {
		cfg.PaymentCurrency = defaultPaymentCurrency
	}
	if !validCurrency(cfg.PaymentCurrency) {
		return Config{}, fmt.Errorf("PAYMENT_CURRENCY must be a three-letter currency code")
	}
	if cfg.FlutterwaveSecretKey != "" || cfg.FlutterwaveSecretHash != "" {
		payment, err := LoadFlutterwaveConfig()
		if err != nil {
			return Config{}, err
		}
		cfg.PaymentCurrency = payment.Currency
	} else if cfg.PaymentRedirectURL != "" {
		if err := validateHTTPURL("PAYMENT_REDIRECT_URL", cfg.PaymentRedirectURL); err != nil {
			return Config{}, err
		}
	}

	// Validate SMS provider configuration.
	switch cfg.SMSProvider {
	case "disabled":
	case "termii":
		if cfg.TermiiAPIKey == "" {
			return Config{}, fmt.Errorf("TERMII_API_KEY must not be empty when SMS_PROVIDER=termii")
		}

		if cfg.TermiiSenderID == "" {
			return Config{}, fmt.Errorf("TERMII_SENDER_ID must not be empty when SMS_PROVIDER=termii")
		}
	case "brevo":
		if cfg.BrevoSMTPHost == "" || cfg.BrevoSMTPLogin == "" || cfg.BrevoSMTPKey == "" || cfg.BrevoFromEmail == "" {
			return Config{}, fmt.Errorf("Brevo SMTP host, login, key, and sender email are required when SMS_PROVIDER=brevo")
		}
	default:
		return Config{}, fmt.Errorf("SMS_PROVIDER must be disabled, termii, or brevo")
	}

	return cfg, nil
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func positiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	value, err := time.ParseDuration(envString(name, fallback.String()))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func positiveInt(name string, fallback int) (int, error) {
	value, err := strconv.Atoi(envString(name, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func validateHTTPURL(name, value string) error {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL without credentials, query, or fragment", name)
	}
	return nil
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, c := range value {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
