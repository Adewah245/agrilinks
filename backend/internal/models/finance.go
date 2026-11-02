package models

import (
	"backend/internal/money"
	"database/sql"
	"time"
)

type FinanceAccount struct {
	Code, Name, Category, Currency     string
	DebitKobo, CreditKobo, BalanceKobo int64
}

func (a FinanceAccount) Debit() string   { return money.Format(a.DebitKobo) }
func (a FinanceAccount) Credit() string  { return money.Format(a.CreditKobo) }
func (a FinanceAccount) Balance() string { return money.Format(a.BalanceKobo) }

type FinanceMetric struct{ Label, Value, Detail string }
type CompanyAccount struct {
	ID                                                                    int64
	CompanyName, BankName, BankCode, AccountNumber, AccountName, Currency string
}
type TransferPolicy struct {
	ID, PerTransferLimit, DailyLimit, SeniorThreshold int64
	CompanyAccountID                                  sql.NullInt64
}
type CompanyQuote struct {
	ID                                                              string
	InitiatedBy                                                     int
	AmountKobo, ProviderKobo, TotalKobo, CompanyAccountID, PolicyID int64
	ProviderEvidence                                                string
	ExpiresAt                                                       time.Time
}

func (q CompanyQuote) Amount() string      { return money.Format(q.AmountKobo) }
func (q CompanyQuote) ProviderFee() string { return money.Format(q.ProviderKobo) }
func (q CompanyQuote) Total() string       { return money.Format(q.TotalKobo) }

type CompanyWithdrawal struct {
	ActualProviderKobo                                          sql.NullInt64
	ID                                                          int64
	Reference, QuoteID, Currency                                string
	AmountKobo, ProviderKobo, TotalKobo, CompanyAccountID       int64
	InitiatedBy                                                 int
	ApprovedBy                                                  sql.NullInt64
	Reason, ApprovalReason, Status, ProviderID, SubmissionState string
	CreatedAt                                                   time.Time
	ApprovedAt, VerifiedAt                                      sql.NullTime
}

func (w CompanyWithdrawal) Amount() string { return money.Format(w.AmountKobo) }
func (w CompanyWithdrawal) ProviderFee() string {
	if w.ActualProviderKobo.Valid {
		return money.Format(w.ActualProviderKobo.Int64)
	}
	return money.Format(w.ProviderKobo)
}
func (w CompanyWithdrawal) Total() string { return money.Format(w.TotalKobo) }

type FinanceChange struct {
	ID                                                 int64
	Key, Kind, Payload, Reason, Status, DecisionReason string
	InitiatedBy                                        int
	ApprovedBy                                         sql.NullInt64
	CreatedAt                                          time.Time
}
type SettlementItem struct {
	Reference                          string
	GrossKobo, DeductionsKobo, NetKobo int64
}
type SettlementBatch struct {
	Direction                                                                                        string
	ID                                                                                               int64
	Provider, Reference, Currency, Destination, Status, Source, Evidence, StatementReference, Reason string
	GrossKobo, DeductionsKobo, NetKobo, StatementKobo, CompanyAccountID                              int64
	ImportedBy                                                                                       int
	ApprovedBy                                                                                       sql.NullInt64
	Items                                                                                            []SettlementItem
}
type FinancePage struct {
	User                                      *User
	Escrows                                   []Escrow
	Metrics                                   []FinanceMetric
	Accounts                                  []FinanceAccount
	Withdrawals                               []CompanyWithdrawal
	Changes                                   []FinanceChange
	Settlements                               []SettlementBatch
	CompanyAccount                            *CompanyAccount
	Quote                                     *CompanyQuote
	Policy                                    *TransferPolicy
	FeePolicy                                 *FeePolicy
	CSRF, Error, Notice, ReportFrom, ReportTo string
	CanRequest, CanApprove                    bool
}
