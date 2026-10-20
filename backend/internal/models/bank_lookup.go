package models

import "time"

// BankLookup is one cached answer to the question of which bank an
// account number belongs to and whose name is on the account.
type BankLookup struct {
	AccountNumber string    `json:"account_number"`
	BankCode      string    `json:"bank_code"`
	BankName      string    `json:"bank_name"`
	AccountName   string    `json:"account_name"`
	ProbedAt      time.Time `json:"probed_at"`
}
