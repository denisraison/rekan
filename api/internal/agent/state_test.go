package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "github.com/denisraison/rekan/api/migrations"
)

func setupTestApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Cleanup() })
	return app
}

func TestLoadState_NoExisting(t *testing.T) {
	app := setupTestApp(t)
	state, err := LoadState(app, "5511999990000")
	if err != nil {
		t.Fatal(err)
	}
	if state.State != StateIdle {
		t.Errorf("expected idle, got %s", state.State)
	}
}

func TestSetConfirming_And_ClearState(t *testing.T) {
	app := setupTestApp(t)
	jid := "5511999990000"

	state, _ := LoadState(app, jid)
	fields := map[string]string{"name": "Patricia", "type": "Salão de Beleza", "city": "Belo Horizonte"}
	if err := SetConfirming(app, state, jid, "CUSTOMER_CREATE", fields); err != nil {
		t.Fatal(err)
	}

	state, _ = LoadState(app, jid)
	if state.State != StateConfirming {
		t.Errorf("expected confirming, got %s", state.State)
	}
	if state.ActionType != "CUSTOMER_CREATE" {
		t.Errorf("expected CUSTOMER_CREATE, got %s", state.ActionType)
	}
	if !strings.Contains(string(state.CollectedFields), `"name":"Patricia"`) {
		t.Errorf("expected CollectedFields to contain Patricia, got %s", string(state.CollectedFields))
	}

	if err := ClearState(app, state, jid); err != nil {
		t.Fatal(err)
	}
	state, _ = LoadState(app, jid)
	if state.State != StateIdle {
		t.Errorf("expected idle after clear, got %s", state.State)
	}
}

func TestState_AutoExpiry(t *testing.T) {
	app := setupTestApp(t)
	jid := "5511999990000"

	col, _ := app.FindCachedCollectionByNameOrId("agent_state")
	record := core.NewRecord(col)
	record.Set("operator_jid", jid)
	record.Set("state", StateConfirming)
	record.Set("action_type", "CUSTOMER_CREATE")
	record.Set("collected_fields", `{"name":"Test"}`)
	record.Set("expires_at", time.Now().Add(-1*time.Minute).UTC().Format(time.RFC3339))
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	state, _ := LoadState(app, jid)
	if state.State != StateIdle {
		t.Errorf("expected idle (auto-expired), got %s", state.State)
	}
}

func TestPerOperatorIsolation(t *testing.T) {
	app := setupTestApp(t)
	jid1 := "5511999990000"
	jid2 := "5511999991111"

	state1, _ := LoadState(app, jid1)
	_ = SetConfirming(app, state1, jid1, "CUSTOMER_CREATE", map[string]string{"name": "Patricia"})

	state2, _ := LoadState(app, jid2)
	_ = SetConfirming(app, state2, jid2, "CUSTOMER_PAUSE", map[string]string{"name": "Maria"})

	state1, _ = LoadState(app, jid1)
	state2, _ = LoadState(app, jid2)

	if state1.ActionType != "CUSTOMER_CREATE" {
		t.Errorf("operator 1 expected CUSTOMER_CREATE, got %s", state1.ActionType)
	}
	if state2.ActionType != "CUSTOMER_PAUSE" {
		t.Errorf("operator 2 expected CUSTOMER_PAUSE, got %s", state2.ActionType)
	}

	_ = ClearState(app, state1, jid1)
	state1, _ = LoadState(app, jid1)
	state2, _ = LoadState(app, jid2)

	if state1.State != StateIdle {
		t.Errorf("operator 1 expected idle after clear, got %s", state1.State)
	}
	if state2.State != StateConfirming {
		t.Errorf("operator 2 should still be confirming, got %s", state2.State)
	}
}

func TestHasPendingAction_Conflict(t *testing.T) {
	app := setupTestApp(t)
	jid1 := "5511999990000"
	jid2 := "5511999991111"

	state, _ := LoadState(app, jid1)
	_ = SetConfirming(app, state, jid1, "CUSTOMER_CREATE", map[string]string{"name": "Patricia"})

	_, conflict := HasPendingAction(app, jid2, "Patricia")
	if !conflict {
		t.Error("expected conflict for Patricia")
	}

	_, conflict = HasPendingAction(app, jid2, "Maria")
	if conflict {
		t.Error("unexpected conflict for Maria")
	}

	_, conflict = HasPendingAction(app, jid1, "Patricia")
	if conflict {
		t.Error("should not conflict with own pending action")
	}
}
