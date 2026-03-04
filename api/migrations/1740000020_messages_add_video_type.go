package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds "video" to the messages.type select and raises the media file size
// limit to 50 MB to accommodate WhatsApp video clips.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("messages")
		if err != nil {
			return err
		}

		typeField := collection.Fields.GetByName("type")
		if typeField == nil {
			return nil
		}
		typeField.(*core.SelectField).Values = []string{"text", "audio", "image", "video"}

		mediaField := collection.Fields.GetByName("media")
		if mediaField == nil {
			return nil
		}
		mediaField.(*core.FileField).MaxSize = 50 * 1024 * 1024 // 50 MB

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("messages")
		if err != nil {
			return nil
		}

		typeField := collection.Fields.GetByName("type")
		if typeField == nil {
			return nil
		}
		typeField.(*core.SelectField).Values = []string{"text", "audio", "image"}

		mediaField := collection.Fields.GetByName("media")
		if mediaField == nil {
			return nil
		}
		mediaField.(*core.FileField).MaxSize = 10 * 1024 * 1024 // 10 MB

		return app.Save(collection)
	})
}
