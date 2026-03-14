package service_test

import (
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	_ "github.com/denisraison/rekan/api/migrations"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   bool
	}{
		// BR with country code
		{"5511940699184", "5511940699184", false},
		{"551134567890", "551134567890", false},

		// BR without country code
		{"11940699184", "5511940699184", false},
		{"1134567890", "551134567890", false},

		// BR with formatting
		{"(11) 94069-9184", "5511940699184", false},
		{"+55 11 94069-9184", "5511940699184", false},

		// BR area code 61 (Brasília), should not be confused with AU
		{"61999887766", "5561999887766", false},

		// AU local mobile
		{"0466889216", "61466889216", false},
		{"0412 345 678", "61412345678", false},

		// AU international mobile
		{"61466889216", "61466889216", false},
		{"+61 466 889 216", "61466889216", false},

		// AU with stray zero (country code + local leading zero)
		{"610466889216", "61466889216", false},

		// Invalid
		{"", "", true},
		{"123", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := service.NormalizePhone(tt.input)
			if tt.err {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

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
