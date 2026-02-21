package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/denisraison/rekan/api/internal/asaas"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	_ "github.com/denisraison/rekan/api/migrations"
)

func main() {
	if err := run(os.Getenv); err != nil {
		log.Fatal(err)
	}
}

func run(getenv func(string) string) error {
	app := pocketbase.New()
	isDev := getenv("DEV_MODE") == "true"

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         "migrations",
		Automigrate: true,
	})

	apphttp.RegisterHooks(app)

	var asaasClient *asaas.Client
	if key := getenv("ASAAS_API_KEY"); key != "" {
		asaasClient = asaas.NewClient(key, isDev)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if isDev {
			disableRateLimits(app)
		}
		apphttp.RegisterRoutes(se.Router, handlers.Deps{
			App:          app,
			Asaas:        asaasClient,
			WebhookToken: getenv("ASAAS_WEBHOOK_TOKEN"),
		})
		return se.Next()
	})

	return app.Start()
}

func disableRateLimits(app core.App) {
	settings := app.Settings()
	settings.RateLimits.Enabled = false
	if err := app.Save(settings); err != nil {
		log.Printf("warning: failed to disable rate limits in dev mode: %v", err)
	}
}
