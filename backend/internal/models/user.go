package models

import "time"

type User struct {
	ID int

	// Identity
	FullName string
	Phone    string
	Email    string

	// Security
	PasswordHash string

	// Location
	Location string
	LGA      string
	State    string
	Country  string

	// Account
	Role        string
	AccountType string

	// Verification
	NIN                string
	VerificationStatus string

	// Organisation / Partner
	OrganisationName string

	// Profile
	PhotoURL string

	// Farming information
	FarmingTypes []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) IsBuyer() bool {
	return u.Role == "buyer"
}

func (u *User) IsFarmer() bool {
	return u.Role == "farmer"
}

func (u *User) IsOrganisation() bool {
	return u.Role == "organisation"
}

func (u *User) IsMinistry() bool {
	return u.Role == "ministry"
}