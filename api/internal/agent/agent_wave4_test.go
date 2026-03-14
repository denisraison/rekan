package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/denisraison/rekan/api/internal/domain"

	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newWave4TestApp(t *testing.T) core.App {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Cleanup() })
	return app
}

func wave4SeedBusiness(t *testing.T, app core.App, name, bizType, city string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Set("name", name)
	record.Set("type", bizType)
	record.Set("city", city)
	record.Set("invite_status", domain.InviteStatusActive)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func wave4SeedPost(t *testing.T, app core.App, businessID, caption string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(domain.CollPosts)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Set("business", businessID)
	record.Set("caption", caption)
	record.Set("hashtags", []string{"#test"})
	record.Set("production_note", "test")
	record.Set("role", "bastidor")
	record.Set("hook", "test hook")
	record.Set("reviewed", false)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func newExecutor(t *testing.T, app core.App) *ToolExecutor {
	t.Helper()
	return &ToolExecutor{
		Ctx: context.Background(),
		App: app,
	}
}

func callTool(t *testing.T, te *ToolExecutor, name string, args any, operatorName string) toolResult {
	t.Helper()
	input, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return te.executeTool(name, json.RawMessage(input), operatorName)
}

// TestCustomerCreate_HappyPath: call create_customer with confirmed=true, verify DB record.
func TestCustomerCreate_HappyPath(t *testing.T) {
	app := newWave4TestApp(t)
	te := newExecutor(t, app)

	result := callTool(t, te, "create_customer", map[string]any{
		"name":      "Ana",
		"type":      "Manicure",
		"city":      "Goiania",
		"phone":     "62999990000",
		"confirmed": true,
	}, "Elenice")

	if !strings.Contains(result.Text, "cadastrada") {
		t.Errorf("expected 'cadastrada' in result, got: %s", result.Text)
	}

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	var biz *core.Record
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			biz = b
			break
		}
	}
	if biz == nil {
		t.Fatal("business 'Ana' not found in DB")
	}
	if biz.GetString("type") != "Manicure" {
		t.Errorf("type: got %q, want %q", biz.GetString("type"), "Manicure")
	}
	if biz.GetString("city") != "Goiania" {
		t.Errorf("city: got %q, want %q", biz.GetString("city"), "Goiania")
	}
}

// TestCustomerCreate_Preview: confirmed=false returns preview, no DB record.
func TestCustomerCreate_Preview(t *testing.T) {
	app := newWave4TestApp(t)
	te := newExecutor(t, app)

	result := callTool(t, te, "create_customer", map[string]any{
		"name":      "Ana",
		"type":      "Manicure",
		"city":      "Goiania",
		"phone":     "62999990000",
		"confirmed": false,
	}, "Elenice")

	if !result.IsPreview {
		t.Error("expected IsPreview=true for confirmed=false")
	}
	if !strings.Contains(result.Text, "Preview") {
		t.Errorf("expected 'Preview' in result, got: %s", result.Text)
	}

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			t.Error("business 'Ana' should not exist after preview")
		}
	}
}

// TestCustomerUpdate_HappyPath: seed business, call update with confirmed=true, verify DB.
func TestCustomerUpdate_HappyPath(t *testing.T) {
	app := newWave4TestApp(t)
	wave4SeedBusiness(t, app, "Patricia", "Salão", "BH")
	te := newExecutor(t, app)

	result := callTool(t, te, "update_customer", map[string]any{
		"name":      "Patricia",
		"city":      "Contagem",
		"confirmed": true,
	}, "Bruna")

	if !strings.Contains(result.Text, "atualizada") {
		t.Errorf("expected 'atualizada' in result, got: %s", result.Text)
	}

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range allBiz {
		if b.GetString("name") == "Patricia" {
			if b.GetString("city") != "Contagem" {
				t.Errorf("city: got %q, want %q", b.GetString("city"), "Contagem")
			}
			return
		}
	}
	t.Fatal("Patricia not found in DB")
}

// TestCustomerUpdate_NotFound: update non-existent customer.
func TestCustomerUpdate_NotFound(t *testing.T) {
	app := newWave4TestApp(t)
	te := newExecutor(t, app)

	result := callTool(t, te, "update_customer", map[string]any{
		"name":      "Inexistente",
		"city":      "SP",
		"confirmed": true,
	}, "Bruna")

	if !strings.Contains(result.Text, "não encontrei") {
		t.Errorf("expected 'not found' message, got: %s", result.Text)
	}
}

