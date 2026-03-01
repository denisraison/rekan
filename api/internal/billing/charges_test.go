package billing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newBillingApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	businesses := core.NewBaseCollection("businesses")
	businesses.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "user"},
		&core.SelectField{
			Name:      "invite_status",
			Values:    []string{"draft", "invited", "accepted", "active", "payment_failed", "cancelled"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "authorization_id"},
		&core.TextField{Name: "customer_id"},
		&core.SelectField{
			Name:      "tier",
			Values:    []string{"basico", "parceiro", "profissional"},
			MaxSelect: 1,
		},
		&core.SelectField{
			Name:      "commitment",
			Values:    []string{"mensal", "trimestral"},
			MaxSelect: 1,
		},
		&core.DateField{Name: "next_charge_date"},
		&core.BoolField{Name: "charge_pending"},
	)
	if err := app.Save(businesses); err != nil {
		t.Fatalf("save businesses collection: %v", err)
	}

	return app
}

func createBusiness(t testing.TB, app *tests.TestApp, name string, status string, tier string, commitment string, nextChargeDate string, chargePending bool) *core.Record {
	t.Helper()
	businesses, err := app.FindCollectionByNameOrId("businesses")
	if err != nil {
		t.Fatalf("find businesses: %v", err)
	}
	biz := core.NewRecord(businesses)
	biz.Set("name", name)
	biz.Set("invite_status", status)
	biz.Set("authorization_id", "auth_"+name)
	biz.Set("customer_id", "cus_"+name)
	biz.Set("tier", tier)
	biz.Set("commitment", commitment)
	if nextChargeDate != "" {
		biz.Set("next_charge_date", nextChargeDate)
	}
	biz.Set("charge_pending", chargePending)
	if err := app.Save(biz); err != nil {
		t.Fatalf("save business %s: %v", name, err)
	}
	return biz
}

func TestCreatePendingCharges(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	// Business due in 3 days: should get a charge
	dueDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02 00:00:00.000Z")
	biz := createBusiness(t, app, "due_soon", "active", "parceiro", "mensal", dueDate, false)

	expectedDueDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02")

	var chargeCreated bool
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payments" && r.Method == http.MethodPost {
			chargeCreated = true
			var body asaas.CreateChargeReq
			json.NewDecoder(r.Body).Decode(&body)
			if body.Customer != "cus_due_soon" {
				t.Errorf("expected customer cus_due_soon, got %s", body.Customer)
			}
			if body.PixAutomaticAuthorizationId != "auth_due_soon" {
				t.Errorf("expected auth_due_soon, got %s", body.PixAutomaticAuthorizationId)
			}
			wantRef := biz.Id + "_" + expectedDueDate
			if body.ExternalReference != wantRef {
				t.Errorf("expected externalReference %s, got %s", wantRef, body.ExternalReference)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "pay_test", "status": "PENDING"})
	}))
	defer mockAsaas.Close()

	client := asaas.NewTestClient(mockAsaas.URL, "test-key")
	CreatePendingCharges(app, client)

	if !chargeCreated {
		t.Error("expected charge to be created for due_soon business")
	}

	// Verify charge_pending is now true
	updated, err := app.FindRecordById("businesses", biz.Id)
	if err != nil {
		t.Fatalf("find business: %v", err)
	}
	if !updated.GetBool("charge_pending") {
		t.Error("charge_pending should be true after charge creation")
	}
}

func TestCreatePendingChargesSkipsPending(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	// Business already has charge_pending=true: should be skipped
	dueDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02 00:00:00.000Z")
	createBusiness(t, app, "already_pending", "active", "parceiro", "mensal", dueDate, true)

	var chargeCreated bool
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payments" {
			chargeCreated = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "pay_test", "status": "PENDING"})
	}))
	defer mockAsaas.Close()

	client := asaas.NewTestClient(mockAsaas.URL, "test-key")
	CreatePendingCharges(app, client)

	if chargeCreated {
		t.Error("should not create charge for business with charge_pending=true")
	}
}

func TestCreatePendingChargesSkipsFarOut(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	// Business due in 15 days: too far out, should be skipped
	dueDate := time.Now().AddDate(0, 0, 15).Format("2006-01-02 00:00:00.000Z")
	createBusiness(t, app, "far_out", "active", "parceiro", "mensal", dueDate, false)

	var chargeCreated bool
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payments" {
			chargeCreated = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "pay_test", "status": "PENDING"})
	}))
	defer mockAsaas.Close()

	client := asaas.NewTestClient(mockAsaas.URL, "test-key")
	CreatePendingCharges(app, client)

	if chargeCreated {
		t.Error("should not create charge for business with next_charge_date > 7 days out")
	}
}

func TestCreatePendingChargesSkipsNonActive(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	// Business with status "cancelled" should be skipped
	dueDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02 00:00:00.000Z")
	createBusiness(t, app, "cancelled_biz", "cancelled", "parceiro", "mensal", dueDate, false)

	var chargeCreated bool
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payments" {
			chargeCreated = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "pay_test", "status": "PENDING"})
	}))
	defer mockAsaas.Close()

	client := asaas.NewTestClient(mockAsaas.URL, "test-key")
	CreatePendingCharges(app, client)

	if chargeCreated {
		t.Error("should not create charge for non-active business")
	}
}

func TestCreatePendingChargesRollbackOnFailure(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	dueDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02 00:00:00.000Z")
	biz := createBusiness(t, app, "fail_charge", "active", "parceiro", "mensal", dueDate, false)

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"description": "A autorização deve estar ativa"},
			},
		})
	}))
	defer mockAsaas.Close()

	client := asaas.NewTestClient(mockAsaas.URL, "test-key")
	CreatePendingCharges(app, client)

	updated, err := app.FindRecordById("businesses", biz.Id)
	if err != nil {
		t.Fatalf("find business: %v", err)
	}
	if updated.GetBool("charge_pending") {
		t.Error("charge_pending should be false after Asaas failure (rollback)")
	}
}

func TestCreatePendingChargesNilClient(t *testing.T) {
	app := newBillingApp(t)
	defer app.Cleanup()

	// Should not panic with nil client
	CreatePendingCharges(app, nil)
}
