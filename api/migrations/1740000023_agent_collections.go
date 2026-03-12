package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Creates collections for the WhatsApp group agent (PEP-023).
func init() {
	m.Register(func(app core.App) error {
		// operators: authorized agent users
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
		if err := app.Save(operators); err != nil {
			return err
		}

		// agent_state: per-operator confirmation state
		agentState := core.NewBaseCollection("agent_state")
		agentState.Fields.Add(
			&core.TextField{Name: "operator_jid", Required: true},
			&core.SelectField{Name: "state", Values: []string{"idle", "collecting", "confirming"}},
			&core.TextField{Name: "action_type"},
			&core.JSONField{Name: "collected_fields"},
			&core.DateField{Name: "expires_at"},
		)
		if err := app.Save(agentState); err != nil {
			return err
		}

		// agent_conversations: group message buffer
		agentConversations := core.NewBaseCollection("agent_conversations")
		agentConversations.Fields.Add(
			&core.TextField{Name: "operator_name"},
			&core.TextField{Name: "operator_jid"},
			&core.SelectField{Name: "role", Values: []string{"user", "assistant"}},
			&core.TextField{Name: "content"},
			&core.TextField{Name: "media_type"},
			&core.DateField{Name: "timestamp"},
		)
		if err := app.Save(agentConversations); err != nil {
			return err
		}

		// agent_action_log: audit trail
		agentActionLog := core.NewBaseCollection("agent_action_log")
		agentActionLog.Fields.Add(
			&core.TextField{Name: "operator_name"},
			&core.TextField{Name: "operator_jid"},
			&core.TextField{Name: "action_type"},
			&core.JSONField{Name: "params"},
			&core.TextField{Name: "result"},
			&core.BoolField{Name: "success"},
			&core.NumberField{Name: "latency_ms"},
		)
		if err := app.Save(agentActionLog); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		for _, name := range []string{"agent_action_log", "agent_conversations", "agent_state", "operators"} {
			col, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if err := app.Delete(col); err != nil {
				return err
			}
		}
		return nil
	})
}
