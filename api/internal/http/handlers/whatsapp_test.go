package handlers_test

import (
	"net/http"
	"testing"

	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestWhatsAppStreamNotConfigured(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method:         http.MethodGet,
		URL:            "/api/whatsapp/stream",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{App: app, WhatsApp: nil})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusServiceUnavailable,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestWhatsAppStreamUnauthenticated(t *testing.T) {
	app, _, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method:         http.MethodGet,
		URL:            "/api/whatsapp/stream",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{App: app, WhatsApp: nil})
		},
		ExpectedStatus:  http.StatusUnauthorized,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}
