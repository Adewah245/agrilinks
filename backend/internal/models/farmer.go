package models

import "time"

type Farmer struct {
	ID           int
	FullName     string
	Phone        string
	Email        string
	PasswordHash string
	Location     string
	LGA          string
	State        string
	Country      string
	Role         string // "farmer" or "buyer"
	PhotoURL     string

	// NIN verification
	// NIN is optional for now.
	NIN                      string
	NINVerified              bool
	NINVerificationReference string
	NINVerifiedAt            *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f *Farmer) IsBuyer() bool {
	return f.Role == "buyer"
}

func (f *Farmer) IsFarmer() bool {
	return f.Role != "buyer"
}
