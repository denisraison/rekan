package agent

import (
	"fmt"
	"strings"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// HydratedContext holds the pre-queried data for both the BAML prompt and the action router.
type HydratedContext struct {
	OperatorName string
	OperatorJID  string
	Businesses   []*core.Record
	PendingPosts []*core.Record // posts where reviewed = false
	PostCount    int64
	Formatted    string
}

// HydrateContext builds context for the BAML agent prompt and caches query results
// so the action router can reuse them without re-querying.
func HydrateContext(app core.App, operatorName, operatorJID string) HydratedContext {
	ctx := HydratedContext{
		OperatorName: operatorName,
		OperatorJID:  operatorJID,
	}

	// Active businesses
	if err := app.RecordQuery(domain.CollBusinesses).
		AndWhere(dbx.NewExp("invite_status = 'active'")).
		OrderBy("name ASC").
		All(&ctx.Businesses); err != nil {
		return ctx
	}

	// Post count
	if count, err := app.CountRecords(domain.CollPosts); err != nil {
		app.Logger().Error("agent: count posts", "error", err)
	} else {
		ctx.PostCount = count
	}

	// Pending posts (not yet reviewed, newest first, max 20)
	if err := app.RecordQuery(domain.CollPosts).
		AndWhere(dbx.NewExp("reviewed = FALSE OR reviewed = ''")).
		OrderBy("created DESC").
		Limit(20).
		All(&ctx.PendingPosts); err != nil {
		app.Logger().Error("agent: query pending posts", "error", err)
	}

	// Recent agent actions (last 5)
	var actions []*core.Record
	if err := app.RecordQuery(domain.CollAgentActionLog).
		OrderBy("created DESC").
		Limit(5).
		All(&actions); err != nil {
		app.Logger().Error("agent: query recent actions", "error", err)
	}

	// Format for BAML prompt
	var b strings.Builder
	fmt.Fprintf(&b, "Operadora: %s\n", operatorName)

	fmt.Fprintf(&b, "\nClientes ativas: %d\n", len(ctx.Businesses))
	for _, biz := range ctx.Businesses {
		fmt.Fprintf(&b, "  - %s (%s, %s)\n", biz.GetString("name"), biz.GetString("type"), biz.GetString("city"))
	}

	fmt.Fprintf(&b, "\nPosts gerados (total): %d\n", ctx.PostCount)

	if len(ctx.PendingPosts) > 0 {
		fmt.Fprintf(&b, "\nPosts pendentes de revisão: %d\n", len(ctx.PendingPosts))
		for _, p := range ctx.PendingPosts {
			bizName := businessNameByID(ctx.Businesses, p.GetString("business"))
			caption := p.GetString("caption")
			if len(caption) > 60 {
				caption = caption[:60] + "..."
			}
			fmt.Fprintf(&b, "  - %s: \"%s\" (%s)\n", bizName, caption, p.Id)
		}
	}

	if len(actions) > 0 {
		b.WriteString("\nÚltimas ações:\n")
		for _, a := range actions {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n",
				a.GetString("operator_name"),
				a.GetString("action_type"),
				a.GetString("result"),
			)
		}
	}

	ctx.Formatted = b.String()
	return ctx
}
