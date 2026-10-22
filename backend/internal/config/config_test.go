package config

import (
	"testing"
	"time"
)

func setValidTestEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "test-diagnosis-model")
	t.Setenv("GROQ_API_KEY", "test-key")
	t.Setenv("GROQ_MODEL", "test-learning-model")
	t.Setenv("PAYMENT_REDIRECT_URL", "https://example.test/payment/callback")
	t.Setenv("SMS_PROVIDER", "disabled")
	t.Setenv("TRUST_PROXY_HEADERS", "")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")
}

func TestLoadRejectsForwardedHeadersWithoutTrustedProxyNetworks(t *testing.T) {
	setValidTestEnvironment(t)
	t.Setenv("TRUST_PROXY_HEADERS", "true")

	if _, err := Load(); err == nil {
		t.Fatal("expected startup to reject trust without TRUSTED_PROXY_CIDRS")
	}
}

func TestLoadRejectsTrustedProxyNetworksWithoutExplicitTrust(t *testing.T) {
	setValidTestEnvironment(t)
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")

	if _, err := Load(); err == nil {
		t.Fatal("expected startup to reject TRUSTED_PROXY_CIDRS without TRUST_PROXY_HEADERS=true")
	}
}

func TestLoadRejectsInvalidTrustedProxyNetwork(t *testing.T) {
	setValidTestEnvironment(t)
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,not-a-network")

	if _, err := Load(); err == nil {
		t.Fatal("expected startup to reject an invalid proxy CIDR")
	}
}

func TestLoadParsesTrustedProxyNetworksOnlyWhenEnabled(t *testing.T) {
	setValidTestEnvironment(t)
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 192.168.1.0/24")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.TrustProxyHeaders {
		t.Fatal("expected TrustProxyHeaders to be enabled")
	}
	if len(cfg.TrustedProxyCIDRs) != 2 {
		t.Fatalf("got %d trusted networks, want 2", len(cfg.TrustedProxyCIDRs))
	}
	if got := cfg.TrustedProxyCIDRs[0].String(); got != "10.0.0.0/8" {
		t.Errorf("first network = %q", got)
	}
}

func TestLoadDefaultsToNoForwardedHeaderTrust(t *testing.T) {
	setValidTestEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TrustProxyHeaders || len(cfg.TrustedProxyCIDRs) != 0 {
		t.Fatalf("forwarded headers trusted by default: trust=%v nets=%v", cfg.TrustProxyHeaders, cfg.TrustedProxyCIDRs)
	}
}

func TestLoadNegotiationPolicyDefaultsMatchDocumentedRules(t *testing.T) {
	t.Setenv("NEGOTIATION_MAX_ROUNDS", "")
	t.Setenv("NEGOTIATION_WINDOW", "")

	policy, err := LoadNegotiationPolicy()
	if err != nil {
		t.Fatalf("LoadNegotiationPolicy() error = %v", err)
	}

	if policy.MaxRounds != 5 {
		t.Fatalf("MaxRounds = %d, want 5", policy.MaxRounds)
	}

	if policy.Window != 24*time.Hour {
		t.Fatalf("Window = %v, want 24h", policy.Window)
	}
}

func TestLoadNegotiationPolicyReadsOverrides(t *testing.T) {
	t.Setenv("NEGOTIATION_MAX_ROUNDS", "7")
	t.Setenv("NEGOTIATION_WINDOW", "36h")

	policy, err := LoadNegotiationPolicy()
	if err != nil {
		t.Fatalf("LoadNegotiationPolicy() error = %v", err)
	}

	if policy.MaxRounds != 7 {
		t.Fatalf("MaxRounds = %d, want 7", policy.MaxRounds)
	}

	if policy.Window != 36*time.Hour {
		t.Fatalf("Window = %v, want 36h", policy.Window)
	}
}

func TestLoadNegotiationPolicyRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		maxRounds string
		window    string
	}{
		{name: "non-numeric rounds", maxRounds: "five"},
		{name: "zero rounds", maxRounds: "0"},
		{name: "invalid duration", window: "tomorrow"},
		{name: "zero duration", window: "0s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("NEGOTIATION_MAX_ROUNDS", test.maxRounds)
			t.Setenv("NEGOTIATION_WINDOW", test.window)

			if _, err := LoadNegotiationPolicy(); err == nil {
				t.Fatal("expected invalid negotiation policy to be rejected")
			}
		})
	}
}

func setValidFlutterwaveEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("FLUTTERWAVE_SECRET_KEY", "test-secret-key")
	t.Setenv("FLUTTERWAVE_SECRET_HASH", "test-secret-hash")
	t.Setenv("PAYMENT_REDIRECT_URL", "https://example.test/payments/callback")
	t.Setenv("FLUTTERWAVE_PAYMENT_OPTIONS", "")
	t.Setenv("PAYMENT_CURRENCY", "")
}

func TestLoadFlutterwaveConfigDefaultsMatchDocumentedRules(t *testing.T) {
	setValidFlutterwaveEnvironment(t)

	cfg, err := LoadFlutterwaveConfig()
	if err != nil {
		t.Fatalf("LoadFlutterwaveConfig() error = %v", err)
	}
	if cfg.Currency != "NGN" {
		t.Fatalf("Currency = %q, want NGN", cfg.Currency)
	}
	if cfg.PaymentOptions != "card,banktransfer" {
		t.Fatalf("PaymentOptions = %q", cfg.PaymentOptions)
	}
}

func TestLoadFlutterwaveConfigReadsOverrides(t *testing.T) {
	setValidFlutterwaveEnvironment(t)
	t.Setenv("PAYMENT_CURRENCY", "usd")
	t.Setenv("FLUTTERWAVE_PAYMENT_OPTIONS", "card,ussd")

	cfg, err := LoadFlutterwaveConfig()
	if err != nil {
		t.Fatalf("LoadFlutterwaveConfig() error = %v", err)
	}
	if cfg.Currency != "USD" {
		t.Fatalf("Currency = %q, want USD", cfg.Currency)
	}
	if cfg.PaymentOptions != "card,ussd" {
		t.Fatalf("PaymentOptions = %q", cfg.PaymentOptions)
	}
}

func TestLoadFlutterwaveConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T)
	}{
		{
			name: "missing secret key",
			mutate: func(t *testing.T) {
				t.Setenv("FLUTTERWAVE_SECRET_KEY", "")
			},
		},
		{
			name: "missing secret hash",
			mutate: func(t *testing.T) {
				t.Setenv("FLUTTERWAVE_SECRET_HASH", "")
			},
		},
		{
			name: "relative redirect URL",
			mutate: func(t *testing.T) {
				t.Setenv("PAYMENT_REDIRECT_URL", "/payments/callback")
			},
		},
		{
			name: "invalid currency",
			mutate: func(t *testing.T) {
				t.Setenv("PAYMENT_CURRENCY", "US")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidFlutterwaveEnvironment(t)
			test.mutate(t)

			if _, err := LoadFlutterwaveConfig(); err == nil {
				t.Fatal("expected invalid Flutterwave configuration to be rejected")
			}
		})
	}
}
