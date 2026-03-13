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
// For NEEDS_CONFIRMATION actions, it validates, stores state, and returns a structured confirmation.
// For EXECUTE actions, it runs the action immediately.
func RouteAction(app core.App, ctx HydratedContext, state *OperatorState, action types.AgentAction, gen content.GenerateFunc) (string, error) {
	switch action.ActionType {
	case types.AgentActionTypeSTATUS_OVERVIEW:
		return fmt.Sprintf("%s, temos %d clientes ativas e %d posts gerados.",
			ctx.OperatorName, len(ctx.Businesses), ctx.PostCount), nil

	case types.AgentActionTypeCUSTOMER_LIST:
		return customerList(ctx)

	case types.AgentActionTypeCUSTOMER_INFO:
		p := action.CustomerInfo
		if p == nil {
			return fmt.Sprintf("%s, qual cliente você quer saber?", ctx.OperatorName), nil
		}
		return customerInfo(ctx, p.Name)

	case types.AgentActionTypeCUSTOMER_CREATE:
		p := action.CustomerCreate
		if p == nil {
			return fmt.Sprintf("%s, faltam informações pra cadastrar. Diz nome, tipo e cidade.", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			if err := validateCustomerCreate(p, ctx.OperatorName); err != nil {
				return err.Error(), nil
			}
			if dup := findDuplicate(ctx.Businesses, p.Name); dup != nil {
				return fmt.Sprintf("%s, a %s já existe (%s, %s).",
					ctx.OperatorName, dup.GetString("name"),
					dup.GetString("type"), dup.GetString("city")), nil
			}
			if who, conflict := HasPendingAction(app, ctx.OperatorJID, p.Name); conflict {
				return fmt.Sprintf("%s, outro operador (%s) já tem uma ação pendente pra essa cliente.", ctx.OperatorName, who), nil
			}
			return storeAndConfirmCustomerCreate(app, state, ctx, p)
		}
		return executeCustomerCreate(app, ctx, p)

	case types.AgentActionTypeCUSTOMER_UPDATE:
		p := action.CustomerUpdate
		if p == nil {
			return fmt.Sprintf("%s, qual cliente você quer alterar?", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirmTyped(app, state, ctx, string(action.ActionType), p)
		}
		return executeCustomerUpdate(app, ctx, p)

	case types.AgentActionTypeCUSTOMER_PAUSE:
		p := action.CustomerPause
		if p == nil {
			return fmt.Sprintf("%s, qual cliente você quer pausar?", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirmTyped(app, state, ctx, string(action.ActionType), p)
		}
		return executeCustomerPause(app, ctx, p)

	case types.AgentActionTypePOST_GENERATE:
		p := action.PostGenerate
		if p == nil {
			return fmt.Sprintf("%s, pra qual cliente você quer gerar post?", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			return storeAndConfirmTyped(app, state, ctx, string(action.ActionType), p)
		}
		return executePostGenerate(app, ctx, p, gen)

	case types.AgentActionTypePOST_LIST_PENDING:
		return postListPending(ctx)

	case types.AgentActionTypePOST_APPROVE:
		p := action.PostApprove
		if p == nil {
			return fmt.Sprintf("%s, qual post você quer aprovar?", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			if err := validatePostApprove(p, ctx.OperatorName); err != nil {
				return err.Error(), nil
			}
			return storeAndConfirmTyped(app, state, ctx, string(action.ActionType), p)
		}
		return executePostApprove(app, ctx, p)

	case types.AgentActionTypePOST_REJECT:
		p := action.PostReject
		if p == nil {
			return fmt.Sprintf("%s, qual post você quer rejeitar?", ctx.OperatorName), nil
		}
		if action.ActionStatus == types.AgentActionStatusNEEDS_CONFIRMATION {
			if err := validatePostReject(p, ctx.OperatorName); err != nil {
				return err.Error(), nil
			}
			return storeAndConfirmTyped(app, state, ctx, string(action.ActionType), p)
		}
		return executePostReject(app, ctx, p)

	default:
		return "", fmt.Errorf("unknown action type: %s", action.ActionType)
	}
}

// storeAndConfirmCustomerCreate stores CUSTOMER_CREATE state and returns a structured confirmation message.
func storeAndConfirmCustomerCreate(app core.App, state *OperatorState, ctx HydratedContext, p *types.CustomerCreateParams) (string, error) {
	if err := SetConfirming(app, state, ctx.OperatorJID, string(types.AgentActionTypeCUSTOMER_CREATE), p); err != nil {
		return "", fmt.Errorf("saving state: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s, cadastrar:\n", ctx.OperatorName)
	fmt.Fprintf(&b, "- Nome: %s\n", p.Name)
	fmt.Fprintf(&b, "- Tipo: %s\n", p.Type)
	fmt.Fprintf(&b, "- Cidade: %s\n", p.City)
	if p.Phone != nil {
		fmt.Fprintf(&b, "- Tel: %s\n", *p.Phone)
	}
	if p.TargetAudience != nil {
		fmt.Fprintf(&b, "- Público: %s\n", *p.TargetAudience)
	}
	if p.BrandVibe != nil {
		fmt.Fprintf(&b, "- Vibe: %s\n", *p.BrandVibe)
	}
	if p.Quirks != nil {
		fmt.Fprintf(&b, "- Obs: %s\n", *p.Quirks)
	}
	b.WriteString("Confirma?")
	return b.String(), nil
}

// storeAndConfirmTyped stores state for non-create actions and returns empty string (BAML reply carries the prompt).
func storeAndConfirmTyped(app core.App, state *OperatorState, ctx HydratedContext, actionType string, params any) (string, error) {
	if err := SetConfirming(app, state, ctx.OperatorJID, actionType, params); err != nil {
		return "", fmt.Errorf("saving state: %w", err)
	}
	return "", nil
}

// ExecuteConfirmed runs the pending action after the operator said "sim".
func ExecuteConfirmed(app core.App, ctx HydratedContext, state *OperatorState, gen content.GenerateFunc) (string, error) {
	defer func() {
		if err := ClearState(app, state, ctx.OperatorJID); err != nil {
			app.Logger().Error("agent: clear state after confirmed action", "error", err)
		}
	}()

	switch state.ActionType {
	case string(types.AgentActionTypeCUSTOMER_CREATE):
		var p types.CustomerCreateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executeCustomerCreate(app, ctx, &p)
	case string(types.AgentActionTypeCUSTOMER_UPDATE):
		var p types.CustomerUpdateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executeCustomerUpdate(app, ctx, &p)
	case string(types.AgentActionTypeCUSTOMER_PAUSE):
		var p types.CustomerPauseParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executeCustomerPause(app, ctx, &p)
	case string(types.AgentActionTypePOST_GENERATE):
		var p types.PostGenerateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executePostGenerate(app, ctx, &p, gen)
	case string(types.AgentActionTypePOST_APPROVE):
		var p types.PostApproveParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executePostApprove(app, ctx, &p)
	case string(types.AgentActionTypePOST_REJECT):
		var p types.PostRejectParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return fmt.Sprintf("%s, algo deu errado com os dados. Pode repetir?", ctx.OperatorName), nil
		}
		return executePostReject(app, ctx, &p)
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

func customerInfo(ctx HydratedContext, name string) (string, error) {
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

func executeCustomerCreate(app core.App, ctx HydratedContext, p *types.CustomerCreateParams) (string, error) {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		return "", fmt.Errorf("businesses collection: %w", err)
	}
	record := core.NewRecord(col)
	record.Set("name", p.Name)
	record.Set("type", p.Type)
	record.Set("city", p.City)
	if p.Phone != nil {
		record.Set("phone", *p.Phone)
	}
	if p.TargetAudience != nil {
		record.Set("target_audience", *p.TargetAudience)
	}
	if p.BrandVibe != nil {
		record.Set("brand_vibe", *p.BrandVibe)
	}
	if p.Quirks != nil {
		record.Set("quirks", *p.Quirks)
	}
	record.Set("invite_status", domain.InviteStatusDraft)

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("creating business: %w", err)
	}

	return fmt.Sprintf("%s, %s cadastrada! (%s, %s)",
		ctx.OperatorName, p.Name, p.Type, p.City), nil
}

func executeCustomerUpdate(app core.App, ctx HydratedContext, p *types.CustomerUpdateParams) (string, error) {
	matches := findBusinessRecords(ctx.Businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, p.Name), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	record := matches[0]
	var updated []string
	if p.Type != nil {
		record.Set("type", *p.Type)
		updated = append(updated, "tipo")
	}
	if p.City != nil {
		record.Set("city", *p.City)
		updated = append(updated, "cidade")
	}
	if p.Phone != nil {
		record.Set("phone", *p.Phone)
		updated = append(updated, "telefone")
	}
	if p.TargetAudience != nil {
		record.Set("target_audience", *p.TargetAudience)
		updated = append(updated, "público")
	}
	if p.BrandVibe != nil {
		record.Set("brand_vibe", *p.BrandVibe)
		updated = append(updated, "vibe")
	}
	if p.Quirks != nil {
		record.Set("quirks", *p.Quirks)
		updated = append(updated, "obs")
	}

	if len(updated) == 0 {
		return fmt.Sprintf("%s, nenhum campo pra atualizar na %s.", ctx.OperatorName, p.Name), nil
	}

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("updating business: %w", err)
	}

	return fmt.Sprintf("%s, %s atualizada! Campos: %s.",
		ctx.OperatorName, p.Name, strings.Join(updated, ", ")), nil
}

func executeCustomerPause(app core.App, ctx HydratedContext, p *types.CustomerPauseParams) (string, error) {
	matches := findBusinessRecords(ctx.Businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, p.Name), nil
	}
	if len(matches) > 1 {
		return disambiguate(ctx.OperatorName, matches), nil
	}

	record := matches[0]
	record.Set("invite_status", domain.InviteStatusCancelled)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("pausing business: %w", err)
	}

	if p.Reason != nil && *p.Reason != "" {
		return fmt.Sprintf("%s, %s pausada. Motivo: %s.", ctx.OperatorName, p.Name, *p.Reason), nil
	}
	return fmt.Sprintf("%s, %s pausada.", ctx.OperatorName, p.Name), nil
}

func executePostGenerate(app core.App, ctx HydratedContext, p *types.PostGenerateParams, gen content.GenerateFunc) (string, error) {
	if p.Name == "" {
		return fmt.Sprintf("%s, pra qual cliente você quer gerar post?", ctx.OperatorName), nil
	}

	matches := findBusinessRecords(ctx.Businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", ctx.OperatorName, p.Name), nil
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
	return fmt.Sprintf("%s, post gerado pra %s! \"%s\" (%s)", ctx.OperatorName, biz.GetString("name"), truncate(post.Caption, 80), post.ID), nil
}

func postListPending(ctx HydratedContext) (string, error) {
	if len(ctx.PendingPosts) == 0 {
		return fmt.Sprintf("%s, não tem posts pendentes no momento.", ctx.OperatorName), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, posts pendentes:\n", ctx.OperatorName)
	for _, p := range ctx.PendingPosts {
		bizName := businessNameByID(ctx.Businesses, p.GetString("business"))
		fmt.Fprintf(&b, "- %s: \"%s\" (%s)\n", bizName, truncate(p.GetString("caption"), 50), p.Id)
	}
	return b.String(), nil
}

func executePostApprove(app core.App, ctx HydratedContext, p *types.PostApproveParams) (string, error) {
	record := findPendingPost(ctx.PendingPosts, p.PostId)
	if record == nil {
		var err error
		record, err = app.FindRecordById(domain.CollPosts, p.PostId)
		if err != nil {
			return fmt.Sprintf("%s, não encontrei o post %s.", ctx.OperatorName, p.PostId), nil
		}
	}

	record.Set("reviewed", true)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("approving post: %w", err)
	}

	bizName := businessNameByID(ctx.Businesses, record.GetString("business"))
	return fmt.Sprintf("%s, post da %s aprovado!", ctx.OperatorName, bizName), nil
}

func executePostReject(app core.App, ctx HydratedContext, p *types.PostRejectParams) (string, error) {
	record := findPendingPost(ctx.PendingPosts, p.PostId)
	if record == nil {
		var err error
		record, err = app.FindRecordById(domain.CollPosts, p.PostId)
		if err != nil {
			return fmt.Sprintf("%s, não encontrei o post %s.", ctx.OperatorName, p.PostId), nil
		}
	}

	record.Set("reviewed", true)
	record.Set("review_note", p.Feedback)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("rejecting post: %w", err)
	}

	bizName := businessNameByID(ctx.Businesses, record.GetString("business"))
	if p.Feedback != "" {
		return fmt.Sprintf("%s, post da %s rejeitado. Feedback: %s.", ctx.OperatorName, bizName, p.Feedback), nil
	}
	return fmt.Sprintf("%s, post da %s rejeitado.", ctx.OperatorName, bizName), nil
}

// findDuplicate checks if a business with the same name already exists.
func findDuplicate(businesses []*core.Record, name string) *core.Record {
	normalized := normalizeForMatch(name)
	for _, biz := range businesses {
		if normalizeForMatch(biz.GetString("name")) == normalized {
			return biz
		}
	}
	return nil
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
func LogAction(app core.App, operatorName, operatorJID, actionType string, params any, result string, success bool, start time.Time) {
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
	if err := app.Save(record); err != nil {
		app.Logger().Error("agent: save action log", "error", err)
	}
}
