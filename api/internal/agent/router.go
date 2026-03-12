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
// For NEEDS_CONFIRMATION actions, it stores state and returns empty string (BAML reply carries the prompt).
// For EXECUTE actions, it runs the action immediately.
func RouteAction(app core.App, ctx HydratedContext, state *OperatorState, action types.AgentAction) (string, error) {
	switch action.ActionType {
	case types.AgentActionTypeSTATUS_OVERVIEW:
		return fmt.Sprintf("%s, temos %d clientes ativas e %d posts gerados.",
			ctx.OperatorName, len(ctx.Businesses), ctx.PostCount), nil

	case types.AgentActionTypeCUSTOMER_LIST:
		return customerList(ctx)

	case types.AgentActionTypeCUSTOMER_INFO:
		return customerInfo(ctx, action.ActionParams)

	case types.AgentActionTypeCUSTOMER_CREATE:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			name := action.ActionParams["name"]
			if name != "" {
				for _, biz := range ctx.Businesses {
					if normalizeForMatch(biz.GetString("name")) == normalizeForMatch(name) {
						return fmt.Sprintf("%s, a %s já existe (%s, %s).",
							ctx.OperatorName, biz.GetString("name"),
							biz.GetString("type"), biz.GetString("city")), nil
					}
				}
			}
			if who, conflict := HasPendingAction(app, ctx.OperatorJID, name); conflict {
				return fmt.Sprintf("%s, outro operador (%s) já tem uma ação pendente pra essa cliente.", ctx.OperatorName, who), nil
			}
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executeCustomerCreate(app, ctx, action.ActionParams)

	case types.AgentActionTypeCUSTOMER_UPDATE:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executeCustomerUpdate(app, ctx, action.ActionParams)

	case types.AgentActionTypeCUSTOMER_PAUSE:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executeCustomerPause(app, ctx, action.ActionParams)

	default:
		return "", fmt.Errorf("unknown action type: %s", action.ActionType)
	}
}

func storeAndConfirm(app core.App, state *OperatorState, ctx HydratedContext, actionType string, params map[string]string) (string, error) {
	if err := SetConfirming(app, state, ctx.OperatorJID, actionType, params); err != nil {
		return "", fmt.Errorf("saving state: %w", err)
	}
	// BAML reply carries the confirmation prompt
	return "", nil
}

// ExecuteConfirmed runs the pending action after the operator said "sim".
func ExecuteConfirmed(app core.App, ctx HydratedContext, state *OperatorState) (string, error) {
	defer ClearState(app, state, ctx.OperatorJID)

	switch state.ActionType {
	case string(types.AgentActionTypeCUSTOMER_CREATE):
		return executeCustomerCreate(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypeCUSTOMER_UPDATE):
		return executeCustomerUpdate(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypeCUSTOMER_PAUSE):
		return executeCustomerPause(app, ctx, state.CollectedFields)
	default:
		return "", fmt.Errorf("unknown pending action: %s", state.ActionType)
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

func customerInfo(ctx HydratedContext, params map[string]string) (string, error) {
	name := params["name"]
	if name == "" {
		return fmt.Sprintf("%s, qual cliente você quer saber?", ctx.OperatorName), nil
	}

	matches := findBusinessRecords(ctx.Businesses, name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei nenhuma cliente com esse nome.", ctx.OperatorName), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	m := matches[0]
	return fmt.Sprintf("%s, %s: %s em %s.", ctx.OperatorName, m.GetString("name"), m.GetString("type"), m.GetString("city")), nil
}

func executeCustomerCreate(app core.App, ctx HydratedContext, fields map[string]string) (string, error) {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		return "", fmt.Errorf("businesses collection: %w", err)
	}
	record := core.NewRecord(col)
	record.Set("name", fields["name"])
	record.Set("type", fields["type"])
	record.Set("city", fields["city"])
	record.Set("invite_status", domain.InviteStatusDraft)

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("creating business: %w", err)
	}

	return fmt.Sprintf("%s, %s cadastrada! (%s, %s)",
		ctx.OperatorName, fields["name"], fields["type"], fields["city"]), nil
}

func executeCustomerUpdate(app core.App, ctx HydratedContext, fields map[string]string) (string, error) {
	name := fields["name"]
	matches := findBusinessRecords(ctx.Businesses, name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, name), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	record := matches[0]
	updated := make([]string, 0, len(fields))
	for k, v := range fields {
		if k == "name" {
			continue
		}
		record.Set(k, v)
		updated = append(updated, k)
	}

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("updating business: %w", err)
	}

	return fmt.Sprintf("%s, %s atualizada! Campos: %s.",
		ctx.OperatorName, name, strings.Join(updated, ", ")), nil
}

func executeCustomerPause(app core.App, ctx HydratedContext, fields map[string]string) (string, error) {
	name := fields["name"]
	matches := findBusinessRecords(ctx.Businesses, name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, name), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	record := matches[0]
	record.Set("invite_status", domain.InviteStatusCancelled)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("pausing business: %w", err)
	}

	reason := fields["reason"]
	if reason != "" {
		return fmt.Sprintf("%s, %s pausada. Motivo: %s.", ctx.OperatorName, name, reason), nil
	}
	return fmt.Sprintf("%s, %s pausada.", ctx.OperatorName, name), nil
}

// findBusinessRecords returns matching business records for a name query.
func findBusinessRecords(businesses []*core.Record, query string) []*core.Record {
	normalized := normalizeForMatch(query)
	if normalized == "" {
		return nil
	}

	var matches []*core.Record
	for _, biz := range businesses {
		name := normalizeForMatch(biz.GetString("name"))
		if name == normalized || strings.Contains(name, normalized) || strings.Contains(normalized, name) {
			matches = append(matches, biz)
		}
	}
	return matches
}

func disambiguate(operatorName string, matches []*core.Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s, encontrei mais de uma:\n", operatorName)
	for _, m := range matches {
		fmt.Fprintf(&b, "- %s (%s, %s)\n", m.GetString("name"), m.GetString("type"), m.GetString("city"))
	}
	b.WriteString("Qual delas?")
	return b.String()
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
