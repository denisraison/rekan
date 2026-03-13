package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const stateExpiry = 10 * time.Minute

const (
	StateIdle       = "idle"
	StateConfirming = "confirming"
)

// OperatorState represents the per-operator confirmation state machine.
type OperatorState struct {
	Record          *core.Record
	State           string            // idle, collecting, confirming
	ActionType      string            // e.g. CUSTOMER_CREATE
	CollectedFields map[string]string // fields gathered so far
	ExpiresAt       time.Time
}

// LoadState loads the current state for an operator, creating an idle record if none exists.
func LoadState(app core.App, operatorJID string) (*OperatorState, error) {
	var records []*core.Record
	app.RecordQuery(domain.CollAgentState).
		AndWhere(dbx.NewExp("operator_jid = {:jid}", dbx.Params{"jid": operatorJID})).
		Limit(1).
		All(&records)

	if len(records) == 0 {
		return &OperatorState{State: StateIdle, CollectedFields: make(map[string]string)}, nil
	}

	record := records[0]
	fields := make(map[string]string)
	if raw := record.GetString("collected_fields"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &fields); err != nil {
			return nil, fmt.Errorf("corrupted agent_state fields for %s: %w", operatorJID, err)
		}
	}

	state := &OperatorState{
		Record:          record,
		State:           record.GetString("state"),
		ActionType:      record.GetString("action_type"),
		CollectedFields: fields,
		ExpiresAt:       record.GetDateTime("expires_at").Time(),
	}

	// Auto-expire stale state
	if state.State != StateIdle && !state.ExpiresAt.IsZero() && time.Now().After(state.ExpiresAt) {
		state.State = StateIdle
		state.ActionType = ""
		state.CollectedFields = make(map[string]string)
		saveState(app, state, operatorJID)
	}

	return state, nil
}

// SetConfirming transitions to confirming state with the given action type and fields.
// Uses the already-loaded state to avoid a redundant DB query.
func SetConfirming(app core.App, state *OperatorState, operatorJID, actionType string, fields map[string]string) error {
	state.State = StateConfirming
	state.ActionType = actionType
	state.CollectedFields = fields
	state.ExpiresAt = time.Now().Add(stateExpiry)
	return saveState(app, state, operatorJID)
}

// ClearState resets the operator to idle.
// Uses the already-loaded state to avoid a redundant DB query.
func ClearState(app core.App, state *OperatorState, operatorJID string) error {
	if state.Record == nil {
		return nil
	}
	state.State = StateIdle
	state.ActionType = ""
	state.CollectedFields = make(map[string]string)
	return saveState(app, state, operatorJID)
}

// HasPendingAction checks if another operator has a pending action on the same entity (by name).
func HasPendingAction(app core.App, excludeJID, entityName string) (string, bool) {
	var records []*core.Record
	app.RecordQuery(domain.CollAgentState).
		AndWhere(dbx.NewExp("operator_jid != {:jid}", dbx.Params{"jid": excludeJID})).
		AndWhere(dbx.NewExp("state != {:state}", dbx.Params{"state": StateIdle})).
		All(&records)

	if len(records) == 0 {
		return "", false
	}

	normalizedEntity := normalizeForMatch(entityName)
	for _, r := range records {
		expiresAt := r.GetDateTime("expires_at").Time()
		if !expiresAt.IsZero() && time.Now().After(expiresAt) {
			continue
		}
		var fields map[string]string
		if raw := r.GetString("collected_fields"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &fields); err != nil {
				continue
			}
		}
		if fields != nil && normalizeForMatch(fields["name"]) == normalizedEntity {
			return fmt.Sprintf("operator %s", r.GetString("operator_jid")), true
		}
	}
	return "", false
}

func saveState(app core.App, state *OperatorState, operatorJID string) error {
	record := state.Record
	if record == nil {
		col, err := app.FindCachedCollectionByNameOrId(domain.CollAgentState)
		if err != nil {
			return fmt.Errorf("agent_state collection: %w", err)
		}
		record = core.NewRecord(col)
		record.Set("operator_jid", operatorJID)
		state.Record = record
	}

	fieldsJSON, _ := json.Marshal(state.CollectedFields)

	record.Set("state", state.State)
	record.Set("action_type", state.ActionType)
	record.Set("collected_fields", string(fieldsJSON))
	if !state.ExpiresAt.IsZero() {
		record.Set("expires_at", state.ExpiresAt.UTC().Format(time.RFC3339))
	}
	return app.Save(record)
}
