package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("businesses")

		collection.Fields.Add(
			&core.RelationField{Name: "user", CollectionId: "users", MaxSelect: 1, Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.TextField{Name: "type", Required: true},
			&core.TextField{Name: "city", Required: true},
			&core.TextField{Name: "state", Required: true},
			&core.TextField{Name: "description"},
			&core.JSONField{Name: "services", Required: true},
			&core.TextField{Name: "target_audience"},
			&core.TextField{Name: "brand_vibe"},
			&core.TextField{Name: "quirks"},
			&core.NumberField{Name: "onboarding_step"},
		)

		// One business per user
		collection.AddIndex("idx_businesses_user_unique", true, "user", "")

		listRule := "user = @request.auth.id"
		viewRule := "user = @request.auth.id"
		createRule := `@request.auth.id != "" && @request.body.user = @request.auth.id`
		updateRule := "user = @request.auth.id && @request.body.user:isset = false"
		deleteRule := "user = @request.auth.id"

		collection.ListRule = &listRule
		collection.ViewRule = &viewRule
		collection.CreateRule = &createRule
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &deleteRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
