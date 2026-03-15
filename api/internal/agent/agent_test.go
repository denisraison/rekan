package agent_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/denisraison/rekan/api/internal/agent"
	"github.com/denisraison/rekan/api/internal/domain"
	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func newTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Cleanup() })
	return app
}

// TestStructuredStorage verifies that messages are stored with structured JSON.
func TestStructuredStorage(t *testing.T) {
	app := newTestApp(t)

	userJSON := `{"role":"user","content":[{"type":"text","text":"oi"}]}`
	if err := agent.StoreMessage(app, "Elenice", "5511999990000", "user", "oi", "", userJSON); err != nil {
		t.Fatal(err)
	}
	assistantJSON := `{"role":"assistant","content":[{"type":"text","text":"oi Elenice!"}]}`
	if err := agent.StoreMessage(app, "Rekan", "", "assistant", "oi Elenice!", "", assistantJSON); err != nil {
		t.Fatal(err)
	}

	records, err := app.FindAllRecords(domain.CollAgentConversations)
	if err != nil {
		t.Fatal(err)
	}

	var userStructured, assistantStructured int
	for _, r := range records {
		s := r.GetString("structured")
		if s == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Errorf("invalid structured JSON: %v", err)
			continue
		}
		role, _ := m["role"].(string)
		switch role {
		case "user":
			userStructured++
		case "assistant":
			assistantStructured++
		}
	}

	if userStructured == 0 {
		t.Error("expected at least one user message with structured data")
	}
	if assistantStructured == 0 {
		t.Error("expected at least one assistant message with structured data")
	}
}

// TestBuildClaudeMessages_StructuredToolUse verifies that structured messages with
// tool_use blocks are deserialized and passed to buildClaudeMessages correctly.
func TestBuildClaudeMessages_StructuredToolUse(t *testing.T) {
	app := newTestApp(t)

	userJSON := `{"role":"user","content":[{"type":"text","text":"busca a Nika"}]}`
	if err := agent.StoreMessage(app, "Elenice", "5511999990000", "user", "busca a Nika", "", userJSON); err != nil {
		t.Fatal(err)
	}

	assistantJSON := `{"role":"assistant","content":[{"type":"text","text":"Deixa eu verificar..."},{"type":"tool_use","id":"toolu_xxx","name":"search_customers","input":{"query":"Nika"}}]}`
	if err := agent.StoreMessage(app, "Rekan", "", "assistant", "Deixa eu verificar...", "", assistantJSON); err != nil {
		t.Fatal(err)
	}

	toolResultJSON := `{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_xxx","content":"Nome: Nika\nTipo: Moda"}]}`
	if err := agent.StoreMessage(app, "Rekan", "", "user", "", "", toolResultJSON); err != nil {
		t.Fatal(err)
	}

	replyJSON := `{"role":"assistant","content":[{"type":"text","text":"Encontrei a Nika!"}]}`
	if err := agent.StoreMessage(app, "Rekan", "", "assistant", "Encontrei a Nika!", "", replyJSON); err != nil {
		t.Fatal(err)
	}

	msgs, err := agent.LoadRecent(app, 15)
	if err != nil {
		t.Fatal(err)
	}

	var hasStructured int
	for _, m := range msgs {
		if m.Structured != "" {
			hasStructured++
		}
	}
	if hasStructured != 4 {
		t.Errorf("expected 4 messages with structured data, got %d", hasStructured)
	}

	foundToolUse := false
	for _, m := range msgs {
		if strings.Contains(m.Structured, `"type":"tool_use"`) {
			foundToolUse = true
			if !strings.Contains(m.Structured, "search_customers") {
				t.Errorf("tool_use block lost tool name after storage, got: %s", m.Structured)
			}
			break
		}
	}
	if !foundToolUse {
		for i, m := range msgs {
			t.Logf("msg[%d] role=%s structured=%q", i, m.Role, m.Structured)
		}
		t.Error("no message with tool_use found in structured data")
	}
}

// TestOldMessagesLoadWithoutStructured verifies backward compatibility.
func TestOldMessagesLoadWithoutStructured(t *testing.T) {
	app := newTestApp(t)

	if err := agent.StoreMessage(app, "Elenice", "5511999990000", "user", "oi", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := agent.StoreMessage(app, "Rekan", "", "assistant", "oi Elenice!", "", ""); err != nil {
		t.Fatal(err)
	}

	msgs, err := agent.LoadRecent(app, 15)
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	var foundUser, foundAssistant bool
	for _, m := range msgs {
		switch {
		case m.Role == "user" && m.Content == "oi":
			foundUser = true
		case m.Role == "assistant" && m.Content == "oi Elenice!":
			foundAssistant = true
		}
	}
	if !foundUser {
		t.Error("user message not found in loaded messages")
	}
	if !foundAssistant {
		t.Error("assistant message not found in loaded messages")
	}
}
