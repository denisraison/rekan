package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("posts")

		collection.Fields.Add(
			&core.RelationField{Name: "business", CollectionId: "businesses", MaxSelect: 1, Required: true},
			&core.TextField{Name: "caption", Required: true},
			&core.JSONField{Name: "hashtags"},
			&core.TextField{Name: "production_note"},
			&core.TextField{Name: "role"},
			&core.TextField{Name: "hook"},
			&core.TextField{Name: "batch_id"},
			&core.BoolField{Name: "edited"},
		)

		listRule := "business.user = @request.auth.id"
		viewRule := "business.user = @request.auth.id"
		updateRule := "business.user = @request.auth.id" +
			" && @request.body.business:isset = false" +
			" && @request.body.batch_id:isset = false" +
			" && @request.body.role:isset = false" +
			" && @request.body.hook:isset = false"
		deleteRule := "business.user = @request.auth.id"

		collection.ListRule = &listRule
		collection.ViewRule = &viewRule
		collection.CreateRule = nil // server-only via generate endpoint
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &deleteRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
