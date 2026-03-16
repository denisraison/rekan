package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"

	"github.com/denisraison/rekan/api/internal/agent"
	"github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/billing"
	content "github.com/denisraison/rekan/api/internal/content"
	apphttp "github.com/denisraison/rekan/api/internal/http"
	"github.com/denisraison/rekan/api/internal/http/handlers"
	"github.com/denisraison/rekan/api/internal/operator"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/denisraison/rekan/api/internal/whatsapp"
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
			var extractSignal content.ExtractSignalFunc
			if key := getenv("GEMINI_API_KEY"); key != "" {
				whisperClient = transcribe.NewClient(key)
				extractSignal = content.ExtractProfileSignal
			}

			// Create group agent if CLAUDE_API_KEY is set
			var handleGroupMsg whatsapp.GroupMessageHandler
			if key := getenv("CLAUDE_API_KEY"); key != "" {
				groupAgent := agent.New(app, wac, app.Logger(), whisperClient, content.Generate, key)
				handleGroupMsg = groupAgent.HandleGroupMessage
			}

			whatsapp.RegisterMessageHandler(whatsapp.HandlerDeps{
				Client:         wac,
				App:            app,
				Logger:         app.Logger(),
				Transcribe:     whisperClient,
				ExtractSignal:  extractSignal,
				HandleGroupMsg: handleGroupMsg,
				AgentGroupJID:  getenv("REKAN_AGENT_GROUP_JID"),
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

		var extractFromAudio content.ExtractFromAudioFunc
		if key := getenv("GEMINI_API_KEY"); key != "" {
			tc := transcribe.NewClient(key)
			extractFromAudio = func(ctx context.Context, audioBytes []byte, mimeType string, businessType string) (content.PartialBusinessProfile, error) {
				transcript, err := tc.Transcribe(ctx, audioBytes, mimeType)
				if err != nil {
					return content.PartialBusinessProfile{}, err
				}
				return content.ExtractBusinessProfile(ctx, transcript, businessType)
			}
		}

		var transcribeClient *transcribe.Client
		if key := getenv("GEMINI_API_KEY"); key != "" {
			transcribeClient = transcribe.NewClient(key)
		}

		apphttp.RegisterRoutes(se.Router, handlers.Deps{
			App:                 app,
			Asaas:               asaasClient,
			WhatsApp:            waClient,
			Transcribe:          transcribeClient,
			WebhookToken:        getenv("ASAAS_WEBHOOK_TOKEN"),
			AppURL:              getenv("APP_URL"),
			Generate:            content.Generate,
			GenerateFromMessage: content.GenerateFromMessage,
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

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "list-groups",
		Short: "List WhatsApp groups and their JIDs",
		RunE: func(_ *cobra.Command, _ []string) error {
			dbPath := filepath.Join(app.DataDir(), "whatsapp.db")
			wac, err := whatsapp.New(ctx, dbPath, "Rekan", app.Logger())
			if err != nil {
				return fmt.Errorf("whatsapp init: %w", err)
			}
			defer wac.Disconnect()

			if err := wac.Connect(ctx); err != nil {
				return fmt.Errorf("whatsapp connect: %w", err)
			}

			groups, err := wac.GetJoinedGroups(ctx)
			if err != nil {
				return fmt.Errorf("get groups: %w", err)
			}

			for _, g := range groups {
				fmt.Printf("%s  %s\n", g.JID.User, g.Name)
			}
			return nil
		},
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
