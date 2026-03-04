package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds profile_picture (file) and profile_picture_updated (date) to businesses.
// profile_picture holds the WhatsApp contact photo downloaded on first contact.
// profile_picture_updated records when it was last fetched so we can refresh
// after 7 days without hammering the WhatsApp API.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.FileField{Name: "profile_picture", MaxSelect: 1, MaxSize: 2 * 1024 * 1024},
			&core.TextField{Name: "profile_picture_id"},
			&core.DateField{Name: "profile_picture_updated"},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("profile_picture")
		collection.Fields.RemoveByName("profile_picture_id")
		collection.Fields.RemoveByName("profile_picture_updated")

		return app.Save(collection)
	})
}
