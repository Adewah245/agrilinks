package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OTPDevDelivery sends Agro-Shield's generated codes through OTP.dev. Local
// verification retains the user/purpose binding, expiry, and single-use rules.
type OTPDevDelivery struct {
	apiKey, sender, templateID, baseURL string
	client                              *http.Client
}

func NewOTPDevDelivery(apiKey, sender, templateID, baseURL string) (*OTPDevDelivery, error) {
	apiKey = strings.TrimSpace(apiKey)
	sender = strings.TrimSpace(sender)
	templateID = strings.TrimSpace(templateID)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if apiKey == "" || sender == "" || templateID == "" {
		return nil, errors.New("OTP.dev API key, sender, and template ID are required")
	}
	if baseURL == "" {
		baseURL = "https://api.otp.dev/v1"
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("OTP.dev base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	return &OTPDevDelivery{
		apiKey: apiKey, sender: sender, templateID: templateID, baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// Do not forward the API key or resend a code to a redirect target.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (d *OTPDevDelivery) SendOTP(ctx context.Context, message OTPMessage) error {
	if d == nil || d.client == nil {
		return errors.New("OTP.dev delivery is not configured")
	}
	phone := NormalizeNigerianPhone(message.Phone)
	if phone == "" {
		return errors.New("a valid Nigerian phone number is required for OTP.dev delivery")
	}
	if len(message.Code) != otpCodeDigits || strings.IndexFunc(message.Code, func(r rune) bool { return r < '0' || r > '9' }) != -1 {
		return errors.New("OTP.dev delivery requires a six-digit code")
	}

	// OTP.dev accepts either code or code_length. Supplying our code ensures
	// the delivered value matches the hash stored by OTPService.
	payload, err := json.Marshal(map[string]any{"data": map[string]string{
		"channel": "sms", "sender": d.sender, "phone": phone,
		"template": d.templateID, "code": message.Code,
	}})
	if err != nil {
		return fmt.Errorf("encode OTP.dev request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/verifications", bytes.NewReader(payload))
	if err != nil {
		return errors.New("could not create OTP.dev request")
	}
	request.Header.Set("X-OTP-Key", d.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("OTP.dev delivery request failed")
	}
	defer response.Body.Close()
	// Provider errors can echo the request, including the code and phone.
	// Keep those bodies out of errors that handlers may log or render.
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OTP.dev rejected delivery (HTTP %d)", response.StatusCode)
	}
	var result struct {
		MessageID string `json:"message_id"`
		Phone     string `json:"phone"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil || strings.TrimSpace(result.MessageID) == "" || result.Phone != phone {
		return errors.New("OTP.dev returned an invalid delivery confirmation")
	}
	return nil
}
