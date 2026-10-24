package models

import "time"

// RegistrationPhoneCode is a challenge for a phone that has no account yet.
type RegistrationPhoneCode struct {
	TokenHash, Phone, CodeHash string
	Attempts                   int
	ExpiresAt                  time.Time
}
