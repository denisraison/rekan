package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	_ "github.com/denisraison/rekan/api/migrations"
)

func TestSendInviteNoPhone(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("phone", "")
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	_, err = service.SendInvite(context.Background(), app, nil, bizID, "https://app.rekan.com.br")
	if err == nil {
		t.Fatal("expected error for missing phone")
	}
}

func TestSendInviteMissingTierCommitment(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("tier", "")
	biz.Set("commitment", "")
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	_, err = service.SendInvite(context.Background(), app, nil, bizID, "https://app.rekan.com.br")
	if err == nil {
		t.Fatal("expected error for missing tier/commitment")
	}
}

func TestSendInviteRejectsActiveOrAccepted(t *testing.T) {
	for _, status := range []string{"active", "accepted"} {
		t.Run(status, func(t *testing.T) {
			app, _, bizID := newInviteTestApp(t)
			defer app.Cleanup()

			biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
			if err != nil {
				t.Fatal(err)
			}
			biz.Set("invite_status", status)
			if err := app.Save(biz); err != nil {
				t.Fatal(err)
			}

			_, err = service.SendInvite(context.Background(), app, nil, bizID, "https://app.rekan.com.br")
			if err == nil {
				t.Fatalf("expected error for status %q", status)
			}
		})
	}
}

func TestAcceptInvite(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "accept-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/customers"):
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "cus_test"})
		case strings.Contains(r.URL.Path, "/pix/automatic/authorizations"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "auth_accept_test",
				"status":  "CREATED",
				"payload": "00020126580014br.gov.bcb.pix0136test-payload",
			})
		}
	}))
	defer mockAsaas.Close()

	qrPayload, err := service.AcceptInvite(context.Background(), app, asaas.NewTestClient(mockAsaas.URL, "test-key"), "accept-token", "12345678900")
	if err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}
	if qrPayload == "" {
		t.Error("qr_payload should not be empty")
	}

	// Verify DB state
	biz, err = app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	if biz.GetString("invite_status") != "accepted" {
		t.Errorf("invite_status: got %q, want accepted", biz.GetString("invite_status"))
	}
	if biz.GetString("authorization_id") != "auth_accept_test" {
		t.Errorf("authorization_id: got %q, want auth_accept_test", biz.GetString("authorization_id"))
	}
	if biz.GetString("customer_id") != "cus_test" {
		t.Errorf("customer_id: got %q, want cus_test", biz.GetString("customer_id"))
	}
}

func TestAcceptInviteIdempotent(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "idempotent-token")
	biz.Set("invite_status", "accepted")
	biz.Set("authorization_id", "auth_existing")
	biz.Set("qr_payload", "existing-pix-payload")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	qrPayload, err := service.AcceptInvite(context.Background(), app, asaas.NewTestClient("http://unused", "test-key"), "idempotent-token", "12345678900")
	if err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}
	if qrPayload != "existing-pix-payload" {
		t.Errorf("qr_payload: got %q, want %q", qrPayload, "existing-pix-payload")
	}
}

func TestAcceptInviteActiveConflict(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "active-token")
	biz.Set("invite_status", "active")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	_, err = service.AcceptInvite(context.Background(), app, asaas.NewTestClient("http://unused", "key"), "active-token", "12345678900")
	if err == nil {
		t.Fatal("expected conflict error for active status")
	}
}

func TestAcceptInviteWrongStatus(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "draft-token")
	biz.Set("invite_status", "draft")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	_, err = service.AcceptInvite(context.Background(), app, asaas.NewTestClient("http://unused", "key"), "draft-token", "12345678900")
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
}

func TestAcceptInviteExpired(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "expired-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().Add(-8*24*time.Hour).UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	_, err = service.AcceptInvite(context.Background(), app, asaas.NewTestClient("http://unused", "key"), "expired-token", "12345678900")
	if err == nil {
		t.Fatal("expected error for expired invite")
	}
}

func TestCancelAuthorization(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_status", "active")
	biz.Set("authorization_id", "auth_to_cancel")
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	var deletedPath string
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedPath = r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockAsaas.Close()

	err = service.CancelAuthorization(context.Background(), app, asaas.NewTestClient(mockAsaas.URL, "test-key"), bizID)
	if err != nil {
		t.Fatalf("CancelAuthorization: %v", err)
	}

	if deletedPath != "/pix/automatic/authorizations/auth_to_cancel" {
		t.Errorf("expected DELETE to /pix/automatic/authorizations/auth_to_cancel, got %s", deletedPath)
	}

	// Verify DB state
	biz, err = app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	if biz.GetString("invite_status") != "cancelled" {
		t.Errorf("invite_status: got %q, want cancelled", biz.GetString("invite_status"))
	}
}
