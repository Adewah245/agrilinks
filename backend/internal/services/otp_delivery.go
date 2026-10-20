package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

const defaultTermiiBaseURL = "https://api.ng.termii.com"

// TermiiOTPDelivery sends OTP messages through Termii's SMS API.
type TermiiOTPDelivery struct {
	apiKey   string
	senderID string
	baseURL  string
	client   *http.Client
}

func NewTermiiOTPDelivery(
	apiKey string,
	senderID string,
	baseURL string,
) (*TermiiOTPDelivery, error) {
	apiKey = strings.TrimSpace(apiKey)
	senderID = strings.TrimSpace(senderID)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	if apiKey == "" {
		return nil, fmt.Errorf("Termii API key is required")
	}
	if senderID == "" {
		return nil, fmt.Errorf("Termii sender ID is required")
	}
	if baseURL == "" {
		baseURL = defaultTermiiBaseURL
	}

	return &TermiiOTPDelivery{
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (d *TermiiOTPDelivery) SendOTP(ctx context.Context, message OTPMessage) error {
	if d == nil || d.client == nil {
		return fmt.Errorf("Termii OTP delivery is not configured")
	}

	phone := strings.TrimSpace(message.Phone)
	if phone == "" {
		return fmt.Errorf("phone number is required for SMS delivery")
	}

	payload, err := json.Marshal(struct {
		APIKey  string `json:"api_key"`
		To      string `json:"to"`
		From    string `json:"from"`
		SMS     string `json:"sms"`
		Type    string `json:"type"`
		Channel string `json:"channel"`
	}{
		APIKey:  d.apiKey,
		To:      phone,
		From:    d.senderID,
		SMS:     otpDeliveryText(message),
		Type:    "plain",
		Channel: "generic",
	})
	if err != nil {
		return fmt.Errorf("encode Termii request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		d.baseURL+"/api/sms/send",
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("create Termii request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Termii OTP: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf(
			"Termii rejected OTP delivery with status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	return nil
}

type smtpSendFunc func(
	addr string,
	auth smtp.Auth,
	from string,
	to []string,
	message []byte,
) error

// BrevoOTPDelivery sends OTP messages through Brevo's SMTP relay.
type BrevoOTPDelivery struct {
	host      string
	login     string
	key       string
	fromEmail string
	fromName  string
	sendMail  smtpSendFunc
}

func NewBrevoOTPDelivery(
	host string,
	login string,
	key string,
	fromEmail string,
	fromName string,
) (*BrevoOTPDelivery, error) {
	host = strings.TrimSpace(host)
	login = strings.TrimSpace(login)
	key = strings.TrimSpace(key)
	fromEmail = strings.TrimSpace(fromEmail)
	fromName = strings.TrimSpace(fromName)

	if host == "" || login == "" || key == "" || fromEmail == "" {
		return nil, fmt.Errorf("Brevo SMTP host, login, key, and sender email are required")
	}

	return &BrevoOTPDelivery{
		host:      host,
		login:     login,
		key:       key,
		fromEmail: fromEmail,
		fromName:  fromName,
		sendMail:  smtp.SendMail,
	}, nil
}

func (d *BrevoOTPDelivery) SendOTP(ctx context.Context, message OTPMessage) error {
	if d == nil || d.sendMail == nil {
		return fmt.Errorf("Brevo OTP delivery is not configured")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	email := strings.TrimSpace(message.Email)
	if email == "" {
		return fmt.Errorf("email address is required for email delivery")
	}
	if strings.ContainsAny(email, "\r\n") {
		return fmt.Errorf("invalid email address")
	}

	address := d.host
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "587")
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid Brevo SMTP host: %w", err)
	}

	auth := smtp.PlainAuth("", d.login, d.key, host)
	return d.sendMail(
		address,
		auth,
		d.fromEmail,
		[]string{email},
		brevoOTPMessage(d.fromEmail, d.fromName, email, message),
	)
}

func otpDeliveryText(message OTPMessage) string {
	minutes := int(time.Until(message.ExpiresAt).Minutes())
	if minutes < 1 {
		minutes = 1
	}

	return fmt.Sprintf(
		"Agro-Shield: %s is your verification code. It expires in %d minutes. Never share this code.",
		message.Code,
		minutes,
	)
}

func brevoOTPMessage(
	fromEmail string,
	fromName string,
	to string,
	message OTPMessage,
) []byte {
	from := fromEmail
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}

	return []byte(strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: Agro-Shield verification code",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		otpDeliveryText(message),
	}, "\r\n"))
}
