package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("messages")

		collection.Fields.Add(
			&core.RelationField{Name: "business", CollectionId: "businesses", MaxSelect: 1},
			&core.TextField{Name: "phone", Required: true},
			&core.SelectField{Name: "type", Values: []string{"text", "audio", "image"}, Required: true, MaxSelect: 1},
			&core.TextField{Name: "content"},
			&core.FileField{Name: "media", MaxSelect: 1, MaxSize: 10 * 1024 * 1024}, // 10MB
			&core.SelectField{Name: "direction", Values: []string{"incoming", "outgoing"}, Required: true, MaxSelect: 1},
			&core.DateField{Name: "wa_timestamp"},
			&core.TextField{Name: "wa_message_id"},
		)

		// Deduplication index on WhatsApp message ID
		collection.AddIndex("idx_messages_wa_message_id", true, "wa_message_id", "wa_message_id != ''")

		// Lookup by business
		collection.AddIndex("idx_messages_business", false, "business", "")

		// Lookup by phone
		collection.AddIndex("idx_messages_phone", false, "phone", "")

		// Operator can list/view messages for their businesses.
		// Single-operator MVP: any authenticated user can read all messages.
		listRule := `@request.auth.id != ""`
		viewRule := `@request.auth.id != ""`
		collection.ListRule = &listRule
		collection.ViewRule = &viewRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("messages")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
