package models

import (
	"database/sql"
	"time"
)

// WalletConfirmation is internal verification state and is never serialized.
type WalletConfirmation struct {
	TokenHash, SessionHash             string
	UserID                             int
	Kind, Currency, Amount             string
	RecipientWallet, Note, QuoteID     string
	BankCode, AccountNumber, Reference string
	OTPHash                            string
	MaxAttempts, Attempts              int
	ExpiresAt, CreatedAt               time.Time
	SentAt, VerifiedAt, InvalidatedAt  sql.NullTime
	FinalReference                     sql.NullString
}

type WalletConfirmationResult struct {
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Reference string `json:"reference"`
}
