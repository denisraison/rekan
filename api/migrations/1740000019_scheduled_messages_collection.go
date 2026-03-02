package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		businesses, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		collection := core.NewBaseCollection("scheduled_messages")

		collection.Fields.Add(
			&core.RelationField{Name: "business", CollectionId: businesses.Id, MaxSelect: 1, Required: true},
			&core.TextField{Name: "text"},
			&core.DateField{Name: "scheduled_for"},
			&core.BoolField{Name: "approved"},
			&core.BoolField{Name: "dismissed"},
		)

		authed := `@request.auth.id != ""`
		collection.ListRule = &authed
		collection.ViewRule = &authed
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("scheduled_messages")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
