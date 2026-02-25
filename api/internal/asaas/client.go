package asaas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	prodBaseURL    = "https://api.asaas.com/v3"
	sandboxBaseURL = "https://sandbox.asaas.com/api/v3"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(apiKey string, sandbox bool) *Client {
	base := prodBaseURL
	if sandbox {
		base = sandboxBaseURL
	}
	return &Client{baseURL: base, apiKey: apiKey, http: &http.Client{}}
}

// NewTestClient creates a client pointing at a custom base URL (e.g. httptest.NewServer).
func NewTestClient(baseURL, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, http: &http.Client{}}
}

type Customer struct {
	ID string `json:"id"`
}

type Authorization struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Payload string `json:"payload"` // Pix copia-e-cola for the combined QR code
}

type ImmediateQrCodeReq struct {
	Value             float64 `json:"value"`
	OriginalValue     float64 `json:"originalValue"`
	DueDate           string  `json:"dueDate"`
	ExpirationSeconds int     `json:"expirationSeconds"`
}

type CreateAuthorizationReq struct {
	CustomerID       string             `json:"customerId"`
	Description      string             `json:"description"`
	Frequency        string             `json:"frequency"` // MONTHLY, WEEKLY, etc.
	ContractID       string             `json:"contractId"`
	StartDate        string             `json:"startDate"`
	ImmediateQrCode  ImmediateQrCodeReq `json:"immediateQrCode"`
}

type Payment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type CreateChargeReq struct {
	Customer                      string  `json:"customer"`
	BillingType                   string  `json:"billingType"`
	Value                         float64 `json:"value"`
	DueDate                       string  `json:"dueDate"`
	Description                   string  `json:"description"`
	ExternalReference             string  `json:"externalReference,omitempty"`
	PixAutomaticAuthorizationId   string  `json:"pixAutomaticAuthorizationId"`
}

func (c *Client) CreateCustomer(ctx context.Context, name, email, cpfCnpj string) (Customer, error) {
	body := map[string]string{"name": name, "email": email, "cpfCnpj": cpfCnpj}
	var out Customer
	if err := c.post(ctx, "/customers", body, &out); err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return out, nil
}

func (c *Client) CreateAuthorization(ctx context.Context, req CreateAuthorizationReq) (Authorization, error) {
	var out Authorization
	if err := c.post(ctx, "/pix/automatic/authorizations", req, &out); err != nil {
		return Authorization{}, fmt.Errorf("create authorization: %w", err)
	}
	return out, nil
}

func (c *Client) CancelAuthorization(ctx context.Context, id string) error {
	return c.del(ctx, "/pix/automatic/authorizations/"+id)
}

func (c *Client) CreateCharge(ctx context.Context, req CreateChargeReq) (Payment, error) {
	var out Payment
	if err := c.post(ctx, "/payments", req, &out); err != nil {
		return Payment{}, fmt.Errorf("create charge: %w", err)
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errResp struct {
			Errors []struct {
				Description string `json:"description"`
			} `json:"errors"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if len(errResp.Errors) > 0 {
			return fmt.Errorf("asaas %s: %s", resp.Status, errResp.Errors[0].Description)
		}
		return fmt.Errorf("asaas %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) del(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("access_token", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errResp struct {
			Errors []struct {
				Description string `json:"description"`
			} `json:"errors"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if len(errResp.Errors) > 0 {
			return fmt.Errorf("asaas %s: %s", resp.Status, errResp.Errors[0].Description)
		}
		return fmt.Errorf("asaas %s", resp.Status)
	}
	return nil
}
