package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}

		// Password auth (OAuth2 was removed in migration 1740000011)
		collection.PasswordAuth.Enabled = true

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
