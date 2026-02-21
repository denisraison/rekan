package asaas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCustomer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/customers" {
			t.Errorf("expected /customers, got %s", r.URL.Path)
		}
		if r.Header.Get("access_token") != "test-key" {
			t.Errorf("missing or wrong access_token header")
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["name"] != "João Silva" || body["email"] != "joao@example.com" {
			t.Errorf("unexpected body: %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "cus_test123"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	customer, err := c.CreateCustomer(context.Background(), "João Silva", "joao@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customer.ID != "cus_test123" {
		t.Errorf("expected ID cus_test123, got %s", customer.ID)
	}
}

func TestCreateSubscription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscriptions" {
			t.Errorf("expected /subscriptions, got %s", r.URL.Path)
		}

		var body CreateSubscriptionReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Customer != "cus_test123" {
			t.Errorf("expected customer cus_test123, got %s", body.Customer)
		}
		if body.BillingType != "PIX" {
			t.Errorf("expected PIX, got %s", body.BillingType)
		}
		if body.Cycle != "MONTHLY" {
			t.Errorf("expected MONTHLY, got %s", body.Cycle)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":          "sub_test456",
			"paymentLink": "https://pay.asaas.com/sub_test456",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	sub, err := c.CreateSubscription(context.Background(), CreateSubscriptionReq{
		Customer:    "cus_test123",
		BillingType: "PIX",
		Value:       89.90,
		NextDueDate: "2026-02-21",
		Cycle:       "MONTHLY",
		Description: "Rekan - Plano Mensal",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID != "sub_test456" {
		t.Errorf("expected ID sub_test456, got %s", sub.ID)
	}
	if sub.PaymentLink != "https://pay.asaas.com/sub_test456" {
		t.Errorf("unexpected payment link: %s", sub.PaymentLink)
	}
}

func TestErrorWithDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"description": "CPF/CNPJ inválido"},
			},
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "key", http: &http.Client{}}
	_, err := c.CreateCustomer(context.Background(), "Test", "test@test.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestErrorWithoutDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "key", http: &http.Client{}}
	_, err := c.CreateCustomer(context.Background(), "Test", "test@test.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewClientSandboxURL(t *testing.T) {
	c := NewClient("key", true)
	if c.baseURL != sandboxBaseURL {
		t.Errorf("expected sandbox URL, got %s", c.baseURL)
	}
}

func TestNewClientProdURL(t *testing.T) {
	c := NewClient("key", false)
	if c.baseURL != prodBaseURL {
		t.Errorf("expected prod URL, got %s", c.baseURL)
	}
}
