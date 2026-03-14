package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Action type constants for the confirmation state machine.
const (
	ActionCustomerCreate = "CUSTOMER_CREATE"
	ActionCustomerUpdate = "CUSTOMER_UPDATE"
	ActionCustomerPause  = "CUSTOMER_PAUSE"
	ActionPostGenerate   = "POST_GENERATE"
	ActionPostApprove    = "POST_APPROVE"
	ActionPostReject     = "POST_REJECT"
)

// ExecuteConfirmed runs the pending action after the operator said "sim".
func ExecuteConfirmed(ctx context.Context, app core.App, operatorName string, state *OperatorState, gen content.GenerateFunc) (string, error) {
	defer func() {
		if err := ClearState(app, state, state.Record.GetString("operator_jid")); err != nil {
			app.Logger().Error("agent: clear state after confirmed action", "error", err)
		}
	}()

	switch state.ActionType {
	case ActionCustomerCreate:
		var p CustomerCreateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executeCustomerCreate(app, operatorName, &p)
	case ActionCustomerUpdate:
		var p CustomerUpdateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executeCustomerUpdate(app, operatorName, &p)
	case ActionCustomerPause:
		var p CustomerPauseParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executeCustomerPause(app, operatorName, &p)
	case ActionPostGenerate:
		var p PostGenerateParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executePostGenerate(ctx, app, operatorName, &p, gen)
	case ActionPostApprove:
		var p PostApproveParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executePostApprove(app, operatorName, &p)
	case ActionPostReject:
		var p PostRejectParams
		if err := unmarshalCollectedFields(state.CollectedFields, &p); err != nil {
			return operatorName + ", algo deu errado com os dados. Pode repetir?", nil
		}
		return executePostReject(app, operatorName, &p)
	default:
		return "", fmt.Errorf("unknown pending action: %s", state.ActionType)
	}
}

// loadActiveBusinesses queries active and draft businesses.
func loadActiveBusinesses(app core.App) []*core.Record {
	var businesses []*core.Record
	if err := app.RecordQuery(domain.CollBusinesses).
		AndWhere(dbx.NewExp("invite_status IN ('active', 'draft')")).
		OrderBy("name ASC").
		All(&businesses); err != nil {
		return nil
	}
	return businesses
}

func executeCustomerCreate(app core.App, operatorName string, p *CustomerCreateParams) (string, error) {
	col, err := app.FindCachedCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		return "", fmt.Errorf("businesses collection: %w", err)
	}
	record := core.NewRecord(col)
	record.Set("name", p.Name)
	record.Set("type", p.Type)
	record.Set("city", p.City)
	record.Set("phone", p.Phone)
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
		operatorName, p.Name, p.Type, p.City), nil
}

func executeCustomerUpdate(app core.App, operatorName string, p *CustomerUpdateParams) (string, error) {
	businesses := loadActiveBusinesses(app)
	matches := findBusinessRecords(businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", operatorName, p.Name), nil
	}
	if len(matches) > 1 {
		return disambiguate(operatorName, matches), nil
	}

	record := matches[0]
	var updated []string
	if p.NewName != nil {
		record.Set("name", *p.NewName)
		updated = append(updated, "nome")
	}
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
		return fmt.Sprintf("%s, nenhum campo pra atualizar na %s.", operatorName, p.Name), nil
	}

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("updating business: %w", err)
	}

	displayName := p.Name
	if p.NewName != nil {
		displayName = *p.NewName
	}
	return fmt.Sprintf("%s, %s atualizada! Campos: %s.",
		operatorName, displayName, strings.Join(updated, ", ")), nil
}

func executeCustomerPause(app core.App, operatorName string, p *CustomerPauseParams) (string, error) {
	businesses := loadActiveBusinesses(app)
	matches := findBusinessRecords(businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", operatorName, p.Name), nil
	}
	if len(matches) > 1 {
		return disambiguate(operatorName, matches), nil
	}

	record := matches[0]
	record.Set("invite_status", domain.InviteStatusCancelled)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("pausing business: %w", err)
	}

	if p.Reason != nil && *p.Reason != "" {
		return fmt.Sprintf("%s, %s pausada. Motivo: %s.", operatorName, p.Name, *p.Reason), nil
	}
	return fmt.Sprintf("%s, %s pausada.", operatorName, p.Name), nil
}

func executePostGenerate(ctx context.Context, app core.App, operatorName string, p *PostGenerateParams, gen content.GenerateFunc) (string, error) {
	if p.Name == "" {
		return operatorName + ", pra qual cliente você quer gerar post?", nil
	}

	businesses := loadActiveBusinesses(app)
	matches := findBusinessRecords(businesses, p.Name)
	if len(matches) == 0 {
		return fmt.Sprintf("%s, não encontrei cliente '%s'.", operatorName, p.Name), nil
	}
	if len(matches) > 1 {
		return disambiguate(operatorName, matches), nil
	}

	biz := matches[0]
	if gen == nil {
		return operatorName + ", geração de posts não está configurada.", nil
	}

	result, err := service.GeneratePosts(ctx, app, gen, biz.Id)
	if err != nil {
		return "", fmt.Errorf("generating posts: %w", err)
	}

	if len(result.Posts) == 0 {
		return fmt.Sprintf("%s, não consegui gerar post pra %s.", operatorName, biz.GetString("name")), nil
	}

	post := result.Posts[0]
	var b strings.Builder
	fmt.Fprintf(&b, "%s, post gerado pra %s!\n\n", operatorName, biz.GetString("name"))
	appendPostFields(&b, post.Caption, post.Hashtags, post.ProductionNote)
	fmt.Fprintf(&b, "ID: %s", post.ID)
	return b.String(), nil
}

func executePostApprove(app core.App, operatorName string, p *PostApproveParams) (string, error) {
	record, err := app.FindRecordById(domain.CollPosts, p.PostId)
	if err != nil {
		return fmt.Sprintf("%s, não encontrei o post %s.", operatorName, p.PostId), nil //nolint:nilerr // not-found is a user-facing reply, not an error
	}

	record.Set("reviewed", true)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("approving post: %w", err)
	}

	bizName := resolveBusinessName(app, record, p.Name)
	return fmt.Sprintf("%s, post da %s aprovado!", operatorName, bizName), nil
}

func executePostReject(app core.App, operatorName string, p *PostRejectParams) (string, error) {
	record, err := app.FindRecordById(domain.CollPosts, p.PostId)
	if err != nil {
		return fmt.Sprintf("%s, não encontrei o post %s.", operatorName, p.PostId), nil //nolint:nilerr // not-found is a user-facing reply, not an error
	}

	record.Set("reviewed", true)
	record.Set("review_note", p.Feedback)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("rejecting post: %w", err)
	}

	bizName := resolveBusinessName(app, record, p.Name)
	if p.Feedback != "" {
		return fmt.Sprintf("%s, post da %s rejeitado. Feedback: %s.", operatorName, bizName, p.Feedback), nil
	}
	return fmt.Sprintf("%s, post da %s rejeitado.", operatorName, bizName), nil
}

// resolveBusinessName looks up the business name from a post record, falling back to the given name.
func resolveBusinessName(app core.App, postRecord *core.Record, fallback string) string {
	if bizID := postRecord.GetString("business"); bizID != "" {
		if biz, err := app.FindRecordById(domain.CollBusinesses, bizID); err == nil {
			return biz.GetString("name")
		}
	}
	return fallback
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
