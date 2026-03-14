package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("agent_state")
		if err != nil {
			return err
		}
		return app.Delete(col)
	}, func(app core.App) error {
		col := core.NewBaseCollection("agent_state")
		col.Fields.Add(
			&core.TextField{Name: "operator_jid", Required: true},
			&core.SelectField{Name: "state", Values: []string{"idle", "collecting", "confirming"}},
			&core.TextField{Name: "action_type"},
			&core.JSONField{Name: "collected_fields"},
			&core.DateField{Name: "expires_at"},
		)
		return app.Save(col)
	})
}
