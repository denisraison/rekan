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
			&core.BoolField{Name: "reviewed"},
			&core.TextField{Name: "review_note"},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("reviewed")
		collection.Fields.RemoveByName("review_note")
		return app.Save(collection)
	})
}
