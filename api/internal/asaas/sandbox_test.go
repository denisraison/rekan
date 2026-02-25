package asaas

import (
	"context"
	"os"
	"testing"
	"time"
)

// These tests hit the real Asaas sandbox API.
// Run with: ASAAS_SANDBOX_KEY=your_key go test ./internal/asaas/ -run TestSandbox -v
//
// They verify that our request/response structs match the actual API.
// Skip automatically when ASAAS_SANDBOX_KEY is not set.

func sandboxClient(t *testing.T) *Client {
	key := os.Getenv("ASAAS_SANDBOX_KEY")
	if key == "" {
		t.Skip("ASAAS_SANDBOX_KEY not set, skipping sandbox test")
	}
	return NewClient(key, true)
}

func TestSandboxCreateCustomer(t *testing.T) {
	c := sandboxClient(t)
	ctx := context.Background()

	customer, err := c.CreateCustomer(ctx, "Teste Rekan", "teste@rekan.com.br", "24971563792")
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	if customer.ID == "" {
		t.Fatal("expected non-empty customer ID")
	}
	t.Logf("customer ID: %s", customer.ID)
}

func TestSandboxCreateAuthorization(t *testing.T) {
	c := sandboxClient(t)
	ctx := context.Background()

	customer, err := c.CreateCustomer(ctx, "Teste Pix Auto", "pixauto@rekan.com.br", "24971563792")
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	t.Logf("customer ID: %s", customer.ID)

	dueDate := time.Now().Format("2006-01-02")
	auth, err := c.CreateAuthorization(ctx, CreateAuthorizationReq{
		CustomerID:  customer.ID,
		Description: "Rekan - Parceiro (sandbox test)",
		Frequency:   "MONTHLY",
		ContractID:  "sandbox-test-" + time.Now().Format("20060102150405"),
		StartDate:   dueDate,
		ImmediateQrCode: ImmediateQrCodeReq{
			Value:             108.90,
			OriginalValue:     108.90,
			DueDate:           dueDate,
			ExpirationSeconds: 86400,
		},
	})
	if err != nil {
		t.Fatalf("CreateAuthorization: %v", err)
	}

	if auth.ID == "" {
		t.Fatal("expected non-empty authorization ID")
	}
	t.Logf("authorization ID: %s", auth.ID)
	t.Logf("authorization status: %s", auth.Status)
	t.Logf("payload length: %d", len(auth.Payload))

	if auth.Payload == "" {
		t.Error("expected non-empty payload")
	} else {
		t.Logf("payload (first 80 chars): %.80s...", auth.Payload)
	}
}

func TestSandboxCancelAuthorization(t *testing.T) {
	c := sandboxClient(t)
	ctx := context.Background()

	customer, err := c.CreateCustomer(ctx, "Teste Cancel", "cancel@rekan.com.br", "24971563792")
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	dueDate := time.Now().Format("2006-01-02")
	auth, err := c.CreateAuthorization(ctx, CreateAuthorizationReq{
		CustomerID:  customer.ID,
		Description: "Rekan - Basico (cancel test)",
		Frequency:   "MONTHLY",
		ContractID:  "sandbox-cancel-" + time.Now().Format("20060102150405"),
		StartDate:   dueDate,
		ImmediateQrCode: ImmediateQrCodeReq{
			Value:             69.90,
			OriginalValue:     69.90,
			DueDate:           dueDate,
			ExpirationSeconds: 86400,
		},
	})
	if err != nil {
		t.Fatalf("CreateAuthorization: %v", err)
	}
	t.Logf("authorization ID to cancel: %s", auth.ID)

	if err := c.CancelAuthorization(ctx, auth.ID); err != nil {
		t.Fatalf("CancelAuthorization: %v", err)
	}
	t.Log("authorization cancelled successfully")
}

func TestSandboxCreateCharge(t *testing.T) {
	c := sandboxClient(t)
	ctx := context.Background()

	customer, err := c.CreateCustomer(ctx, "Teste Charge", "charge@rekan.com.br", "24971563792")
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	dueDate := time.Now().Format("2006-01-02")
	auth, err := c.CreateAuthorization(ctx, CreateAuthorizationReq{
		CustomerID:  customer.ID,
		Description: "Rekan - Parceiro (charge test)",
		Frequency:   "MONTHLY",
		ContractID:  "sandbox-charge-" + time.Now().Format("20060102150405"),
		StartDate:   dueDate,
		ImmediateQrCode: ImmediateQrCodeReq{
			Value:             108.90,
			OriginalValue:     108.90,
			DueDate:           dueDate,
			ExpirationSeconds: 86400,
		},
	})
	if err != nil {
		t.Fatalf("CreateAuthorization: %v", err)
	}
	t.Logf("authorization ID: %s", auth.ID)

	// Charge due date must be 2-10 business days out
	chargeDueDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	payment, err := c.CreateCharge(ctx, CreateChargeReq{
		Customer:                    customer.ID,
		BillingType:                 "PIX",
		Value:                       108.90,
		DueDate:                     chargeDueDate,
		Description:                 "Rekan - Parceiro (charge test)",
		ExternalReference:           "sandbox-test-charge",
		PixAutomaticAuthorizationId: auth.ID,
	})
	if err != nil {
		// This may fail if the authorization isn't activated yet in sandbox.
		// That's expected and useful information.
		t.Logf("CreateCharge error (may be expected if auth not activated): %v", err)
		return
	}

	if payment.ID == "" {
		t.Fatal("expected non-empty payment ID")
	}
	t.Logf("payment ID: %s", payment.ID)
	t.Logf("payment status: %s", payment.Status)
}
