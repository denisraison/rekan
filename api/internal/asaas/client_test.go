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

func TestCreateAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/pix/automatic/authorizations" {
			t.Errorf("expected /pix/automatic/authorizations, got %s", r.URL.Path)
		}

		var body CreateAuthorizationReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.CustomerID != "cus_test123" {
			t.Errorf("expected customerId cus_test123, got %s", body.CustomerID)
		}
		if body.Frequency != "MONTHLY" {
			t.Errorf("expected frequency MONTHLY, got %s", body.Frequency)
		}
		if body.ContractID != "biz_123" {
			t.Errorf("expected contractId biz_123, got %s", body.ContractID)
		}
		if body.ImmediateQrCode.Value != 108.90 {
			t.Errorf("expected immediateQrCode.value 108.90, got %f", body.ImmediateQrCode.Value)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "auth_test456",
			"status":  "CREATED",
			"payload": "00020126580014br.gov.bcb.pix0136test-payload-string",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	auth, err := c.CreateAuthorization(context.Background(), CreateAuthorizationReq{
		CustomerID:  "cus_test123",
		Description: "Rekan - parceiro",
		Frequency:   "MONTHLY",
		ContractID:  "biz_123",
		StartDate:   "2026-02-24",
		ImmediateQrCode: ImmediateQrCodeReq{
			Value:             108.90,
			OriginalValue:     108.90,
			DueDate:           "2026-02-24",
			ExpirationSeconds: 86400,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.ID != "auth_test456" {
		t.Errorf("expected ID auth_test456, got %s", auth.ID)
	}
	if auth.Payload != "00020126580014br.gov.bcb.pix0136test-payload-string" {
		t.Errorf("unexpected payload: %s", auth.Payload)
	}
}

func TestCancelAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/pix/automatic/authorizations/auth_del123" {
			t.Errorf("expected /pix/automatic/authorizations/auth_del123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	err := c.CancelAuthorization(context.Background(), "auth_del123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCharge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/payments" {
			t.Errorf("expected /payments, got %s", r.URL.Path)
		}

		var body CreateChargeReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Customer != "cus_test123" {
			t.Errorf("expected customer cus_test123, got %s", body.Customer)
		}
		if body.BillingType != "PIX" {
			t.Errorf("expected PIX, got %s", body.BillingType)
		}
		if body.PixAutomaticAuthorizationId != "auth_test456" {
			t.Errorf("expected auth_test456, got %s", body.PixAutomaticAuthorizationId)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "pay_test789",
			"status": "PENDING",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, apiKey: "test-key", http: &http.Client{}}
	payment, err := c.CreateCharge(context.Background(), CreateChargeReq{
		Customer:                    "cus_test123",
		BillingType:                 "PIX",
		Value:                       108.90,
		DueDate:                     "2026-03-24",
		Description:                 "Rekan - parceiro",
		ExternalReference:           "biz_123",
		PixAutomaticAuthorizationId: "auth_test456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.ID != "pay_test789" {
		t.Errorf("expected ID pay_test789, got %s", payment.ID)
	}
}

func TestErrorWithDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
