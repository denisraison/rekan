package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/domain"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newInviteApp creates a test app using real migrations, plus a test user and business.
// Returns (app, userID, businessID).
func newInviteApp(t testing.TB) (*tests.TestApp, string, string) {
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
	user.SetEmail("invite-test@rekan.com.br")
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


func TestInviteGetNotFound(t *testing.T) {
	app, _, _ := newInviteApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method:         http.MethodGet,
		URL:            "/api/invites/nonexistent-token",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{App: app})
		},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteGetSuccess(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById("businesses", bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "valid-token-abc")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	s := &tests.ApiScenario{
		Method:         http.MethodGet,
		URL:            "/api/invites/valid-token-abc",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{App: app})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"business_name"`, `"client_name"`, `"tier"`, `"commitment"`, `"price"`},
	}
	s.Test(t)
}

func TestInviteGetAcceptedReturnsQrPayload(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById("businesses", bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_token", "accepted-token")
	biz.Set("invite_status", "accepted")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	biz.Set("qr_payload", "00020126580014br.gov.bcb.pix")
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	s := &tests.ApiScenario{
		Method:         http.MethodGet,
		URL:            "/api/invites/accepted-token",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{App: app})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"qr_payload"`, `"00020126580014br.gov.bcb.pix"`},
	}
	s.Test(t)
}

func TestInviteAcceptSuccess(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById("businesses", bizID)
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

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/accept-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				Asaas:  asaas.NewTestClient(mockAsaas.URL, "test-key"),
				AppURL: "https://app.rekan.com.br",
			})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"qr_payload"`},
	}
	s.Test(t)
}

func TestAuthorizationCancelNoActiveAuth(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method:         http.MethodPost,
		URL:            "/api/businesses/" + bizID + "/authorization:cancel",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestAuthorizationCancelSuccess(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, err := app.FindRecordById("businesses", bizID)
	if err != nil {
		t.Fatal(err)
	}
	biz.Set("invite_status", "active")
	biz.Set("authorization_id", "auth_to_cancel")
	if err := app.Save(biz); err != nil {
		t.Fatal(err)
	}

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockAsaas.Close()

	s := &tests.ApiScenario{
		Method:         http.MethodPost,
		URL:            "/api/businesses/" + bizID + "/authorization:cancel",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient(mockAsaas.URL, "test-key"),
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}
