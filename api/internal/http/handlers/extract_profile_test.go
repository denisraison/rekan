package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/denisraison/rekan/api/internal/http/handlers"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// stubExtractFromAudio simulates extracting a business profile from audio.
// Returns a fixed result regardless of input for testing.
func stubExtractFromAudio(_ context.Context, _ []byte, _ string, _ string) (content.PartialBusinessProfile, error) {
	price80 := 80.0
	price200 := 200.0
	audience := "mulheres de 30 a 50 anos"
	return content.PartialBusinessProfile{
		Services: []content.PartialService{
			{Name: "Hidratação", PriceBRL: &price80},
			{Name: "Progressiva", PriceBRL: &price200},
		},
		TargetAudience: &audience,
	}, nil
}

func TestExtractProfileSuccess(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	// Build multipart body manually as a raw string for ApiScenario.
	// ApiScenario sets Content-Type header for us; use a raw body with boundary.
	boundary := "testboundary"
	body := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"business_type\"\r\n\r\n" +
		"salão de beleza\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"audio\"; filename=\"audio.webm\"\r\n" +
		"Content-Type: audio/webm\r\n\r\n" +
		"fake audio bytes\r\n" +
		"--" + boundary + "--\r\n"

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/profile:extract",
		Body:   strings.NewReader(body),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:              app,
				ExtractFromAudio: stubExtractFromAudio,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "multipart/form-data; boundary=" + boundary,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"Hidratação"`,
			`"price_brl":80`,
			`"Progressiva"`,
			`"price_brl":200`,
			`"target_audience":"mulheres de 30 a 50 anos"`,
			`"brand_vibe":null`,
		},
	}
	s.Test(t)
}

func TestExtractProfileMissingAudio(t *testing.T) {
	app, userID, _ := newHandlerApp(t)
	defer app.Cleanup()

	boundary := "testboundary"
	body := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"business_type\"\r\n\r\n" +
		"salão\r\n" +
		"--" + boundary + "--\r\n"

	s := &tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/businesses/profile:extract",
		Body:   strings.NewReader(body),
		TestAppFactory: func(_ testing.TB) *tests.TestApp { return app },
		BeforeTestFunc: func(_ testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			registerHandlerRoutes(app, e, handlers.Deps{
				App:              app,
				ExtractFromAudio: stubExtractFromAudio,
			})
		},
		Headers: map[string]string{
			"Authorization": authHeader(app, userID),
			"Content-Type":  "multipart/form-data; boundary=" + boundary,
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"message"`},
	}
	s.Test(t)
}
