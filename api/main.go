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
	"github.com/denisraison/rekan/api/internal/operator"
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
			app.Logger().Warn("failed to configure backups", "error", err)
		}

		// Start WhatsApp client, store session alongside PocketBase data
		dbPath := filepath.Join(app.DataDir(), "whatsapp.db")
		wac, err := whatsapp.New(ctx, dbPath, "Rekan", app.Logger())
		if err != nil {
			app.Logger().Warn("whatsapp client failed to init", "error", err)
		} else {
			var whisperClient *transcribe.Client
			var extractSignal whatsapp.ExtractSignalFunc
			if key := getenv("GEMINI_API_KEY"); key != "" {
				whisperClient = transcribe.NewClient(key)
				extractSignal = func(ctx context.Context, message, businessType string) (*whatsapp.ProfileSignal, error) {
					sig, err := eval.ExtractProfileSignal(ctx, message, businessType)
					if err != nil || sig == nil {
						return nil, err
					}
					return &whatsapp.ProfileSignal{Field: sig.Field, Value: sig.Value}, nil
				}
			}
			whatsapp.RegisterMessageHandler(whatsapp.HandlerDeps{
				Client:        wac,
				App:           app,
				Logger:        app.Logger(),
				Transcribe:    whisperClient,
				ExtractSignal: extractSignal,
			})
			if err := wac.Connect(ctx); err != nil {
				app.Logger().Warn("whatsapp connect failed", "error", err)
			} else {
				waClient = wac
			}
		}

		app.Cron().MustAdd("create-charges", "0 10 * * *", func() {
			billing.CreatePendingCharges(ctx, app, asaasClient)
		})

		app.Cron().MustAdd("seasonal-messages", "0 8 * * *", func() {
			operator.QueueSeasonalMessages(app)
		})

		var extractFromAudio eval.ExtractFromAudioFunc
		if key := getenv("GEMINI_API_KEY"); key != "" {
			tc := transcribe.NewClient(key)
			extractFromAudio = func(ctx context.Context, audioBytes []byte, mimeType string, businessType string) (eval.PartialBusinessProfile, error) {
				transcript, err := tc.Transcribe(ctx, audioBytes, mimeType)
				if err != nil {
					return eval.PartialBusinessProfile{}, err
				}
				return eval.ExtractBusinessProfile(ctx, transcript, businessType)
			}
		}

		apphttp.RegisterRoutes(se.Router, handlers.Deps{
			App:                 app,
			Asaas:               asaasClient,
			WhatsApp:            waClient,
			WebhookToken:        getenv("ASAAS_WEBHOOK_TOKEN"),
			AppURL:              getenv("APP_URL"),
			Generate:            eval.Generate,
			GenerateFromMessage: eval.GenerateFromMessage,
			ExtractFromAudio:    extractFromAudio,
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
		app.Logger().Warn("failed to disable rate limits in dev mode", "error", err)
	}
}
