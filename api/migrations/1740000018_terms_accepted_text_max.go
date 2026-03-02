package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// terms_accepted_text stores the full ToS snapshot (~7k chars), which exceeds
// the TextField default max of 5000.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		f := collection.Fields.GetByName("terms_accepted_text")
		if f == nil {
			return nil
		}
		f.(*core.TextField).Max = 20000

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}

		f := collection.Fields.GetByName("terms_accepted_text")
		if f == nil {
			return nil
		}
		f.(*core.TextField).Max = 0

		return app.Save(collection)
	})
}
