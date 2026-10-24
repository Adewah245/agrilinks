package services

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

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
