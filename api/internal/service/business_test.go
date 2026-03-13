package service_test

import (
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	_ "github.com/denisraison/rekan/api/migrations"
)

func TestGetInviteInfo(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "info-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	info, err := service.GetInviteInfo(app, "info-token")
	if err != nil {
		t.Fatalf("GetInviteInfo: %v", err)
	}
	if info.BusinessName != "Padaria Convite" {
		t.Errorf("business_name: got %q, want %q", info.BusinessName, "Padaria Convite")
	}
	if info.ClientName != "Maria Silva" {
		t.Errorf("client_name: got %q, want %q", info.ClientName, "Maria Silva")
	}
	if info.Tier != "parceiro" {
		t.Errorf("tier: got %q, want %q", info.Tier, "parceiro")
	}
}

func TestGetInviteInfoExpired(t *testing.T) {
	app, _, bizID := newInviteTestApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById(domain.CollBusinesses, bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "expired-info-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().Add(-8*24*time.Hour).UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	info, err := service.GetInviteInfo(app, "expired-info-token")
	if err != nil {
		t.Fatalf("GetInviteInfo: %v", err)
	}

	// SentAt should enable the handler to check expiry
	if time.Since(info.SentAt) < 7*24*time.Hour {
		t.Error("SentAt should be older than 7 days for expired invite")
	}
}
