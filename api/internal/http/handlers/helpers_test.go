package handlers_test

import (
	"context"
	"testing"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	_ "github.com/denisraison/rekan/api/migrations"
)

const (
	testUserEmail    = "handler-test@rekan.com.br"
	testUserPassword = "testpassword123!"
)

// stubGenerate returns a fixed set of posts for testing.
func stubGenerate(_ context.Context, _ content.BusinessProfile, _ []content.Role, _ []string) ([]content.Post, error) {
	return []content.Post{
		{
			Caption:        "Legenda de teste",
			Hashtags:       []string{"#teste", "#rekan"},
			ProductionNote: "Nota de teste",
		},
	}, nil
}

// stubGenerateFromMessage returns a fixed single post for testing.
func stubGenerateFromMessage(_ context.Context, _ content.BusinessProfile, _ string, _ []string) (content.Post, error) {
	return content.Post{
		Caption:        "Legenda do operador",
		Hashtags:       []string{"#operador"},
		ProductionNote: "Nota operador",
	}, nil
}

// newHandlerApp creates a test PocketBase app using real migrations, plus a test user and business.
func newHandlerApp(t testing.TB) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

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

	businesses, err := app.FindCollectionByNameOrId("businesses")
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

	return app, testUser.Id, biz.Id
}

// registerHandlerRoutes registers all routes with stub generate functions.
func registerHandlerRoutes(_ *tests.TestApp, e *core.ServeEvent, deps handlers.Deps) {
	apphttp.RegisterRoutes(e.Router, deps)
}

// authHeader builds a direct auth token for the given user ID.
func authHeader(app *tests.TestApp, userID string) string {
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		panic("find user for auth: " + err.Error())
	}
	token, err := user.NewAuthToken()
	if err != nil {
		panic("new auth token: " + err.Error())
	}
	return token
}