// TestCustomerPause_HappyPath: seed, call pause with confirmed=true, verify invite_status.
func TestCustomerPause_HappyPath(t *testing.T) {
	app := newWave4TestApp(t)
	wave4SeedBusiness(t, app, "Joana", "Loja", "RJ")
	te := newExecutor(t, app)

	result := callTool(t, te, "pause_customer", map[string]any{
		"name":      "Joana",
		"reason":    "vai viajar",
		"confirmed": true,
	}, "Bruna")

	if !strings.Contains(result.Text, "pausada") {
		t.Errorf("expected 'pausada' in result, got: %s", result.Text)
	}

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range allBiz {
		if b.GetString("name") == "Joana" {
			if b.GetString("invite_status") != domain.InviteStatusCancelled {
				t.Errorf("invite_status: got %q, want %q", b.GetString("invite_status"), domain.InviteStatusCancelled)
			}
			return
		}
	}
	t.Fatal("Joana not found in DB")
}

// TestPostGenerate_HappyPath: seed business, call generate with confirmed=true, verify post created.
func TestPostGenerate_HappyPath(t *testing.T) {
	app := newWave4TestApp(t)
	wave4SeedBusiness(t, app, "Patricia", "Salão", "BH")

	fakeGenerate := func(_ context.Context, _ content.BusinessProfile, _ []content.Role, _ []string) ([]content.Post, error) {
		return []content.Post{{
			Caption:        "Post gerado de teste",
			Hashtags:       []string{"#teste"},
			ProductionNote: "nota",
		}}, nil
	}

	te := &ToolExecutor{
		Ctx:      context.Background(),
		App:      app,
		Generate: fakeGenerate,
	}

	result := callTool(t, te, "generate_post", map[string]any{
		"customer_name": "Patricia",
		"confirmed":     true,
	}, "Elenice")

	if !strings.Contains(result.Text, "post gerado") {
		t.Errorf("expected 'post gerado' in result, got: %s", result.Text)
	}

	posts, err := app.FindAllRecords(domain.CollPosts)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range posts {
		if strings.Contains(p.GetString("caption"), "Post gerado de teste") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected post to be created after generate")
	}
}

