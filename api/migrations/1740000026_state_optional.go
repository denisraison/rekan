package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		field := collection.Fields.GetByName("state")
		if tf, ok := field.(*core.TextField); ok {
			tf.Required = false
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		field := collection.Fields.GetByName("state")
		if tf, ok := field.(*core.TextField); ok {
			tf.Required = true
		}

		return app.Save(collection)
	})
}
