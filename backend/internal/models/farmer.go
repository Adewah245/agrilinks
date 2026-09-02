package models

import "time"

type Farmer struct {

	ID int

	// Identity
	FullName string
	Phone string
	Email string

	// Security
	PasswordHash string

	// Location
	Location string
	LGA string
	State string
	Country string

	// Account
	Role string // farmer, buyer, ministry, organisation
	AccountType string

	// Verification
	NIN string
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


func (f *Farmer) IsBuyer() bool {
	return f.Role == "buyer"
}


func (f *Farmer) IsFarmer() bool {
	return f.Role == "farmer"
}


func (f *Farmer) IsOrganisation() bool {
	return f.Role == "organisation"
}


func (f *Farmer) IsMinistry() bool {
	return f.Role == "ministry"
}