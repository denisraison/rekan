package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates profile_suggestions collection.
// Populated by background extraction after incoming WhatsApp messages.
// Elenice reviews and applies or dismisses each suggestion from the operator UI.
func init() {
	m.Register(func(app core.App) error {
		businesses, err := app.FindCollectionByNameOrId("businesses")
		if err != nil {
			return err
		}

		collection := core.NewBaseCollection("profile_suggestions")

		collection.Fields.Add(
			&core.RelationField{Name: "business", CollectionId: businesses.Id, MaxSelect: 1, Required: true},
			&core.TextField{Name: "field", Required: true},    // "services", "quirks", "target_audience", "brand_vibe"
			&core.TextField{Name: "suggestion", Required: true}, // for services: "Name|price_brl"; for others: plain text
			&core.BoolField{Name: "dismissed"},
		)

		// Authenticated operators can list, view, and update (dismiss) suggestions.
		// Creation happens only from the backend (no create rule needed for the frontend).
		listRule := `@request.auth.id != ""`
		viewRule := `@request.auth.id != ""`
		updateRule := `@request.auth.id != ""`

		collection.ListRule = &listRule
		collection.ViewRule = &viewRule
		collection.UpdateRule = &updateRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("profile_suggestions")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
