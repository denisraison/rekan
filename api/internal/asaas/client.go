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

type Subscription struct {
	ID          string `json:"id"`
	PaymentLink string `json:"paymentLink"`
}

type CreateSubscriptionReq struct {
	Customer    string  `json:"customer"`
	BillingType string  `json:"billingType"`
	Value       float64 `json:"value"`
	NextDueDate string  `json:"nextDueDate"`
	Cycle       string  `json:"cycle"`
	Description string  `json:"description"`
}

func (c *Client) CreateCustomer(ctx context.Context, name, email string) (Customer, error) {
	body := map[string]string{"name": name, "email": email}
	var out Customer
	if err := c.post(ctx, "/customers", body, &out); err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return out, nil
}

func (c *Client) CreateSubscription(ctx context.Context, req CreateSubscriptionReq) (Subscription, error) {
	var out Subscription
	if err := c.post(ctx, "/subscriptions", req, &out); err != nil {
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}
	return out, nil
}

func (c *Client) UpdateSubscription(ctx context.Context, id string, value float64) error {
	var out map[string]any
	return c.put(ctx, "/subscriptions/"+id, map[string]float64{"value": value}, &out)
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

func (c *Client) put(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(b))
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
