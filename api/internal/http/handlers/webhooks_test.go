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
const testSubscriptionID = "sub_test_webhook_123"

// newWebhookApp creates a test PocketBase app with a businesses collection
// (including invite fields) and a pre-created test business with subscription_id.
func newWebhookApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	businesses := core.NewBaseCollection("businesses")
	businesses.Fields.Add(
		&core.TextField{Name: "name"},
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

	// Create a test business with a known subscription_id
	biz := core.NewRecord(businesses)
	biz.Set("name", "Webhook Test Biz")
	biz.Set("subscription_id", testSubscriptionID)
	biz.Set("invite_status", "accepted")
	if err := app.Save(biz); err != nil {
		t.Fatalf("save test business: %v", err)
	}

	return app
}

// webhookBody builds a minimal Asaas webhook payload.
// viaSubscription=true uses the "subscription" key; false uses "payment".
func webhookBody(event, subscriptionID string, viaSubscription bool) string {
	if viaSubscription {
		return fmt.Sprintf(`{"event":%q,"subscription":{"id":%q}}`, event, subscriptionID)
	}
	return fmt.Sprintf(`{"event":%q,"payment":{"subscription":%q}}`, event, subscriptionID)
}

// assertBusinessStatus finds the test business by subscription_id and verifies its invite_status.
func assertBusinessStatus(t testing.TB, app *tests.TestApp, want string) {
	t.Helper()
	records, err := app.FindAllRecords("businesses", nil)
	if err != nil {
		t.Fatalf("find businesses: %v", err)
	}
	for _, r := range records {
		if r.GetString("subscription_id") == testSubscriptionID {
			got := r.GetString("invite_status")
			if got != want {
				t.Errorf("invite_status: got %q, want %q", got, want)
			}
			return
		}
	}
	t.Errorf("test business with subscription_id %q not found", testSubscriptionID)
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
		Body:   strings.NewReader(webhookBody("PAYMENT_CONFIRMED", testSubscriptionID, false)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": "wrong-token",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			return newWebhookApp(tb)
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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
		Body:   strings.NewReader(webhookBody("UNKNOWN_EVENT", testSubscriptionID, false)),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			return newWebhookApp(tb)
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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
		Body:   strings.NewReader(webhookBody("SUBSCRIPTION_RENEWED", testSubscriptionID, false)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			return newWebhookApp(tb)
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(app, e)
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message":"ok"`},
	}
	s.Test(t)
}

func TestWebhookPaymentConfirmed(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBody("PAYMENT_CONFIRMED", testSubscriptionID, false)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(t testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			assertBusinessStatus(t, app, "active")
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

func TestWebhookPaymentOverdue(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBody("PAYMENT_OVERDUE", testSubscriptionID, false)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(t testing.TB, a *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(a, e)
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			// PAYMENT_OVERDUE is now a no-op, status should stay as "accepted"
			assertBusinessStatus(t, app, "accepted")
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

func TestWebhookSubscriptionDeleted(t *testing.T) {
	var app *tests.TestApp
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBody("SUBSCRIPTION_DELETED", testSubscriptionID, true)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app = newWebhookApp(tb)
			return app
		},
		BeforeTestFunc: func(t testing.TB, a *tests.TestApp, e *core.ServeEvent) {
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

func TestWebhookUnknownSubscriptionID(t *testing.T) {
	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/webhooks/asaas",
		Body:   strings.NewReader(webhookBody("PAYMENT_CONFIRMED", "sub_nonexistent", false)),
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"asaas-access-token": testWebhookToken,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			return newWebhookApp(tb)
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerRoutes(app, e)
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"message":"ok"`},
	}
	s.Test(t)
}

