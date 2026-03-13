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
	State           string           // idle, confirming
	ActionType      string           // e.g. CUSTOMER_CREATE
	CollectedFields json.RawMessage  // typed params struct as JSON
	ExpiresAt       time.Time
}

// LoadState loads the current state for an operator, creating an idle record if none exists.
func LoadState(app core.App, operatorJID string) (*OperatorState, error) {
	var records []*core.Record
	if err := app.RecordQuery(domain.CollAgentState).
		AndWhere(dbx.NewExp("operator_jid = {:jid}", dbx.Params{"jid": operatorJID})).
		Limit(1).
		All(&records); err != nil {
		return &OperatorState{State: StateIdle}, err
	}

	if len(records) == 0 {
		return &OperatorState{State: StateIdle}, nil
	}

	record := records[0]
	state := &OperatorState{
		Record:     record,
		State:      record.GetString("state"),
		ActionType: record.GetString("action_type"),
		ExpiresAt:  record.GetDateTime("expires_at").Time(),
	}

	if raw := record.GetString("collected_fields"); raw != "" {
		state.CollectedFields = json.RawMessage(raw)
	}

	// Auto-expire stale state
	if state.State != StateIdle && !state.ExpiresAt.IsZero() && time.Now().After(state.ExpiresAt) {
		state.State = StateIdle
		state.ActionType = ""
		state.CollectedFields = nil
		if err := saveState(app, state, operatorJID); err != nil {
			app.Logger().Error("agent: auto-expire state", "error", err)
		}
	}

	return state, nil
}

// SetConfirming transitions to confirming state with the given action type and typed params.
// Uses the already-loaded state to avoid a redundant DB query.
func SetConfirming(app core.App, state *OperatorState, operatorJID, actionType string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	state.State = StateConfirming
	state.ActionType = actionType
	state.CollectedFields = raw
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
	state.CollectedFields = nil
	return saveState(app, state, operatorJID)
}

// HasPendingAction checks if another operator has a pending action on the same entity (by name).
func HasPendingAction(app core.App, excludeJID, entityName string) (string, bool) {
	var records []*core.Record
	if err := app.RecordQuery(domain.CollAgentState).
		AndWhere(dbx.NewExp("operator_jid != {:jid}", dbx.Params{"jid": excludeJID})).
		AndWhere(dbx.NewExp("state != {:state}", dbx.Params{"state": StateIdle})).
		AndWhere(dbx.NewExp("expires_at > {:now}", dbx.Params{"now": time.Now().UTC().Format("2006-01-02 15:04:05.000Z")})).
		All(&records); err != nil {
		return "", false
	}

	normalizedEntity := normalizeForMatch(entityName)
	for _, r := range records {
		raw := r.GetString("collected_fields")
		if raw == "" {
			continue
		}
		var nameHolder struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(raw), &nameHolder); err != nil {
			continue
		}
		if normalizeForMatch(nameHolder.Name) == normalizedEntity {
			return fmt.Sprintf("operator %s", r.GetString("operator_jid")), true
		}
	}
	return "", false
}

// unmarshalCollectedFields deserializes the collected_fields JSON into a typed struct.
func unmarshalCollectedFields(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty collected_fields")
	}
	return json.Unmarshal(raw, dst)
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

	record.Set("state", state.State)
	record.Set("action_type", state.ActionType)
	if len(state.CollectedFields) > 0 {
		record.Set("collected_fields", string(state.CollectedFields))
	} else {
		record.Set("collected_fields", "")
	}
	if !state.ExpiresAt.IsZero() {
		record.Set("expires_at", state.ExpiresAt.UTC().Format("2006-01-02 15:04:05.000Z"))
	}
	return app.Save(record)
}
