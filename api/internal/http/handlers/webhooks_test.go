package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
)

const testWebhookToken = "test-webhook-token"
const testSubscriptionID = "sub_test_webhook_123"

// newWebhookApp creates a test PocketBase app with the users collection extended
// to include our subscription fields, and a pre-created test user.
func newWebhookApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	users.Fields.Add(
		&core.SelectField{
			Name:      "subscription_status",
			Values:    []string{"trial", "active", "past_due", "cancelled"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "subscription_id"},
		&core.NumberField{Name: "generations_used"},
	)
	if err := app.Save(users); err != nil {
		t.Fatalf("save users collection: %v", err)
	}

	// Create a test user with a known subscription_id.
	testUser := core.NewRecord(users)
	testUser.SetEmail("webhook-test@rekan.com.br")
	testUser.SetPassword("testpassword123!")
	testUser.Set("subscription_id", testSubscriptionID)
	testUser.Set("subscription_status", "trial")
	if err := app.Save(testUser); err != nil {
		t.Fatalf("save test user: %v", err)
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

// assertUserStatus finds the test user by subscription_id and verifies their status.
func assertUserStatus(t testing.TB, app *tests.TestApp, want string) {
	t.Helper()
	records, err := app.FindAllRecords("users", nil)
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	for _, r := range records {
		if r.GetString("subscription_id") == testSubscriptionID {
			got := r.GetString("subscription_status")
			if got != want {
				t.Errorf("subscription_status: got %q, want %q", got, want)
			}
			return
		}
	}
	t.Errorf("test user with subscription_id %q not found", testSubscriptionID)
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
	// When WebhookToken is empty string, the handler skips validation entirely.
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
			assertUserStatus(t, app, "active")
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
			assertUserStatus(t, app, "past_due")
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
		// SUBSCRIPTION_DELETED uses the "subscription" key, not "payment"
		Body: strings.NewReader(webhookBody("SUBSCRIPTION_DELETED", testSubscriptionID, true)),
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
			assertUserStatus(t, app, "cancelled")
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
