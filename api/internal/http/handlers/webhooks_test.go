package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const testWebhookToken = "test-webhook-token"
const testAuthorizationID = "auth_test_webhook_123"

// newWebhookApp creates a test PocketBase app using real migrations
// and a pre-created test business with authorization_id.
func newWebhookApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	businesses, err := app.FindCollectionByNameOrId("businesses")
	if err != nil {
		t.Fatalf("find businesses collection: %v", err)
	}

	biz := core.NewRecord(businesses)
	biz.Set("name", "Webhook Test Biz")
	biz.Set("type", "padaria")
	biz.Set("city", "São Paulo")
	biz.Set("state", "SP")
	biz.Set("authorization_id", testAuthorizationID)
	biz.Set("customer_id", "cus_webhook_test")
	biz.Set("invite_status", "accepted")
	biz.Set("tier", "parceiro")
	biz.Set("commitment", "mensal")
	if err := app.Save(biz); err != nil {
		t.Fatalf("save test business: %v", err)
	}

	return app
}

// webhookBodyAuthorization builds a Pix Automatico authorization webhook payload.
func webhookBodyAuthorization(event, authID string) string {
	return fmt.Sprintf(`{"event":%q,"pixAutomaticAuthorization":{"id":%q}}`, event, authID)
}

// webhookBodyPayment builds a Pix Automatico payment webhook payload.
func webhookBodyPayment(event, authID string) string {
	return fmt.Sprintf(`{"event":%q,"payment":{"id":"pay_123","status":"CONFIRMED","pixAutomaticAuthorizationId":%q}}`, event, authID)
}

// assertBusinessStatus finds the test business by authorization_id and verifies its invite_status.
func assertBusinessStatus(t testing.TB, app *tests.TestApp, want string) {
	t.Helper()
	records, err := app.FindAllRecords("businesses", nil)
	if err != nil {
		t.Fatalf("find businesses: %v", err)
	}
	for _, r := range records {
		if r.GetString("authorization_id") == testAuthorizationID {
			got := r.GetString("invite_status")
			if got != want {
				t.Errorf("invite_status: got %q, want %q", got, want)
			}
			return
		}
	}
	t.Errorf("test business with authorization_id %q not found", testAuthorizationID)
}

func assertBusinessBool(t testing.TB, app *tests.TestApp, field string, want bool) {
	t.Helper()
	records, err := app.FindAllRecords("businesses", nil)
	if err != nil {
		t.Fatalf("find businesses: %v", err)
	}
	for _, r := range records {
		if r.GetString("authorization_id") == testAuthorizationID {
			got := r.GetBool(field)
			if got != want {
				t.Errorf("%s: got %v, want %v", field, got, want)
			}
			return
		}
	}
	t.Errorf("test business with authorization_id %q not found", testAuthorizationID)
}

func registerRoutes(app *tests.TestApp, e *core.ServeEvent) {
	apphttp.RegisterRoutes(e.Router, handlers.Deps{
		App:          app,
		WebhookToken: testWebhookToken,
	})
}

func TestWebhookInvalidToken(t *testing.T) {
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": "wrong-token",
		},
		TestAppFactory: newWebhookApp,
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(app, e)
		},
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"message":"unauthorized"`},
	}
	s.Test(t)
}

func TestWebhookNoTokenValidation(t *testing.T) {
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("UNKNOWN_EVENT", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: newWebhookApp,
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			apphttp.RegisterRoutes(e.Router, handlers.Deps{
				App:          app,
				WebhookToken: "",
			})
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message":"ok"`},
	}
	s.Test(t)
}

func TestWebhookUnknownEvent(t *testing.T) {
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("SOME_UNKNOWN_EVENT", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: newWebhookApp,
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(app, e)
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message":"ok"`},
	}
	s.Test(t)
}

func TestWebhookAuthorizationActivated(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "active")
			// next_charge_date should be set (non-empty)
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					ncd := r.GetString("next_charge_date")
					if ncd == "" {
						t.Error("next_charge_date should be set after activation")
					}
				}
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookAuthorizationRefused(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_REFUSED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "cancelled")
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookAuthorizationExpired(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_EXPIRED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "payment_failed")
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookPaymentConfirmed(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyPayment("PAYMENT_CONFIRMED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			// PAYMENT_CONFIRMED requires active status + charge_pending
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				tb.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					r.Set("invite_status", "active")
					r.Set("charge_pending", true)
					if err := app.Save(r); err != nil {
						tb.Fatal(err)
					}
				}
			}
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "active")
			assertBusinessBool(t, app, "charge_pending", false)
			// next_charge_date should be set
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					ncd := r.GetString("next_charge_date")
					if ncd == "" {
						t.Error("next_charge_date should be set after payment confirmed")
					}
				}
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookPaymentInstructionRefused(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyPayment("PIX_AUTOMATIC_RECURRING_PAYMENT_INSTRUCTION_REFUSED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				tb.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					r.Set("invite_status", "active")
					r.Set("charge_pending", true)
					if err := app.Save(r); err != nil {
						tb.Fatal(err)
					}
				}
			}
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "payment_failed")
			assertBusinessBool(t, app, "charge_pending", false)
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookAuthorizationActivatedIdempotent(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			// Already active with a next_charge_date set
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				tb.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					r.Set("invite_status", "active")
					r.Set("next_charge_date", "2026-04-01 00:00:00.000Z")
					if err := app.Save(r); err != nil {
						tb.Fatal(err)
					}
				}
			}
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "active")
			// next_charge_date must NOT have changed
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					ncd := r.GetString("next_charge_date")
					if !strings.Contains(ncd, "2026-04-01") {
						t.Errorf("next_charge_date changed on duplicate activation: got %s", ncd)
					}
				}
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookPaymentConfirmedNotPending(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyPayment("PAYMENT_CONFIRMED", testAuthorizationID)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			// Active but charge_pending=false (duplicate webhook)
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				tb.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					r.Set("invite_status", "active")
					r.Set("charge_pending", false)
					r.Set("next_charge_date", "2026-04-01 00:00:00.000Z")
					if err := app.Save(r); err != nil {
						tb.Fatal(err)
					}
				}
			}
			return app
		},
		BeforeTestFunc: func(_ testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "active")
			assertBusinessBool(t, app, "charge_pending", false)
			// next_charge_date must NOT have advanced
			records, err := app.FindAllRecords("businesses", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range records {
				if r.GetString("authorization_id") == testAuthorizationID {
					ncd := r.GetString("next_charge_date")
					if !strings.Contains(ncd, "2026-04-01") {
						t.Errorf("next_charge_date changed on duplicate payment confirmed: got %s", ncd)
					}
				}
			}
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"message":"ok"`},
	}
	s.Test(t)
	if app != nil {
		app.Cleanup()
	}
}

func TestWebhookUnknownAuthorizationID(t *testing.T) {
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBodyAuthorization("PIX_AUTOMATIC_RECURRING_AUTHORIZATION_ACTIVATED", "auth_nonexistent")),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: newWebhookApp,
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(app, e)
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message":"ok"`},
	}
	s.Test(t)
}
