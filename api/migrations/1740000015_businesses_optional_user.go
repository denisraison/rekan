package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Make user and services optional so placeholder businesses can be created
// for unknown WhatsApp contacts before they complete onboarding.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		if f := collection.Fields.GetByName("user"); f != nil {
			f.(*core.RelationField).Required = false
		}
		if f := collection.Fields.GetByName("services"); f != nil {
			f.(*core.JSONField).Required = false
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}

		if f := collection.Fields.GetByName("user"); f != nil {
			f.(*core.RelationField).Required = true
		}
		if f := collection.Fields.GetByName("services"); f != nil {
			f.(*core.JSONField).Required = true
		}

		return app.Save(collection)
	})
}
