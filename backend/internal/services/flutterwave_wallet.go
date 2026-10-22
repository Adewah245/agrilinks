package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// These calls use the same configured provider and timeout as order payments.
func (c *FlutterwaveClient) walletRequest(ctx context.Context, method, path string, payload any, result any) error {
	if c == nil || c.secretKey == "" {
		return ErrPaymentProvider
	}
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return err
		}
	}
	r, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &body)
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+c.secretKey)
	r.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(r)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPaymentProvider, err)
	}
	defer response.Body.Close()
	data, err := readFlutterwaveBody(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: response %d", ErrPaymentProvider, response.StatusCode)
	}
	var envelope struct {
		Status string `json:"status"`
	}
	if err = json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if envelope.Status != "success" {
		return fmt.Errorf("the payment provider could not complete this request")
	}
	return json.Unmarshal(data, result)
}

type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
}
type ResolvedAccount struct {
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
}
type VerifiedTransfer struct {
	ID        json.Number `json:"id"`
	Reference string      `json:"reference"`
	Status    string      `json:"status"`
	Amount    json.Number `json:"amount"`
	Currency  string      `json:"currency"`
}

func (c *FlutterwaveClient) ListBanks(ctx context.Context) ([]Bank, error) {
	var response struct {
		Data []Bank `json:"data"`
	}
	if err := c.walletRequest(ctx, "GET", "/banks/NG", nil, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *FlutterwaveClient) ResolveAccount(ctx context.Context, number, bank string) (*ResolvedAccount, error) {
	var response struct {
		Data ResolvedAccount `json:"data"`
	}
	if err := c.walletRequest(ctx, "POST", "/accounts/resolve", map[string]string{"account_number": number, "account_bank": bank}, &response); err != nil {
		return nil, err
	}
	if response.Data.AccountNumber != number || response.Data.AccountName == "" {
		return nil, fmt.Errorf("the bank did not confirm this account")
	}
	return &response.Data, nil
}

func (c *FlutterwaveClient) CreateWalletTransfer(ctx context.Context, reference, amount, currency, bank, number, name string) (string, error) {
	var response struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	payload := map[string]any{"account_bank": bank, "account_number": number, "beneficiary_name": name, "amount": json.Number(amount), "currency": currency, "reference": reference, "narration": "Agro-Shield wallet withdrawal"}
	if err := c.walletRequest(ctx, "POST", "/transfers", payload, &response); err != nil {
		return "", err
	}
	if id, err := strconv.ParseInt(response.Data.ID.String(), 10, 64); err != nil || id <= 0 {
		return "", fmt.Errorf("the provider did not return a transfer number")
	}
	return response.Data.ID.String(), nil
}

func (c *FlutterwaveClient) VerifyWalletTransfer(ctx context.Context, id string) (*VerifiedTransfer, error) {
	if n, err := strconv.ParseInt(id, 10, 64); err != nil || n <= 0 {
		return nil, fmt.Errorf("invalid transfer number")
	}
	var response struct {
		Data VerifiedTransfer `json:"data"`
	}
	if err := c.walletRequest(ctx, "GET", "/transfers/"+url.PathEscape(id), nil, &response); err != nil {
		return nil, err
	}
	if response.Data.ID.String() != id {
		return nil, fmt.Errorf("the provider returned a different transfer")
	}
	return &response.Data, nil
}
