package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/denisraison/rekan/api/internal/asaas"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/denisraison/rekan/eval"
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

	// WhatsApp client (optional, skipped if no data dir available)
	var waClient *whatsapp.Client
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if isDev {
			disableRateLimits(app)
		}

		// Start WhatsApp client, store session alongside PocketBase data
		dbPath := filepath.Join(app.DataDir(), "whatsapp.db")
		wac, err := whatsapp.New(context.Background(), dbPath)
		if err != nil {
			log.Printf("warning: whatsapp client failed to init: %v", err)
		} else {
			var whisperClient *transcribe.Client
			if key := getenv("OPENAI_API_KEY"); key != "" {
				whisperClient = transcribe.NewClient(key)
			}
			whatsapp.RegisterMessageHandler(whatsapp.HandlerDeps{
				Client:     wac,
				App:        app,
				Transcribe: whisperClient,
			})
			if err := wac.Connect(context.Background()); err != nil {
				log.Printf("warning: whatsapp connect failed: %v", err)
			} else {
				waClient = wac
			}
		}

		apphttp.RegisterRoutes(se.Router, handlers.Deps{
			App:                 app,
			Asaas:               asaasClient,
			WhatsApp:            waClient,
			WebhookToken:        getenv("ASAAS_WEBHOOK_TOKEN"),
			Generate:            eval.Generate,
			GenerateFromMessage: eval.GenerateFromMessage,
		})
		return se.Next()
	})

	app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		if waClient != nil {
			waClient.Disconnect()
		}
		return te.Next()
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
