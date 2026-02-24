package handlers_test

import (
	"context"
	"testing"

	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
)

const (
	testUserEmail    = "handler-test@rekan.com.br"
	testUserPassword = "testpassword123!"
)

// stubGenerate returns a fixed set of posts for testing.
func stubGenerate(_ context.Context, _ eval.BusinessProfile, _ []eval.Role, _ []string) ([]eval.Post, error) {
	return []eval.Post{
		{
			Caption:        "Legenda de teste",
			Hashtags:       []string{"#teste", "#rekan"},
			ProductionNote: "Nota de teste",
		},
	}, nil
}

// stubGenerateFromMessage returns a fixed single post for testing.
func stubGenerateFromMessage(_ context.Context, _ eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
	return eval.Post{
		Caption:        "Legenda do operador",
		Hashtags:       []string{"#operador"},
		ProductionNote: "Nota operador",
	}, nil
}

// newHandlerApp creates a test PocketBase app with users,
// businesses (including invite fields), and posts collections, plus a test user and business.
func newHandlerApp(t testing.TB) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	// Create businesses collection
	businesses := core.NewBaseCollection("businesses")
	businesses.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "type"},
		&core.TextField{Name: "city"},
		&core.TextField{Name: "target_audience"},
		&core.TextField{Name: "brand_vibe"},
		&core.TextField{Name: "quirks"},
		&core.JSONField{Name: "services"},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "phone"},
		&core.TextField{Name: "client_name"},
		&core.TextField{Name: "client_email"},
		&core.TextField{Name: "invite_token"},
		&core.SelectField{
			Name:      "invite_status",
			Values:    []string{"draft", "invited", "accepted", "active", "payment_failed", "cancelled"},
			MaxSelect: 1,
		},
		&core.DateField{Name: "invite_sent_at"},
		&core.TextField{Name: "subscription_id"},
		&core.DateField{Name: "terms_accepted_at"},
	)
	if err := app.Save(businesses); err != nil {
		t.Fatalf("save businesses collection: %v", err)
	}

	// Create posts collection
	posts := core.NewBaseCollection("posts")
	posts.Fields.Add(
		&core.TextField{Name: "business"},
		&core.TextField{Name: "caption"},
		&core.JSONField{Name: "hashtags"},
		&core.TextField{Name: "production_note"},
		&core.BoolField{Name: "edited"},
		&core.TextField{Name: "batch_id"},
		&core.TextField{Name: "role"},
		&core.TextField{Name: "hook"},
		&core.TextField{Name: "source"},
		&core.TextField{Name: "message"},
	)
	if err := app.Save(posts); err != nil {
		t.Fatalf("save posts collection: %v", err)
	}

	// Create test user
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	testUser := core.NewRecord(users)
	testUser.SetEmail(testUserEmail)
	testUser.SetPassword(testUserPassword)
	if err := app.Save(testUser); err != nil {
		t.Fatalf("save test user: %v", err)
	}

	// Create test business owned by the test user
	biz := core.NewRecord(businesses)
	biz.Set("name", "Padaria Teste")
	biz.Set("type", "padaria")
	biz.Set("city", "São Paulo")
	biz.Set("target_audience", "moradores do bairro")
	biz.Set("brand_vibe", "acolhedora")
	biz.Set("services", []map[string]any{{"name": "Pão francês", "price_brl": 0.75}})
	biz.Set("user", testUser.Id)
	if err := app.Save(biz); err != nil {
		t.Fatalf("save test business: %v", err)
	}

	return app, testUser.Id, biz.Id
}

// registerHandlerRoutes registers all routes with stub generate functions.
func registerHandlerRoutes(app *tests.TestApp, e *core.ServeEvent, deps handlers.Deps) {
	apphttp.RegisterRoutes(e.Router, deps)
}
