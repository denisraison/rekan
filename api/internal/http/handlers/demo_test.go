package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func demoScenario(t *testing.T, body string, genFn eval.GenerateFromMessageFunc) (*tests.TestApp, *tests.ApiScenario) {
	t.Helper()
	app, userID, _ := newHandlerApp(t)
	if genFn == nil {
		genFn = stubGenerateFromMessage
	}
	return app, &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/demo:generate",
		Body:   strings.NewReader(body),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:                 app,
				GenerateFromMessage: genFn,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "application/json",
		},
	}
}

func TestDemoMissingBusinessName(t *testing.T) {
	app, s := demoScenario(t, `{"message":"Fiz um bolo","business_name":""}`, nil)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusBadRequest
	s.ExpectedContent = []string{`"message"`}
	s.Test(t)
}

func TestDemoMissingMessage(t *testing.T) {
	app, s := demoScenario(t, `{"business_name":"Doces da Ana","business_type":"Confeitaria","city":"SP","message":""}`, nil)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusBadRequest
	s.ExpectedContent = []string{`"message"`}
	s.Test(t)
}

func TestDemoInvalidBody(t *testing.T) {
	app, s := demoScenario(t, `not json`, nil)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusBadRequest
	s.ExpectedContent = []string{`"message"`}
	s.Test(t)
}

func TestDemoUnauthenticated(t *testing.T) {
	app, s := demoScenario(t, `{"business_name":"Doces","business_type":"Confeitaria","city":"SP","message":"Bolo lindo"}`, nil)
	defer app.Cleanup()
	s.Headers = map[string]string{"Content-Type": "application/json"}
	s.ExpectedStatus = http.StatusUnauthorized
	s.ExpectedContent = []string{`"message"`}
	s.Test(t)
}

func TestDemoSuccess(t *testing.T) {
	app, s := demoScenario(t, `{"business_name":"Doces da Ana","business_type":"Confeitaria","city":"Guarulhos","services":"Bolo caseiro R$85, Brigadeiro R$35","message":"Fiz um bolo rosa e dourado"}`, nil)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusOK
	s.ExpectedContent = []string{`"caption"`, `"Legenda do operador"`, `"hashtags"`, `"production_note"`}
	s.AfterTestFunc = func(t testing.TB, app *tests.TestApp, _ *http.Response) {
		// Demo should NOT save posts
		posts, err := app.FindAllRecords("posts")
		if err != nil {
			t.Fatalf("find posts: %v", err)
		}
		if len(posts) != 0 {
			t.Errorf("expected 0 posts saved for demo, got %d", len(posts))
		}
	}
	s.Test(t)
}

func TestDemoGenerateError(t *testing.T) {
	failGenerate := func(_ context.Context, _ eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
		return eval.Post{}, fmt.Errorf("LLM unavailable")
	}
	app, s := demoScenario(t, `{"business_name":"Doces","business_type":"Confeitaria","city":"SP","message":"Bolo"}`, failGenerate)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusBadGateway
	s.ExpectedContent = []string{`"message"`}
	s.Test(t)
}

func TestDemoServicesWithoutPrices(t *testing.T) {
	var capturedProfile eval.BusinessProfile
	capture := func(_ context.Context, profile eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
		capturedProfile = profile
		return eval.Post{Caption: "ok", Hashtags: []string{}, ProductionNote: ""}, nil
	}
	app, s := demoScenario(t, `{"business_name":"Salão","business_type":"Salão de Beleza","city":"SP","services":"Corte, Escova progressiva","message":"Cortei hoje"}`, capture)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusOK
	s.ExpectedContent = []string{`"caption"`}
	s.AfterTestFunc = func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
		if len(capturedProfile.Services) != 2 {
			t.Fatalf("expected 2 services, got %d", len(capturedProfile.Services))
		}
		if capturedProfile.Services[0].Name != "Corte" {
			t.Errorf("service[0].Name = %q, want %q", capturedProfile.Services[0].Name, "Corte")
		}
		if capturedProfile.Services[0].PriceBRL != 0 {
			t.Errorf("service[0].PriceBRL = %v, want 0", capturedProfile.Services[0].PriceBRL)
		}
		if capturedProfile.Services[1].Name != "Escova progressiva" {
			t.Errorf("service[1].Name = %q, want %q", capturedProfile.Services[1].Name, "Escova progressiva")
		}
	}
	s.Test(t)
}

func TestDemoServicesPriceParsing(t *testing.T) {
	var capturedProfile eval.BusinessProfile
	capture := func(_ context.Context, profile eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
		capturedProfile = profile
		return eval.Post{Caption: "ok", Hashtags: []string{}, ProductionNote: ""}, nil
	}
	app, s := demoScenario(t, `{"business_name":"Doces","business_type":"Confeitaria","city":"SP","services":"Bolo caseiro R$85, Brigadeiro R$35","message":"teste"}`, capture)
	defer app.Cleanup()
	s.ExpectedStatus = http.StatusOK
	s.ExpectedContent = []string{`"caption"`}
	s.AfterTestFunc = func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
		if len(capturedProfile.Services) != 2 {
			t.Fatalf("expected 2 services, got %d", len(capturedProfile.Services))
		}
		if capturedProfile.Services[0].Name != "Bolo caseiro" {
			t.Errorf("service[0].Name = %q, want %q", capturedProfile.Services[0].Name, "Bolo caseiro")
		}
		if capturedProfile.Services[0].PriceBRL != 85 {
			t.Errorf("service[0].PriceBRL = %v, want 85", capturedProfile.Services[0].PriceBRL)
		}
		if capturedProfile.Services[1].Name != "Brigadeiro" {
			t.Errorf("service[1].Name = %q, want %q", capturedProfile.Services[1].Name, "Brigadeiro")
		}
		if capturedProfile.Services[1].PriceBRL != 35 {
			t.Errorf("service[1].PriceBRL = %v, want 35", capturedProfile.Services[1].PriceBRL)
		}
	}
	s.Test(t)
}
