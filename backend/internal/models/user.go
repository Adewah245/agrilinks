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
	IsAdmin     bool

	// Verification
	NIN                string
	VerificationStatus string

	// Organisation / Partner
	OrganisationName string

	// Profile
	PhotoURL string

	// Farming information
	FarmingTypes []string

	// Payment details.
	// These fields are populated only by payment-specific queries.
	BankName      string
	BankCode      string
	AccountName   string
	AccountNumber string
	WalletNumber  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserBankAccount represents the payout account belonging to a user.
type UserBankAccount struct {
	ID            int
	UserID        int
	BankName      string
	BankCode      string
	AccountName   string
	AccountNumber string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsBuyer reports whether the user has a buyer role.
func (u *User) IsBuyer() bool {
	return u.Role == "buyer"
}

// IsLearner reports whether the user has a learner role.
func (u *User) IsLearner() bool {
	return u.Role == "learner"
}

// IsFarmer reports whether the user has a farmer role.
func (u *User) IsFarmer() bool {
	return u.Role == "farmer"
}

// IsOrganisation reports whether the user has an organisation role.
func (u *User) IsOrganisation() bool {
	return u.Role == "organisation"
}

// IsMinistry reports whether the user has a ministry role.
func (u *User) IsMinistry() bool {
	return u.Role == "ministry"
}

// IsAdministrator reports whether the account can review platform-wide
// operations such as delivery partner approval.
func (u *User) IsAdministrator() bool {
	return u != nil && u.IsAdmin
}
