package services

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type CreateVirtualAccountRequest struct {
	Email, TxRef, Phone, FirstName, LastName, Narration string
	IdentityKind, IdentityNumber string
}

type VirtualAccountDetails struct {
	TxRef string `json:"tx_ref"`
	OrderRef string `json:"order_ref"`
	ProviderRef string `json:"flw_ref"`
	AccountNumber string `json:"account_number"`
	BankName string `json:"bank_name"`
	AccountName string `json:"account_name"`
	Currency string `json:"currency"`
	Status string `json:"status"`
}

func (c *FlutterwaveClient) CreatePermanentVirtualAccount(ctx context.Context, request CreateVirtualAccountRequest) (*VirtualAccountDetails, error) {
	if (request.IdentityKind != "bvn" && request.IdentityKind != "nin") || request.IdentityNumber == "" || request.Email == "" || request.TxRef == "" {
		return nil, errors.New("funding account details are incomplete")
	}
	payload := map[string]any{"email":request.Email,"tx_ref":request.TxRef,"is_permanent":true,"narration":request.Narration,request.IdentityKind:request.IdentityNumber}
	for name, value := range map[string]string{"phonenumber":request.Phone,"firstname":request.FirstName,"lastname":request.LastName} {
		if strings.TrimSpace(value) != "" { payload[name] = value }
	}
	var response struct { Data VirtualAccountDetails `json:"data"` }
	if err := c.walletRequest(ctx, "POST", "/virtual-account-numbers", payload, &response); err != nil { return nil, err }
	response.Data.TxRef = request.TxRef
	if response.Data.AccountName == "" { response.Data.AccountName = request.Narration }
	if response.Data.Currency == "" { response.Data.Currency = "NGN" }
	return &response.Data, nil
}

func (c *FlutterwaveClient) GetPermanentVirtualAccount(ctx context.Context, reference string) (*VirtualAccountDetails, error) {
	if strings.TrimSpace(reference) == "" { return nil, errors.New("a funding account reference is required") }
	var response struct { Data VirtualAccountDetails `json:"data"` }
	if err := c.walletRequest(ctx, "GET", "/virtual-account-numbers/"+url.PathEscape(reference), nil, &response); err != nil { return nil, err }
	if response.Data.Currency == "" { response.Data.Currency = "NGN" }
	return &response.Data, nil
}

func (c *FlutterwaveClient) VerifyTransactionByReference(ctx context.Context, reference string) (*VerifiedTransaction, error) {
	if strings.TrimSpace(reference) == "" { return nil, errors.New("a payment reference is required") }
	var response verifyTransactionResponse
	if err := c.walletRequest(ctx, "GET", "/transactions/verify_by_reference?tx_ref="+url.QueryEscape(reference), nil, &response); err != nil { return nil, err }
	if response.Data.ID <= 0 || response.Data.TxRef != reference || response.Data.Status == "" || response.Data.Currency == "" { return nil, ErrPaymentProvider }
	return &VerifiedTransaction{TransactionID:strconv.FormatInt(response.Data.ID,10),Reference:response.Data.TxRef,Status:response.Data.Status,Amount:response.Data.Amount,Currency:strings.ToUpper(response.Data.Currency)}, nil
}
