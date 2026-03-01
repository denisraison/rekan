package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById(collection.Fields.GetByName("subscription_status").GetId())
		collection.Fields.RemoveById(collection.Fields.GetByName("subscription_id").GetId())
		collection.Fields.RemoveById(collection.Fields.GetByName("generations_used").GetId())

		// Simplify update rule: drop the server-managed field guards
		updateRule := "id = @request.auth.id"
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.SelectField{
				Name:      "subscription_status",
				Values:    []string{"trial", "active", "past_due", "cancelled"},
				MaxSelect: 1,
			},
			&core.TextField{Name: "subscription_id"},
			&core.NumberField{Name: "generations_used"},
		)

		updateRule := "id = @request.auth.id" +
			" && @request.body.subscription_status:isset = false" +
			" && @request.body.subscription_id:isset = false" +
			" && @request.body.generations_used:isset = false"
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	})
}
