package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "source"},                                                           // "dashboard" or "operator"
			&core.RelationField{Name: "message", CollectionId: "messages", MaxSelect: 1}, // source message for operator posts
		)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("source")
		collection.Fields.RemoveByName("message")

		return app.Save(collection)
	})
}
