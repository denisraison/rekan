package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Drops the operators collection (PEP-023 Wave 1.1).
// Group membership replaces per-operator JID auth.
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("operators")
		if err != nil {
			return nil
		}
		return app.Delete(col)
	}, func(app core.App) error {
		operators := core.NewBaseCollection("operators")
		operators.Fields.Add(
			&core.TextField{Name: "jid", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.TextField{Name: "role"},
			&core.BoolField{Name: "active"},
		)
		operators.Indexes = []string{
			"CREATE UNIQUE INDEX idx_operators_jid ON operators (jid)",
		}
		return app.Save(operators)
	})
}
