package models

import (
	"backend/internal/money"
	"database/sql"
	"time"
)

type Escrow struct {
	OrderID, BuyerID, SellerID                                                     int
	SellerType, BuyerName, SellerName, Products, PaymentReference, Currency, State string
	GrossKobo, CommissionKobo                                                      int64
	CommissionBPS                                                                  int64
	DeliveredAt, BuyerConfirmedAt, DecidedAt                                       sql.NullTime
	DeliveryEvidence, DisputeReason, DecisionReason                                string
	DecidedBy                                                                      sql.NullInt64
	CreatedAt                                                                      time.Time
}

func (e Escrow) Amount() string       { return money.Format(e.GrossKobo) }
func (e Escrow) Commission() string   { return money.Format(e.CommissionKobo) }
func (e Escrow) SellerCredit() string { return money.Format(e.GrossKobo - e.CommissionKobo) }
func (e Escrow) Eligible() bool {
	return e.State == "held" && e.DeliveredAt.Valid && e.BuyerConfirmedAt.Valid
}

type EscrowEvent struct {
	ActorName, Action, Reason string
	ActorID                   sql.NullInt64
	CreatedAt                 time.Time
}
type EscrowItem struct{ Product, Quantity, UnitPrice, Total string }
type EscrowPage struct {
	User                         *User
	Escrow                       *Escrow
	Items                        []EscrowItem
	Events                       []EscrowEvent
	IsBuyer, IsSeller, CanReview bool
	Error, Notice, CSRF          string
}
