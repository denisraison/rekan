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
	customer, err := c.CreateCustomer(context.Background(), "João Silva", "joao@example.com", "")
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
	_, err := c.CreateCustomer(context.Background(), "Test", "test@test.com", "")
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
	_, err := c.CreateCustomer(context.Background(), "Test", "test@test.com", "")
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

func TestCreateCustomerWithCPF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["cpfCnpj"] != "12345678900" {
			t.Errorf("expected cpfCnpj 12345678900, got %s", body["cpfCnpj"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "cus_cpf123"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	customer, err := c.CreateCustomer(context.Background(), "Maria", "maria@example.com", "12345678900")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customer.ID != "cus_cpf123" {
		t.Errorf("expected ID cus_cpf123, got %s", customer.ID)
	}
}

func TestCreateSubscriptionWithCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body CreateSubscriptionReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.ExternalReference != "biz_123" {
			t.Errorf("expected externalReference biz_123, got %s", body.ExternalReference)
		}
		if body.Callback == nil {
			t.Fatal("expected callback, got nil")
		}
		if body.Callback.SuccessURL != "https://app.rekan.com.br/sucesso" {
			t.Errorf("unexpected successUrl: %s", body.Callback.SuccessURL)
		}
		if !body.Callback.AutoRedirect {
			t.Error("expected autoRedirect true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":          "sub_cb456",
			"paymentLink": "https://pay.asaas.com/sub_cb456",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	sub, err := c.CreateSubscription(context.Background(), CreateSubscriptionReq{
		Customer:          "cus_test",
		BillingType:       "PIX",
		Value:             19.00,
		NextDueDate:       "2026-03-01",
		Cycle:             "MONTHLY",
		Description:       "Rekan",
		ExternalReference: "biz_123",
		Callback: &Callback{
			SuccessURL:   "https://app.rekan.com.br/sucesso",
			AutoRedirect: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID != "sub_cb456" {
		t.Errorf("expected ID sub_cb456, got %s", sub.ID)
	}
}

func TestGetSubscription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/subscriptions/sub_get123" {
			t.Errorf("expected /subscriptions/sub_get123, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":          "sub_get123",
			"paymentLink": "https://pay.asaas.com/sub_get123",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	sub, err := c.GetSubscription(context.Background(), "sub_get123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID != "sub_get123" {
		t.Errorf("expected ID sub_get123, got %s", sub.ID)
	}
	if sub.PaymentLink != "https://pay.asaas.com/sub_get123" {
		t.Errorf("unexpected payment link: %s", sub.PaymentLink)
	}
}

func TestCancelSubscription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/subscriptions/sub_del123" {
			t.Errorf("expected /subscriptions/sub_del123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	err := c.CancelSubscription(context.Background(), "sub_del123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
