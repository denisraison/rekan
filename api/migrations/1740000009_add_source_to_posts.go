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

		messages, err2 := app.FindCollectionByNameOrId("messages")
		if err2 != nil {
			return err2
		}

		collection.Fields.Add(
			&core.TextField{Name: "source"},                                              // "dashboard" or "operator"
			&core.RelationField{Name: "message", CollectionId: messages.Id, MaxSelect: 1}, // source message for operator posts
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
