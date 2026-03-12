package service_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/denisraison/rekan/api/internal/domain"
	_ "github.com/denisraison/rekan/api/migrations"
)

// newTestApp creates a test PocketBase app with a user and business.
// Returns (app, userID, businessID).
func newTestApp(t testing.TB) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	users, err := app.FindCollectionByNameOrId(domain.CollUsers)
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	user := core.NewRecord(users)
	user.SetEmail("service-test@rekan.com.br")
	user.SetPassword("testpassword123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("save test user: %v", err)
	}

	businesses, err := app.FindCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		t.Fatalf("find businesses collection: %v", err)
	}
	biz := core.NewRecord(businesses)
	biz.Set("name", "Padaria Teste")
	biz.Set("type", "padaria")
	biz.Set("city", "São Paulo")
	biz.Set("state", "SP")
	biz.Set("target_audience", "moradores do bairro")
	biz.Set("brand_vibe", "acolhedora")
	biz.Set("services", []map[string]any{{"name": "Pão francês", "price_brl": 0.75}})
	if err := app.Save(biz); err != nil {
		t.Fatalf("save test business: %v", err)
	}

	return app, user.Id, biz.Id
}

// newInviteTestApp creates a test app with a business that has invite-related fields set.
func newInviteTestApp(t testing.TB) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	users, err := app.FindCollectionByNameOrId(domain.CollUsers)
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	user := core.NewRecord(users)
	user.SetEmail("invite-svc-test@rekan.com.br")
	user.SetPassword("testpassword123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	businesses, err := app.FindCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		t.Fatalf("find businesses collection: %v", err)
	}
	biz := core.NewRecord(businesses)
	biz.Set("name", "Padaria Convite")
	biz.Set("type", "padaria")
	biz.Set("city", "São Paulo")
	biz.Set("state", "SP")
	biz.Set("phone", "5511999998888")
	biz.Set("client_name", "Maria Silva")
	biz.Set("client_email", "maria@example.com")
	biz.Set("invite_status", "draft")
	biz.Set("tier", "parceiro")
	biz.Set("commitment", "mensal")
	if err := app.Save(biz); err != nil {
		t.Fatalf("save business: %v", err)
	}

	return app, user.Id, biz.Id
}
