package agent

import (
	"fmt"
	"strings"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// HydratedContext holds the pre-queried data for both the BAML prompt and the action router.
type HydratedContext struct {
	OperatorName string
	OperatorRole string
	Businesses   []*core.Record
	PostCount    int64
	Formatted    string
}

// HydrateContext builds context for the BAML agent prompt and caches query results
// so the action router can reuse them without re-querying.
func HydrateContext(app core.App, operatorName, operatorRole string) HydratedContext {
	ctx := HydratedContext{
		OperatorName: operatorName,
		OperatorRole: operatorRole,
	}

	// Active businesses
	ctx.Businesses, _ = app.FindRecordsByFilter(
		domain.CollBusinesses,
		"invite_status = 'active'",
		"name",
		0, 0, nil,
	)

	// Post count (cheap COUNT query instead of loading all records)
	ctx.PostCount, _ = app.CountRecords(domain.CollPosts)

	// Recent agent actions
	actions, _ := app.FindRecordsByFilter(
		domain.CollAgentActionLog,
		"1=1",
		"-created",
		0, 5, nil,
	)

	// Format for BAML prompt
	var b strings.Builder
	fmt.Fprintf(&b, "Operadora: %s", operatorName)
	if operatorRole != "" {
		fmt.Fprintf(&b, " (%s)", operatorRole)
	}
	b.WriteByte('\n')

	fmt.Fprintf(&b, "\nClientes ativas: %d\n", len(ctx.Businesses))
	for _, biz := range ctx.Businesses {
		fmt.Fprintf(&b, "  - %s (%s, %s)\n", biz.GetString("name"), biz.GetString("type"), biz.GetString("city"))
	}

	fmt.Fprintf(&b, "\nPosts gerados (total): %d\n", ctx.PostCount)

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
