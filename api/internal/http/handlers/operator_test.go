package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/denisraison/rekan/eval"
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
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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

func TestOperatorForbidden(t *testing.T) {
	app, _, bizID := newHandlerApp(t)
	defer app.Cleanup()

	users, _ := app.FindCollectionByNameOrId("users")
	other := core.NewRecord(users)
	other.SetEmail("operator-other@rekan.com.br")
	other.SetPassword(testUserPassword)
	other.Set("subscription_status", "trial")
	if err := app.Save(other); err != nil {
		t.Fatalf("save other user: %v", err)
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":"test"}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: stubGenerateFromMessage,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, other.Id),
			"Content-Type":  "application/json",
		},
		ExpectedStatus:  http.StatusForbidden,
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
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: stubGenerateFromMessage,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			posts, err := app.FindAllRecords("posts")
			if err != nil {
				t.Fatalf("find posts: %v", err)
			}
			if len(posts) != 1 {
				t.Errorf("expected 1 post, got %d", len(posts))
				return
			}
			if got := posts[0].GetString("source"); got != "operator" {
				t.Errorf("source: got %q, want %q", got, "operator")
			}
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"caption"`, `"Legenda do operador"`},
	}
	s.Test(t)
}

func TestOperatorGenerateError(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	failGenerate := func(_ context.Context, _ eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
		return eval.Post{}, fmt.Errorf("LLM unavailable")
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generateFromMessage",
		Body:   strings.NewReader(`{"message":"Fiz um bolo hoje"}`),
		TestAppFactory: func(tb testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
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
