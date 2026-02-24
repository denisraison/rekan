package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/asaas"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newInviteApp creates a test app with businesses, messages, a user, and a business.
// Returns (app, userID, businessID).
func newInviteApp(t testing.TB) (*tests.TestApp, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	businesses := core.NewBaseCollection("businesses")
	businesses.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "type"},
		&core.TextField{Name: "city"},
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
		&core.JSONField{Name: "services"},
		&core.TextField{Name: "target_audience"},
		&core.TextField{Name: "brand_vibe"},
		&core.TextField{Name: "quirks"},
	)
	if err := app.Save(businesses); err != nil {
		t.Fatalf("save businesses collection: %v", err)
	}

	messages := core.NewBaseCollection("messages")
	messages.Fields.Add(
		&core.TextField{Name: "business"},
		&core.TextField{Name: "phone"},
		&core.SelectField{Name: "type", Values: []string{"text", "audio", "image"}, MaxSelect: 1},
		&core.TextField{Name: "content"},
		&core.SelectField{Name: "direction", Values: []string{"incoming", "outgoing"}, MaxSelect: 1},
		&core.DateField{Name: "wa_timestamp"},
		&core.TextField{Name: "wa_message_id"},
	)
	if err := app.Save(messages); err != nil {
		t.Fatalf("save messages collection: %v", err)
	}

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	user := core.NewRecord(users)
	user.SetEmail("invite-test@rekan.com.br")
	user.SetPassword("testpassword123!")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	biz := core.NewRecord(businesses)
	biz.Set("name", "Padaria Convite")
	biz.Set("type", "padaria")
	biz.Set("city", "São Paulo")
	biz.Set("user", user.Id)
	biz.Set("phone", "5511999998888")
	biz.Set("client_name", "Maria Silva")
	biz.Set("client_email", "maria@example.com")
	biz.Set("invite_status", "draft")
	if err := app.Save(biz); err != nil {
		t.Fatalf("save business: %v", err)
	}

	return app, user.Id, biz.Id
}

func inviteAuthHeader(app *tests.TestApp, userID string) string {
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		panic("find user: " + err.Error())
	}
	token, err := user.NewAuthToken()
	if err != nil {
		panic("new auth token: " + err.Error())
	}
	return token
}

func TestInviteSendRequiresPhone(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	// Remove phone from business
	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("phone", "")
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/invites:send",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				AppURL: "https://app.rekan.com.br",
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, userID),
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteSendRejectsActive(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_status", "active")
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/invites:send",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				AppURL: "https://app.rekan.com.br",
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, userID),
		},
		ExpectedStatus:  http.StatusConflict,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteSendRejectsAccepted(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_status", "accepted")
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/invites:send",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				AppURL: "https://app.rekan.com.br",
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, userID),
		},
		ExpectedStatus:  http.StatusConflict,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteGetNotFound(t *testing.T) {
	app, _, _ := newInviteApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodGet,
		URL:    "/api/invites/nonexistent-token",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "valid-token-abc")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodGet,
		URL:    "/api/invites/valid-token-abc",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{App: app})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"business_name"`, `"client_name"`, `"price_first_month"`, `"price_monthly"`},
	}
	s.Test(t)
}

func TestInviteGetExpired(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "expired-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().Add(-8*24*time.Hour).UTC().Format(time.RFC3339))
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodGet,
		URL:    "/api/invites/expired-token",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{App: app})
		},
		ExpectedStatus:  http.StatusGone,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteAcceptSuccess(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "accept-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	app.Save(biz)

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/customers"):
			json.NewEncoder(w).Encode(map[string]string{"id": "cus_test"})
		case strings.Contains(r.URL.Path, "/subscriptions"):
			json.NewEncoder(w).Encode(map[string]string{
				"id":          "sub_accept_test",
				"paymentLink": "https://pay.example.com/test",
			})
		}
	}))
	defer mockAsaas.Close()

	var savedApp *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/accept-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			savedApp = app
			return app
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				Asaas:  asaas.NewTestClient(mockAsaas.URL, "test-key"),
				AppURL: "https://app.rekan.com.br",
			})
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			biz, err := app.FindRecordById("businesses", bizID)
			if err != nil {
				t.Fatalf("find business: %v", err)
			}
			if biz.GetString("invite_status") != "accepted" {
				t.Errorf("invite_status: got %q, want accepted", biz.GetString("invite_status"))
			}
			if biz.GetString("subscription_id") != "sub_accept_test" {
				t.Errorf("subscription_id: got %q, want sub_accept_test", biz.GetString("subscription_id"))
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"payment_url"`},
	}
	s.Test(t)
	if savedApp != nil {
		savedApp.Cleanup()
	}
}

