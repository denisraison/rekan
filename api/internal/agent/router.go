package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// RouteAction executes the action from an AgentResponse and returns the reply text.
// Uses pre-queried data from ctx to avoid redundant DB queries.
func RouteAction(ctx HydratedContext, action types.AgentAction) (string, error) {
	switch action.ActionType {
	case types.AgentActionTypeSTATUS_OVERVIEW:
		return fmt.Sprintf("%s, temos %d clientes ativas e %d posts gerados.",
			ctx.OperatorName, len(ctx.Businesses), ctx.PostCount), nil
	case types.AgentActionTypeCUSTOMER_LIST:
		return customerList(ctx)
	default:
		return "", fmt.Errorf("unknown action type: %s", action.ActionType)
	}
}

func customerList(ctx HydratedContext) (string, error) {
	if len(ctx.Businesses) == 0 {
		return fmt.Sprintf("%s, nenhuma cliente ativa no momento.", ctx.OperatorName), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, clientes ativas:\n", ctx.OperatorName)
	for _, biz := range ctx.Businesses {
		fmt.Fprintf(&b, "- %s (%s, %s)\n", biz.GetString("name"), biz.GetString("type"), biz.GetString("city"))
	}
	return b.String(), nil
}

// LogAction records an action to the agent_action_log collection.
func LogAction(app core.App, operatorName, operatorJID, actionType string, params map[string]string, result string, success bool, start time.Time) {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollAgentActionLog)
	if err != nil {
		return
	}
	record := core.NewRecord(col)
	record.Set("operator_name", operatorName)
	record.Set("operator_jid", operatorJID)
	record.Set("action_type", actionType)
	record.Set("params", params)
	record.Set("result", result)
	record.Set("success", success)
	record.Set("latency_ms", time.Since(start).Milliseconds())
	_ = app.Save(record)
}
