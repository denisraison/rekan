package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add structured text field to agent_conversations for storing
// full MessageParam JSON (PEP-027 Wave 3).
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("agent_conversations")
		if err != nil {
			return err
		}
		col.Fields.Add(&core.TextField{Name: "structured", Max: 20000})
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("agent_conversations")
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("structured")
		return app.Save(col)
	})
}