// TestPostApprove_HappyPath: seed business + post, call approve with confirmed=true, verify reviewed=true.
func TestPostApprove_HappyPath(t *testing.T) {
	app := newWave4TestApp(t)
	biz := wave4SeedBusiness(t, app, "Patricia", "Salão", "BH")
	post := wave4SeedPost(t, app, biz.Id, "Hoje no salão foi dia de transformação...")
	te := newExecutor(t, app)

	result := callTool(t, te, "approve_post", map[string]any{
		"post_id":       post.Id,
		"customer_name": "Patricia",
		"confirmed":     true,
	}, "Bruna")

	if !strings.Contains(result.Text, "aprovado") {
		t.Errorf("expected 'aprovado' in result, got: %s", result.Text)
	}

	updated, err := app.FindRecordById(domain.CollPosts, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetBool("reviewed") {
		t.Error("post should be reviewed=true after approval")
	}
}

// TestPostReject_WithFeedback: reject with feedback, verify reviewed=true and review_note.
func TestPostReject_WithFeedback(t *testing.T) {
	app := newWave4TestApp(t)
	biz := wave4SeedBusiness(t, app, "Maria", "Confeitaria", "SP")
	post := wave4SeedPost(t, app, biz.Id, "Bolo caseiro é sempre a melhor pedida...")
	te := newExecutor(t, app)

	result := callTool(t, te, "reject_post", map[string]any{
		"post_id":       post.Id,
		"customer_name": "Maria",
		"feedback":      "muito genérico",
		"confirmed":     true,
	}, "Elenice")

	if !strings.Contains(result.Text, "rejeitado") {
		t.Errorf("expected 'rejeitado' in result, got: %s", result.Text)
	}

	updated, err := app.FindRecordById(domain.CollPosts, post.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.GetBool("reviewed") {
		t.Error("post should be reviewed=true after rejection")
	}
	if updated.GetString("review_note") != "muito genérico" {
		t.Errorf("review_note: got %q, want %q", updated.GetString("review_note"), "muito genérico")
	}
}

// TestBuildClaudeMessages_DuplicateCurrentMessage verifies that the current user message
// (already stored in DB before buildClaudeMessages runs) doesn't produce duplicate
// consecutive user messages that violate the Claude API contract.
func TestBuildClaudeMessages_DuplicateCurrentMessage(t *testing.T) {
	// Simulate the exact DB state from the prod bug:
	// ProcessMessage stores the user message first, then LoadRecentAndPrune
	// loads it back, and buildClaudeMessages appends it again.
	history := []ConversationMessage{
		{Role: "user", Structured: `{"role":"user","content":[{"text":"Quais clientes?","type":"text"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"id":"toolu_xxx","input":{},"name":"list_customers","type":"tool_use"}]}`},
		{Role: "user", Structured: `{"role":"user","content":[{"tool_use_id":"toolu_xxx","is_error":false,"content":[{"text":"Clientes: 1","type":"text"}],"type":"tool_result"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"text":"Temos 1 cliente.","type":"text"}]}`},
		{Role: "user", Structured: `{"role":"user","content":[{"text":"algum post pendente?","type":"text"}]}`},
	}

	msgs := buildClaudeMessages(history, "algum post pendente?")

	// Check alternating roles
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Role == msgs[i-1].Role {
			t.Errorf("consecutive same role at index %d and %d: both %q", i-1, i, msgs[i].Role)
		}
	}

	// Check no orphaned tool_result blocks
	for i, msg := range msgs {
		if msg.Role != "user" {
			continue
		}
		for _, block := range msg.Content {
			if block.OfToolResult == nil {
				continue
			}
			toolUseID := block.OfToolResult.ToolUseID
			if i == 0 {
				t.Errorf("tool_result at messages[0] has no preceding assistant message")
				continue
			}
			prev := msgs[i-1]
			found := false
			for _, pb := range prev.Content {
				if pb.OfToolUse != nil && pb.OfToolUse.ID == toolUseID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("messages[%d] has tool_result referencing %q but preceding assistant message has no matching tool_use", i, toolUseID)
			}
		}
	}
}

// TestBuildClaudeMessages_OrphanedToolResult verifies that tool_result blocks
// without a matching tool_use (e.g. from history pruning) are stripped.
func TestBuildClaudeMessages_OrphanedToolResult(t *testing.T) {
	history := []ConversationMessage{
		{Role: "user", Structured: `{"role":"user","content":[{"tool_use_id":"toolu_orphan","is_error":false,"content":[{"text":"result","type":"text"}],"type":"tool_result"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"text":"ok","type":"text"}]}`},
	}

	msgs := buildClaudeMessages(history, "oi")

	for i, msg := range msgs {
		for _, block := range msg.Content {
			if block.OfToolResult != nil {
				t.Errorf("messages[%d] still has orphaned tool_result for %q", i, block.OfToolResult.ToolUseID)
			}
		}
	}
}

// TestBuildClaudeMessages_OrphanedToolUse verifies that tool_use blocks
// without a matching tool_result in the next message are stripped.
func TestBuildClaudeMessages_OrphanedToolUse(t *testing.T) {
	// tool_use in assistant but the following user message is plain text (no tool_result)
	history := []ConversationMessage{
		{Role: "user", Structured: `{"role":"user","content":[{"text":"busca","type":"text"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"id":"toolu_orphan","input":{},"name":"find_customer","type":"tool_use"}]}`},
		{Role: "user", Structured: `{"role":"user","content":[{"text":"esquece","type":"text"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"text":"ok","type":"text"}]}`},
	}

	msgs := buildClaudeMessages(history, "oi")

	for i, msg := range msgs {
		for _, block := range msg.Content {
			if block.OfToolUse != nil {
				t.Errorf("messages[%d] still has orphaned tool_use %q", i, block.OfToolUse.ID)
			}
		}
	}

	// Should still have user/assistant/user messages (without the empty assistant)
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Role == msgs[i-1].Role {
			t.Errorf("consecutive same role at index %d and %d: both %q", i-1, i, msgs[i].Role)
		}
	}
}

