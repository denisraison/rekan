package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOperatorNotFound(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/nonexistent/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":"test"}`),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: stubGenerateFromMessage,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}


func TestOperatorEmptyMessage(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":""}`),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: stubGenerateFromMessage,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}

func TestOperatorSuccess(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":"Fiz um bolo hoje"}`),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: stubGenerateFromMessage,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"caption"`, `"Legenda do operador"`},
	}
	s.Test(t)
}

func TestOperatorGenerateError(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	failGenerate := func(_ context.Context, _ content.BusinessProfile, _ string, _ []string) (content.Post, error) {
		return content.Post{}, errors.New("LLM unavailable")
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":"Fiz um bolo hoje"}`),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: failGenerate,
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
