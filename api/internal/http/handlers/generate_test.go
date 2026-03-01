package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/denisraison/rekan/eval"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestGenerateNotFound(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/nonexistent/posts:generate",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:      app,
				Generate: stubGenerate,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}


func TestGenerateSuccess(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generate",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:      app,
				Generate: stubGenerate,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			posts, err := app.FindAllRecords("posts")
			if err != nil {
				t.Fatalf("find posts: %v", err)
			}
			if len(posts) != 1 {
				t.Errorf("expected 1 post, got %d", len(posts))
			}
		},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"batch_id"`, `"posts"`, `"Legenda de teste"`},
	}
	s.Test(t)
}

func TestGenerateError(t *testing.T) {
	app, userID, bizID := newHandlerApp(t)
	defer app.Cleanup()

	failGenerate := func(_ context.Context, _ eval.BusinessProfile, _ []eval.Role, _ []string) ([]eval.Post, error) {
		return nil, fmt.Errorf("LLM unavailable")
	}

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/" + bizID + "/posts:generate",
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:      app,
				Generate: failGenerate,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
		},
		ExpectedStatus:  http.StatusBadGateway,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
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
