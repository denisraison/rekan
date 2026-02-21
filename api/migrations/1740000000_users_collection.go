package migrations

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		// Google OAuth only, no password auth
		collection.PasswordAuth.Enabled = false
		collection.OAuth2.Enabled = true
		collection.OAuth2.Providers = []core.OAuth2ProviderConfig{
			{
				Name:         "google",
				ClientId:     os.Getenv("GOOGLE_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			},
		}

		// Server-managed fields (client cannot write these)
		collection.Fields.Add(
			&core.SelectField{
				Name:      "subscription_status",
				Values:    []string{"trial", "active", "past_due", "cancelled"},
				MaxSelect: 1,
			},
			&core.TextField{Name: "subscription_id"},
			&core.NumberField{Name: "generations_used"},
		)

		// API rules: user can view own record, cannot write server-managed fields
		viewRule := "id = @request.auth.id"
		updateRule := "id = @request.auth.id" +
			" && @request.body.subscription_status:isset = false" +
			" && @request.body.subscription_id:isset = false" +
			" && @request.body.generations_used:isset = false"

		collection.ListRule = nil // no client listing of other users
		collection.ViewRule = &viewRule
		collection.UpdateRule = &updateRule
		collection.DeleteRule = nil // users cannot self-delete

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
