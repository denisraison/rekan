package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Raise the default 5000-char TextField limit on agent_conversations.content
// and agent_action_log.result so longer Claude replies can be stored.
func init() {
	m.Register(func(app core.App) error {
		if err := setTextFieldMax(app, "agent_conversations", "content", 20000); err != nil {
			return err
		}
		return setTextFieldMax(app, "agent_action_log", "result", 20000)
	}, func(app core.App) error {
		_ = setTextFieldMax(app, "agent_conversations", "content", 0)
		_ = setTextFieldMax(app, "agent_action_log", "result", 0)
		return nil
	})
}

func setTextFieldMax(app core.App, collection, field string, max int) error {
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return err
	}
	f := col.Fields.GetByName(field)
	if f == nil {
		return nil
	}
	f.(*core.TextField).Max = max
	return app.Save(col)
}
