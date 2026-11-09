package services

import (
	"context"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"backend/internal/models"
)

var (
	ErrInvalidRecipient   = errors.New("enter a wallet ID, user ID or Nigerian phone number")
	ErrRecipientNotFound  = errors.New("no active wallet found; check the wallet ID, user ID or phone number")
	ErrRecipientAmbiguous = errors.New("this number matches more than one account; use the recipient's wallet ID")
	ErrRecipientIsSelf    = errors.New("choose another user's wallet to send money")
	recipientDigits       = regexp.MustCompile(`^[0-9]+$`)
	recipientPhone        = regexp.MustCompile(`^\+?[0-9][0-9 ()-]*$`)
	recipientWallet       = regexp.MustCompile(`^[A-Z][A-Z0-9-]{2,63}$`)
)

func recipientAliases(input string) (wallet string, id int, phone string, err error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 || len(input) > 64 {
		return "", 0, "", ErrInvalidRecipient
	}
	upper := strings.ToUpper(input)
	if recipientWallet.MatchString(upper) {
		return upper, 0, "", nil
	}
	if recipientDigits.MatchString(input) {
		if value, parseErr := strconv.ParseInt(input, 10, 32); parseErr == nil && value > 0 {
			id = int(value)
		}
	}
	if recipientPhone.MatchString(input) {
		phone = NormalizeNigerianPhone(input)
	}
	if id == 0 && phone == "" {
		return "", 0, "", ErrInvalidRecipient
	}
	return "", id, phone, nil
}

func (s *WalletService) ResolveRecipient(ctx context.Context, userID int, input string) (*models.WalletRecipient, error) {
	wallet, id, phone, err := recipientAliases(input)
	if err != nil {
		return nil, err
	}
	recipients, err := s.repo.FindWalletRecipients(ctx, wallet, id, phone)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, ErrRecipientNotFound
	}
	if len(recipients) > 1 {
		return nil, ErrRecipientAmbiguous
	}
	recipient := &recipients[0]
	if recipient.UserID == userID {
		return nil, ErrRecipientIsSelf
	}
	recipient.PhotoURL = publicRecipientPhoto(recipient.PhotoURL)
	return recipient, nil
}

// Registration selfies and withdrawal evidence are private, including legacy
// uploads/users paths. Only local public profile assets can leave this endpoint.
func publicRecipientPhoto(value string) string {
	u, err := url.Parse(strings.TrimSpace(value))
	if err == nil && u.Scheme == "" && u.Host == "" && u.RawQuery == "" && u.Fragment == "" && !strings.Contains(u.Path, "\\") {
		clean := path.Clean(u.Path)
		if clean == u.Path && (strings.HasPrefix(clean, "/static/uploads/profiles/") || strings.HasPrefix(clean, "/static/assets/")) {
			return clean
		}
	}
	return "/static/assets/images/un.jpeg"
}
