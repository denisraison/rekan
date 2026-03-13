package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

// RouteAction executes the action from an AgentResponse and returns the reply text.
// For NEEDS_CONFIRMATION actions, it stores state and returns empty string (BAML reply carries the prompt).
// For EXECUTE actions, it runs the action immediately.
func RouteAction(app core.App, ctx HydratedContext, state *OperatorState, action types.AgentAction, gen content.GenerateFunc) (string, error) {
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

	case types.AgentActionTypePOST_GENERATE:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executePostGenerate(app, ctx, action.ActionParams, gen)

	case types.AgentActionTypePOST_LIST_PENDING:
		return postListPending(ctx, action.ActionParams)

	case types.AgentActionTypePOST_APPROVE:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executePostApprove(app, ctx, action.ActionParams)

	case types.AgentActionTypePOST_REJECT:
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirm(app, state, ctx, string(action.ActionType), action.ActionParams)
		}
		return executePostReject(app, ctx, action.ActionParams)

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
func ExecuteConfirmed(app core.App, ctx HydratedContext, state *OperatorState, gen content.GenerateFunc) (string, error) {
	defer ClearState(app, state, ctx.OperatorJID)

	switch state.ActionType {
	case string(types.AgentActionTypeCUSTOMER_CREATE):
		return executeCustomerCreate(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypeCUSTOMER_UPDATE):
		return executeCustomerUpdate(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypeCUSTOMER_PAUSE):
		return executeCustomerPause(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypePOST_GENERATE):
		return executePostGenerate(app, ctx, state.CollectedFields, gen)
	case string(types.AgentActionTypePOST_APPROVE):
		return executePostApprove(app, ctx, state.CollectedFields)
	case string(types.AgentActionTypePOST_REJECT):
		return executePostReject(app, ctx, state.CollectedFields)
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
	if s := fields["state"]; s != "" {
		record.Set("state", s)
	}
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

func executePostGenerate(app core.App, ctx HydratedContext, fields map[string]string, gen content.GenerateFunc) (string, error) {
	name := fields["name"]
	if name == "" {
		return fmt.Sprintf("%s, pra qual cliente você quer gerar post?", ctx.OperatorName), nil
	}

	matches := findBusinessRecords(ctx.Businesses, name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, name), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	biz := matches[0]
	if gen == nil {
		return fmt.Sprintf("%s, geração de posts não está configurada.", ctx.OperatorName), nil
	}

	result, err := service.GeneratePosts(context.Background(), app, gen, biz.Id)
	if err != nil {
		return "", fmt.Errorf("generating posts: %w", err)
	}

	if len(result.Posts) == 0 {
		return fmt.Sprintf("%s, não consegui gerar post pra %s.", ctx.OperatorName, biz.GetString("name")), nil
	}

	post := result.Posts[0]
	caption := post.Caption
	if len(caption) > 80 {
		caption = caption[:80] + "..."
	}
	return fmt.Sprintf("%s, post gerado pra %s! \"%s\" (%s)", ctx.OperatorName, biz.GetString("name"), caption, post.ID), nil
}

func postListPending(ctx HydratedContext, params map[string]string) (string, error) {
	name := params["name"]

	var pending []*core.Record
	for _, p := range ctx.PendingPosts {
		if name == "" {
			pending = append(pending, p)
			continue
		}
		bizID := p.GetString("business")
		for _, biz := range ctx.Businesses {
			if biz.Id == bizID && strings.Contains(normalizeForMatch(biz.GetString("name")), normalizeForMatch(name)) {
				pending = append(pending, p)
				break
			}
		}
	}

	if len(pending) == 0 {
		if name != "" {
			return fmt.Sprintf("%s, não tem posts pendentes da %s.", ctx.OperatorName, name), nil
		}
		return fmt.Sprintf("%s, não tem posts pendentes no momento.", ctx.OperatorName), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, posts pendentes:\n", ctx.OperatorName)
	for _, p := range pending {
		bizName := businessNameByID(ctx.Businesses, p.GetString("business"))
		caption := p.GetString("caption")
		if len(caption) > 50 {
			caption = caption[:50] + "..."
		}
		fmt.Fprintf(&b, "- %s: \"%s\" (%s)\n", bizName, caption, p.Id)
	}
	return b.String(), nil
}

func executePostApprove(app core.App, ctx HydratedContext, fields map[string]string) (string, error) {
	postID := fields["post_id"]
	if postID == "" {
		return fmt.Sprintf("%s, qual post você quer aprovar?", ctx.OperatorName), nil
	}

	record := findPendingPost(ctx.PendingPosts, postID)
	if record == nil {
		var err error
		record, err = app.FindRecordById(domain.CollPosts, postID)
		if err != nil {
			return fmt.Sprintf("%s, não encontrei o post %s.", ctx.OperatorName, postID), nil
		}
	}

	record.Set("reviewed", true)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("approving post: %w", err)
	}

	bizName := businessNameByID(ctx.Businesses, record.GetString("business"))
	return fmt.Sprintf("%s, post da %s aprovado!", ctx.OperatorName, bizName), nil
}

func executePostReject(app core.App, ctx HydratedContext, fields map[string]string) (string, error) {
	postID := fields["post_id"]
	if postID == "" {
		return fmt.Sprintf("%s, qual post você quer rejeitar?", ctx.OperatorName), nil
	}

	record := findPendingPost(ctx.PendingPosts, postID)
	if record == nil {
		var err error
		record, err = app.FindRecordById(domain.CollPosts, postID)
		if err != nil {
			return fmt.Sprintf("%s, não encontrei o post %s.", ctx.OperatorName, postID), nil
		}
	}

	feedback := fields["feedback"]
	record.Set("reviewed", true)
	record.Set("review_note", feedback)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("rejecting post: %w", err)
	}

	bizName := businessNameByID(ctx.Businesses, record.GetString("business"))
	if feedback != "" {
		return fmt.Sprintf("%s, post da %s rejeitado. Feedback: %s.", ctx.OperatorName, bizName, feedback), nil
	}
	return fmt.Sprintf("%s, post da %s rejeitado.", ctx.OperatorName, bizName), nil
}

// findPendingPost looks up a post by ID in the pre-loaded pending posts slice.
func findPendingPost(posts []*core.Record, id string) *core.Record {
	for _, p := range posts {
		if p.Id == id {
			return p
		}
	}
	return nil
}

// businessNameByID returns the business name for a given record ID,
// or the raw ID as fallback if not found.
func businessNameByID(businesses []*core.Record, id string) string {
	for _, biz := range businesses {
		if biz.Id == id {
			return biz.GetString("name")
		}
	}
	return id
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