func TestInviteAcceptIdempotent(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "idempotent-token")
	biz.Set("invite_status", "accepted")
	biz.Set("subscription_id", "sub_existing")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	app.Save(biz)

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":          "sub_existing",
			"paymentLink": "https://pay.example.com/existing",
		})
	}))
	defer mockAsaas.Close()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/idempotent-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:    app,
				Asaas:  asaas.NewTestClient(mockAsaas.URL, "test-key"),
				AppURL: "https://app.rekan.com.br",
			})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"payment_url"`, `existing`},
	}
	s.Test(t)
}

func TestInviteAcceptActiveConflict(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "active-token")
	biz.Set("invite_status", "active")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/active-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		ExpectedStatus:  http.StatusConflict,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteAcceptWrongStatus(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	// Status is "draft", not "invited"
	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "draft-token")
	biz.Set("invite_status", "draft")
	biz.Set("invite_sent_at", time.Now().UTC().Format(time.RFC3339))
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/draft-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestInviteAcceptExpired(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_token", "expired-accept-token")
	biz.Set("invite_status", "invited")
	biz.Set("invite_sent_at", time.Now().Add(-8*24*time.Hour).UTC().Format(time.RFC3339))
	app.Save(biz)

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/invites/expired-accept-token/accept",
		Body:   strings.NewReader(`{"cpf_cnpj":"12345678900"}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		ExpectedStatus:  http.StatusGone,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestSubscriptionCancelNoActiveSub(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	// Status is "draft", no subscription_id
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/subscription:cancel",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, userID),
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestSubscriptionCancelForbidden(t *testing.T) {
	app, _, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_status", "active")
	biz.Set("subscription_id", "sub_xyz")
	app.Save(biz)

	// Create a different user
	users, _ := app.FindCollectionByNameOrId("users")
	other := core.NewRecord(users)
	other.SetEmail("other-cancel@rekan.com.br")
	other.SetPassword("testpassword123!")
	if err := app.Save(other); err != nil {
		t.Fatalf("save other user: %v", err)
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/subscription:cancel",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, other.Id),
		},
		ExpectedStatus:  http.StatusForbidden,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestSubscriptionCancelSuccess(t *testing.T) {
	app, userID, bizID := newInviteApp(t)
	defer app.Cleanup()

	biz, _ := app.FindRecordById("businesses", bizID)
	biz.Set("invite_status", "active")
	biz.Set("subscription_id", "sub_to_cancel")
	app.Save(biz)

	var deletedPath string
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedPath = r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockAsaas.Close()

	var savedApp *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/subscription:cancel",
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			savedApp = app
			return app
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient(mockAsaas.URL, "test-key"),
			})
		},
		Headers: map[string]string{
			"Authorization": inviteAuthHeader(app, userID),
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if deletedPath != "/subscriptions/sub_to_cancel" {
				t.Errorf("expected DELETE to /subscriptions/sub_to_cancel, got %s", deletedPath)
			}
			biz, _ := app.FindRecordById("businesses", bizID)
			if biz.GetString("invite_status") != "cancelled" {
				t.Errorf("invite_status: got %q, want cancelled", biz.GetString("invite_status"))
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message"`},
	}
	s.Test(t)
	if savedApp != nil {
		savedApp.Cleanup()
	}
}
