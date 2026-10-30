package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	flutterwaveBaseURL          = "https://api.flutterwave.com/v3"
	flutterwaveRequestTimeout   = 10 * time.Second
	maxFlutterwaveResponseBytes = 1 << 20
)

// PaymentGateway is the provider boundary used by PaymentService.
type PaymentGateway interface {
	InitializePayment(context.Context, InitializePaymentRequest) (string, error)
	VerifyTransaction(context.Context, string) (*VerifiedTransaction, error)
}

// FlutterwaveClient implements PaymentGateway for Flutterwave hosted checkout.
type FlutterwaveClient struct {
	baseURL        string
	secretKey      string
	redirectURL    string
	paymentOptions string
	httpClient     *http.Client
	bankMu         sync.Mutex
	bankCache      []Bank
	bankCacheUntil time.Time
}

func NewFlutterwaveClient(
	secretKey,
	redirectURL,
	paymentOptions string,
) *FlutterwaveClient {
	return &FlutterwaveClient{
		baseURL:        flutterwaveBaseURL,
		secretKey:      strings.TrimSpace(secretKey),
		redirectURL:    strings.TrimSpace(redirectURL),
		paymentOptions: strings.TrimSpace(paymentOptions),
		httpClient:     &http.Client{Timeout: flutterwaveRequestTimeout},
	}
}

func (c *FlutterwaveClient) SetBaseURL(value string) { c.baseURL = strings.TrimRight(value, "/") }

type InitializePaymentRequest struct {
	Reference     string
	Amount        string
	Currency      string
	CustomerEmail string
	CustomerName  string
	Description   string
}

type initializePaymentPayload struct {
	TxRef          string `json:"tx_ref"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	RedirectURL    string `json:"redirect_url"`
	PaymentOptions string `json:"payment_options,omitempty"`
	Customer       struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"customer"`
	Customizations struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"customizations"`
}

type initializePaymentResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Link string `json:"link"`
	} `json:"data"`
}

func (c *FlutterwaveClient) InitializePayment(
	ctx context.Context,
	req InitializePaymentRequest,
) (string, error) {
	if c == nil || c.httpClient == nil || c.secretKey == "" {
		return "", fmt.Errorf("flutterwave is not configured")
	}

	if strings.TrimSpace(req.Reference) == "" ||
		strings.TrimSpace(req.Amount) == "" ||
		strings.TrimSpace(req.Currency) == "" ||
		strings.TrimSpace(req.CustomerEmail) == "" {
		return "", fmt.Errorf("incomplete Flutterwave payment request")
	}

	payload := initializePaymentPayload{
		TxRef:          req.Reference,
		Amount:         req.Amount,
		Currency:       req.Currency,
		RedirectURL:    c.redirectURL,
		PaymentOptions: c.paymentOptions,
	}
	payload.Customer.Email = req.CustomerEmail
	payload.Customer.Name = req.CustomerName
	payload.Customizations.Title = "Agro-Shield"
	payload.Customizations.Description = req.Description

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode Flutterwave request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/payments",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("build Flutterwave request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call Flutterwave: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := readFlutterwaveBody(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"flutterwave payment initialization failed with status %d",
			resp.StatusCode,
		)
	}

	var parsed initializePaymentResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode Flutterwave response: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(parsed.Status), "success") {
		return "", fmt.Errorf("flutterwave rejected the payment request")
	}
	if strings.TrimSpace(parsed.Data.Link) == "" {
		return "", fmt.Errorf("flutterwave returned no checkout URL")
	}
	checkoutURL, err := url.Parse(parsed.Data.Link)
	if err != nil ||
		(checkoutURL.Scheme != "http" && checkoutURL.Scheme != "https") ||
		checkoutURL.Host == "" {
		return "", fmt.Errorf("flutterwave returned an invalid checkout URL")
	}

	return checkoutURL.String(), nil
}

type VerifiedTransaction struct {
	TransactionID string
	Reference     string
	Status        string
	Amount        json.Number
	Currency      string
}

type verifyTransactionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID       int64       `json:"id"`
		TxRef    string      `json:"tx_ref"`
		Status   string      `json:"status"`
		Amount   json.Number `json:"amount"`
		Currency string      `json:"currency"`
	} `json:"data"`
}

func (c *FlutterwaveClient) VerifyTransaction(
	ctx context.Context,
	transactionID string,
) (*VerifiedTransaction, error) {
	if c == nil || c.httpClient == nil || c.secretKey == "" {
		return nil, fmt.Errorf("flutterwave is not configured")
	}

	transactionID = strings.TrimSpace(transactionID)
	parsedID, err := strconv.ParseInt(transactionID, 10, 64)
	if err != nil || parsedID <= 0 {
		return nil, fmt.Errorf("transaction ID is required")
	}
	transactionID = strconv.FormatInt(parsedID, 10)

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/transactions/"+transactionID+"/verify",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build Flutterwave verification request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("verify Flutterwave transaction: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := readFlutterwaveBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"flutterwave verification failed with status %d",
			resp.StatusCode,
		)
	}

	var parsed verifyTransactionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode Flutterwave verification: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(parsed.Status), "success") {
		return nil, fmt.Errorf("flutterwave could not verify the transaction")
	}
	if parsed.Data.ID <= 0 ||
		strings.TrimSpace(parsed.Data.TxRef) == "" ||
		strings.TrimSpace(parsed.Data.Status) == "" ||
		strings.TrimSpace(parsed.Data.Currency) == "" {
		return nil, fmt.Errorf("flutterwave verification response is incomplete")
	}
	if strings.EqualFold(strings.TrimSpace(parsed.Data.Status), "successful") {
		if _, err := normalizedMoney(parsed.Data.Amount.String()); err != nil {
			return nil, fmt.Errorf("flutterwave returned a successful transaction without a valid amount")
		}
	}

	return &VerifiedTransaction{
		TransactionID: fmt.Sprintf("%d", parsed.Data.ID),
		Reference:     strings.TrimSpace(parsed.Data.TxRef),
		Status:        strings.TrimSpace(parsed.Data.Status),
		Amount:        parsed.Data.Amount,
		Currency:      strings.ToUpper(strings.TrimSpace(parsed.Data.Currency)),
	}, nil
}

func readFlutterwaveBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxFlutterwaveResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Flutterwave response: %w", err)
	}
	if len(data) > maxFlutterwaveResponseBytes {
		return nil, fmt.Errorf("flutterwave response is too large")
	}
	return data, nil
}
