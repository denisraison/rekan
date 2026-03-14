package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// Action type constants for logging.
const (
	ActionCustomerCreate = "CUSTOMER_CREATE"
	ActionCustomerUpdate = "CUSTOMER_UPDATE"
	ActionCustomerPause  = "CUSTOMER_PAUSE"
	ActionPostGenerate   = "POST_GENERATE"
	ActionPostApprove    = "POST_APPROVE"
	ActionPostReject     = "POST_REJECT"
)

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
