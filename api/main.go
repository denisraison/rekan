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
	"github.com/denisraison/rekan/api/internal/billing"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/denisraison/rekan/eval"
	_ "github.com/denisraison/rekan/api/migrations"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Getenv); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, getenv func(string) string) error {
	app := pocketbase.New()
	isDev := getenv("DEV_MODE") == "true"

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         "migrations",
		Automigrate: true,
	})

	var asaasClient *asaas.Client
	if key := getenv("ASAAS_API_KEY"); key != "" {
		asaasClient = asaas.NewClient(key, isDev)
	}

	ctx, cancel := context.WithCancel(ctx)

	// WhatsApp client (optional, skipped if no data dir available)
	var waClient *whatsapp.Client
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if isDev {
			disableRateLimits(app)
		}

		if err := configureBackups(app, getenv); err != nil {
			log.Printf("warning: failed to configure backups: %v", err)
		}

		// Start WhatsApp client, store session alongside PocketBase data
		dbPath := filepath.Join(app.DataDir(), "whatsapp.db")
		wac, err := whatsapp.New(ctx, dbPath)
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
			if err := wac.Connect(ctx); err != nil {
				log.Printf("warning: whatsapp connect failed: %v", err)
			} else {
				waClient = wac
			}
		}

		app.Cron().MustAdd("create-charges", "0 10 * * *", func() {
			billing.CreatePendingCharges(app, asaasClient)
		})

		apphttp.RegisterRoutes(se.Router, handlers.Deps{
			App:                 app,
			Asaas:               asaasClient,
			WhatsApp:            waClient,
			WebhookToken:        getenv("ASAAS_WEBHOOK_TOKEN"),
			AppURL:              getenv("APP_URL"),
			Generate:            eval.Generate,
			GenerateFromMessage: eval.GenerateFromMessage,
		})
		return se.Next()
	})

	app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		cancel()
		if waClient != nil {
			waClient.Disconnect()
		}
		return te.Next()
	})

	return app.Start()
}

func configureBackups(app core.App, getenv func(string) string) error {
	bucket := getenv("GCS_BACKUP_BUCKET")
	if bucket == "" {
		return nil
	}

	settings := app.Settings()
	settings.Backups.Cron = "0 3 * * *" // daily at 03:00
	settings.Backups.CronMaxKeep = 7
	settings.Backups.S3.Enabled = true
	settings.Backups.S3.Bucket = bucket
	settings.Backups.S3.Region = getenv("GCS_BACKUP_REGION")
	settings.Backups.S3.Endpoint = "https://storage.googleapis.com"
	settings.Backups.S3.AccessKey = getenv("GCS_BACKUP_ACCESS_KEY")
	settings.Backups.S3.Secret = getenv("GCS_BACKUP_SECRET")
	settings.Backups.S3.ForcePathStyle = false
	return app.Save(settings)
}

func disableRateLimits(app core.App) {
	settings := app.Settings()
	settings.RateLimits.Enabled = false
	if err := app.Save(settings); err != nil {
		log.Printf("warning: failed to disable rate limits in dev mode: %v", err)
	}
}
