package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestSubscribeAsaasNil(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/subscriptions",
		Body:   strings.NewReader(`{}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:   app,
				Asaas: nil,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusServiceUnavailable,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestSubscribeAlreadyActive(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	user, _ := app.FindRecordById("users", userID)
	user.Set("subscription_status", "active")
	if err := app.Save(user); err != nil {
		t.Fatalf("set active: %v", err)
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/subscriptions",
		Body:   strings.NewReader(`{}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient("http://unused", "key"),
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusConflict,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestSubscribeSuccess(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	// Mock Asaas API
	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/customers"):
			json.NewEncoder(w).Encode(map[string]string{"id": "cus_test123"})
		case strings.Contains(r.URL.Path, "/subscriptions"):
			json.NewEncoder(w).Encode(map[string]string{
				"id":          "sub_test456",
				"paymentLink": "https://pay.example.com/test",
			})
		}
	}))
	defer mockAsaas.Close()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/subscriptions",
		Body:   strings.NewReader(`{"billing_type":"PIX"}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient(mockAsaas.URL, "test-key"),
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			user, err := app.FindRecordById("users", userID)
			if err != nil {
				t.Fatalf("find user: %v", err)
			}
			got := user.GetString("subscription_id")
			if got != "sub_test456" {
				t.Errorf("subscription_id: got %q, want %q", got, "sub_test456")
			}
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"payment_url"`},
	}
	s.Test(t)
}

func TestSubscribeAsaasError(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	mockAsaas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"errors":[{"description":"internal error"}]}`)
	}))
	defer mockAsaas.Close()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/subscriptions",
		Body:   strings.NewReader(`{}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:   app,
				Asaas: asaas.NewTestClient(mockAsaas.URL, "test-key"),
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusBadGateway,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestGetSubscription(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodGet,
		URL:    "/api/subscriptions/current",
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{App: app})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status"`, `"subscription_id"`, `"generations_used"`},
	}
	s.Test(t)
}
