package models

import (
	"database/sql"
	"time"
)

const OTPPurposePasswordReset = "password_reset"

// OTPCode is one issued one-time passcode. The code itself is only ever held
// as a hash; nothing stores or logs the plaintext after it is sent.
type OTPCode struct {
	ID         int64
	UserID     int
	Phone      string
	Purpose    string
	CodeHash   string
	Attempts   int
	ExpiresAt  time.Time
	ConsumedAt sql.NullTime
	CreatedAt  time.Time
}