// TestPreviewFlow_NoExtraAssistantMessage verifies that after a preview (loop stopped
// for confirmation), the conversation context lets Claude see the pending tool_use
// so it can follow up with confirmed=true.
func TestPreviewFlow_NoExtraAssistantMessage(t *testing.T) {
	// Simulate the DB state after a preview flow where sendAndLog stored
	// only the loop messages (no extra text-only assistant message).
	history := []ConversationMessage{
		{Role: "user", Structured: `{"role":"user","content":[{"text":"rejeita o post da Nika, muito genérico","type":"text"}]}`},
		{Role: "assistant", Structured: `{"role":"assistant","content":[{"text":"Vou mostrar o preview.","type":"text"},{"id":"toolu_preview","input":{"post_id":"p1","feedback":"genérico","confirmed":false},"name":"reject_post","type":"tool_use"}]}`},
		{Role: "user", Structured: `{"role":"user","content":[{"tool_use_id":"toolu_preview","is_error":false,"content":[{"text":"Preview: rejeitar post p1. Aguardando confirmação.","type":"text"}],"type":"tool_result"}]}`},
		// NO extra assistant text message (this is the fix)
		// User confirmation stored by ProcessMessage:
		{Role: "user", Structured: `{"role":"user","content":[{"text":"sim","type":"text"}]}`},
	}

	msgs := buildClaudeMessages(history, "sim")

	// The last assistant message should contain the tool_use block,
	// so Claude knows there's a pending confirmation
	var lastAssistant anthropic.MessageParam
	for _, msg := range msgs {
		if msg.Role == anthropic.MessageParamRoleAssistant {
			lastAssistant = msg
		}
	}

	hasToolUse := false
	for _, block := range lastAssistant.Content {
		if block.OfToolUse != nil && block.OfToolUse.Name == "reject_post" {
			hasToolUse = true
		}
	}
	if !hasToolUse {
		t.Error("last assistant message should contain reject_post tool_use for confirmation context")
		for i, msg := range msgs {
			out, _ := json.Marshal(msg)
			t.Logf("messages[%d]: %s", i, out)
		}
	}

	// The last user message should contain the confirmation text
	lastUser := msgs[len(msgs)-1]
	hasConfirmText := false
	for _, block := range lastUser.Content {
		if block.OfText != nil && block.OfText.Text == "sim" {
			hasConfirmText = true
		}
	}
	if !hasConfirmText {
		t.Error("last user message should contain confirmation text")
	}

	// Verify the broken case: if an extra assistant text message were stored
	// (the old behavior), the tool_use context would be lost
	historyWithExtra := append(history[:3:3],
		ConversationMessage{Role: "assistant", Structured: `{"role":"assistant","content":[{"text":"Post da Nika sobre costura...","type":"text"}]}`},
		history[3],
	)
	msgsWithExtra := buildClaudeMessages(historyWithExtra, "sim")

	var lastAssistantExtra anthropic.MessageParam
	for _, msg := range msgsWithExtra {
		if msg.Role == anthropic.MessageParamRoleAssistant {
			lastAssistantExtra = msg
		}
	}
	for _, block := range lastAssistantExtra.Content {
		if block.OfToolUse != nil {
			t.Error("with extra assistant message, tool_use should have been sanitized (proving the old behavior was broken)")
		}
	}
}

// TestDoubleConfirmation_Idempotent: call create_customer with confirmed=true twice, findDuplicate catches second.
func TestDoubleConfirmation_Idempotent(t *testing.T) {
	app := newWave4TestApp(t)
	te := newExecutor(t, app)

	callTool(t, te, "create_customer", map[string]any{
		"name":      "Ana",
		"type":      "Manicure",
		"city":      "SP",
		"phone":     "11999990000",
		"confirmed": true,
	}, "Elenice")

	// Reset cached businesses so findDuplicate picks up the new record
	te.businesses = nil

	result := callTool(t, te, "create_customer", map[string]any{
		"name":      "Ana",
		"type":      "Manicure",
		"city":      "SP",
		"phone":     "11999990000",
		"confirmed": true,
	}, "Elenice")

	if !strings.Contains(result.Text, "já existe") {
		t.Errorf("expected duplicate detection, got: %s", result.Text)
	}

	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 business 'Ana', got %d", count)
	}
}
