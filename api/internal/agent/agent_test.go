package agent_test

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	bamltypes "github.com/denisraison/rekan/api/internal/baml/baml_client/types"
	"github.com/denisraison/rekan/api/internal/domain"

	"github.com/denisraison/rekan/api/internal/agent"
	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

var testGroupJID = types.JID{User: "120363000000000000", Server: "g.us"}

// bamlCall records the arguments passed to a fake BAML function.
type bamlCall struct {
	OperatorName        string
	Message             string
	SystemContext       string
	ConversationHistory string
}

// fakeWAClient implements agent.WAClient for testing.
type fakeWAClient struct {
	mu      sync.Mutex
	sent    []string
	reacted bool
}

func (f *fakeWAClient) SendMessage(_ context.Context, _ types.JID, msg *waE2E.Message) (whatsmeow.SendResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if msg.Conversation != nil {
		f.sent = append(f.sent, *msg.Conversation)
	}
	if msg.ReactionMessage != nil {
		f.reacted = true
	}
	return whatsmeow.SendResponse{}, nil
}

func (f *fakeWAClient) ResolveLID(_ context.Context, jid types.JID) types.JID {
	return jid
}

func (f *fakeWAClient) Download(_ context.Context, _ whatsmeow.DownloadableMessage) ([]byte, error) {
	return nil, nil
}

func (f *fakeWAClient) Upload(_ context.Context, _ []byte, _ whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	return whatsmeow.UploadResponse{}, nil
}

func (f *fakeWAClient) sentMessages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]string, len(f.sent))
	copy(cp, f.sent)
	return cp
}

func newTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Cleanup() })
	return app
}

func seedBusiness(t *testing.T, app core.App, name, bizType, city string) *core.Record {
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

func newAgent(t *testing.T, app core.App, wa *fakeWAClient, bamlFn agent.BAMLFunc) *agent.Agent {
	t.Helper()
	a := agent.New(app, wa, slog.Default(), nil, nil)
	a.BAML = bamlFn
	return a
}

// send is a test helper that calls ProcessMessage with test defaults.
func send(a *agent.Agent, message, operatorName, operatorJID string) {
	senderJID := types.JID{User: operatorJID, Server: "s.whatsapp.net"}
	a.ProcessMessage(testGroupJID, "test-msg-id", senderJID, message, operatorName, operatorJID)
}

// TestConversationHistory_PassedToBAML verifies that previous messages
// stored in agent_conversations are loaded and passed to the BAML function.
func TestConversationHistory_PassedToBAML(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Patricia", "Salão de Beleza", "Belo Horizonte")

	wa := &fakeWAClient{}

	var calls []bamlCall
	var mu sync.Mutex

	fakeBAML := func(_ context.Context, operatorName, message, systemContext, conversationHistory string) (bamltypes.AgentResponse, error) {
		mu.Lock()
		calls = append(calls, bamlCall{
			OperatorName:        operatorName,
			Message:             message,
			SystemContext:       systemContext,
			ConversationHistory: conversationHistory,
		})
		mu.Unlock()

		reply := operatorName + ", tudo certo!"
		return bamltypes.AgentResponse{Reply: &reply}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	// Pre-seed conversation history
	_ = agent.StoreMessage(app, "Elenice", "5511999990000", "user", "oi, como tá tudo?", "")
	_ = agent.StoreMessage(app, "Rekan", "", "assistant", "Elenice, temos 1 cliente ativa.", "")
	_ = agent.StoreMessage(app, "Elenice", "5511999990000", "user", "quais são as clientes?", "")

	// Process a new message through the full pipeline
	send(a, "e a Patricia, como tá?", "Elenice", "5511999990000")

	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 1 {
		t.Fatalf("expected 1 BAML call, got %d", len(calls))
	}

	call := calls[0]

	if call.OperatorName != "Elenice" {
		t.Errorf("operator name: got %q, want %q", call.OperatorName, "Elenice")
	}
	if call.Message != "e a Patricia, como tá?" {
		t.Errorf("message: got %q, want %q", call.Message, "e a Patricia, como tá?")
	}

	// Conversation history must contain all previous messages
	if call.ConversationHistory == "(sem histórico)" {
		t.Fatal("conversation history is empty, expected previous messages")
	}
	if !strings.Contains(call.ConversationHistory, "oi, como tá tudo?") {
		t.Errorf("history missing first user message, got:\n%s", call.ConversationHistory)
	}
	if !strings.Contains(call.ConversationHistory, "Elenice, temos 1 cliente ativa.") {
		t.Errorf("history missing assistant reply, got:\n%s", call.ConversationHistory)
	}
	if !strings.Contains(call.ConversationHistory, "quais são as clientes?") {
		t.Errorf("history missing second user message, got:\n%s", call.ConversationHistory)
	}

	// System context must contain the business
	if !strings.Contains(call.SystemContext, "Patricia") {
		t.Errorf("system context missing business, got:\n%s", call.SystemContext)
	}
}

// TestConfirmationFlow_EndToEnd tests create -> confirm -> DB record.
func TestConfirmationFlow_EndToEnd(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, operatorName, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := operatorName + ", cadastrar Ana, Manicure em Goiânia. Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{
					Name: "Ana",
					Type: "Manicure",
					City: "Goiânia",
				},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	// Step 1: creation request
	send(a, "cria a Ana, manicure em Goiânia", "Elenice", "5511999990000")

	sent := wa.sentMessages()
	if len(sent) == 0 {
		t.Fatal("expected a reply, got none")
	}
	if !strings.Contains(sent[len(sent)-1], "Confirma?") {
		t.Errorf("expected confirmation prompt, got: %s", sent[len(sent)-1])
	}

	// Verify confirming state
	state, err := agent.LoadState(app, "5511999990000")
	if err != nil {
		t.Fatal(err)
	}
	if state.State != agent.StateConfirming {
		t.Fatalf("expected confirming state, got %s", state.State)
	}
	if state.ActionType != "CUSTOMER_CREATE" {
		t.Fatalf("expected CUSTOMER_CREATE, got %s", state.ActionType)
	}

	// Step 2: confirm
	send(a, "sim", "Elenice", "5511999990000")

	// Verify business in DB
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
		t.Fatal("business 'Ana' not found in DB after confirmation")
	}
	if biz.GetString("type") != "Manicure" {
		t.Errorf("business type: got %q, want %q", biz.GetString("type"), "Manicure")
	}
	if biz.GetString("city") != "Goiânia" {
		t.Errorf("business city: got %q, want %q", biz.GetString("city"), "Goiânia")
	}

	// Verify state cleared
	state, _ = agent.LoadState(app, "5511999990000")
	if state.State != agent.StateIdle {
		t.Errorf("expected idle state after confirmation, got %s", state.State)
	}
}

// TestCancellationFlow tests that "não" cancels a pending action cleanly.
func TestCancellationFlow(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, operatorName, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := operatorName + ", cadastrar Ana? Confirma?"
		return bamltypes.AgentResponse{
			Reply: &reply,
			Action: &bamltypes.AgentAction{
				ActionType:   bamltypes.AgentActionTypeCUSTOMER_CREATE,
				ActionStatus: bamltypes.AgentActionStatusNEEDS_CONFIRMATION,
				CustomerCreate: &bamltypes.CustomerCreateParams{Name: "Ana", Type: "Manicure", City: "Goiânia"},
			},
		}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	send(a, "cria a Ana", "Elenice", "5511999990000")
	send(a, "não", "Elenice", "5511999990000")

	// Verify state cleared
	state, _ := agent.LoadState(app, "5511999990000")
	if state.State != agent.StateIdle {
		t.Errorf("expected idle after cancellation, got %s", state.State)
	}

	// Verify no business created
	allBiz, _ := app.FindAllRecords(domain.CollBusinesses)
	for _, b := range allBiz {
		if b.GetString("name") == "Ana" {
			t.Error("business 'Ana' should not exist after cancellation")
		}
	}

	// Verify cancellation reply sent
	sent := wa.sentMessages()
	found := false
	for _, msg := range sent {
		if strings.Contains(msg, "cancelado") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected cancellation reply, got: %v", sent)
	}
}

// TestReplyStoredInConversation verifies that agent replies are stored in the buffer.
func TestReplyStoredInConversation(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, operatorName, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := operatorName + ", tudo certo!"
		return bamltypes.AgentResponse{Reply: &reply}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	send(a, "oi", "Elenice", "5511999990000")

	msgs, err := agent.LoadRecent(app, 15)
	if err != nil {
		t.Fatal(err)
	}

	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages in conversation buffer, got %d", len(msgs))
	}

	foundUser := false
	foundAssistant := false
	for _, m := range msgs {
		if m.Role == "user" && m.Content == "oi" {
			foundUser = true
		}
		if m.Role == "assistant" && strings.Contains(m.Content, "tudo certo!") {
			foundAssistant = true
		}
	}

	if !foundUser {
		t.Error("user message not found in conversation buffer")
	}
	if !foundAssistant {
		t.Error("assistant reply not found in conversation buffer")
	}
}

// TestActionLog_Recorded verifies that actions are logged to agent_action_log.
func TestActionLog_Recorded(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	fakeBAML := func(_ context.Context, operatorName, _, _, _ string) (bamltypes.AgentResponse, error) {
		reply := operatorName + ", resposta."
		return bamltypes.AgentResponse{Reply: &reply}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	send(a, "como tá tudo?", "Elenice", "5511999990000")

	logs, err := app.FindAllRecords(domain.CollAgentActionLog)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least 1 action log entry")
	}

	entry := logs[len(logs)-1]
	if entry.GetString("operator_name") != "Elenice" {
		t.Errorf("log operator_name: got %q, want %q", entry.GetString("operator_name"), "Elenice")
	}
	if !entry.GetBool("success") {
		t.Error("expected success=true in action log")
	}
}

// TestMultiTurnConversation verifies that conversation accumulates across turns
// and each BAML call sees the growing history.
func TestMultiTurnConversation(t *testing.T) {
	app := newTestApp(t)
	seedBusiness(t, app, "Patricia", "Salão de Beleza", "Belo Horizonte")

	wa := &fakeWAClient{}

	var calls []bamlCall
	var mu sync.Mutex

	fakeBAML := func(_ context.Context, operatorName, message, systemContext, conversationHistory string) (bamltypes.AgentResponse, error) {
		mu.Lock()
		calls = append(calls, bamlCall{
			OperatorName:        operatorName,
			Message:             message,
			SystemContext:       systemContext,
			ConversationHistory: conversationHistory,
		})
		mu.Unlock()

		reply := operatorName + ", resposta " + message
		return bamltypes.AgentResponse{Reply: &reply}, nil
	}

	a := newAgent(t, app, wa, fakeBAML)

	send(a, "como tá tudo?", "Elenice", "5511999990000")
	send(a, "e a Patricia?", "Elenice", "5511999990000")
	send(a, "gera post pra ela", "Elenice", "5511999990000")

	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 3 {
		t.Fatalf("expected 3 BAML calls, got %d", len(calls))
	}

	// Turn 2: should see turn 1's user message + assistant reply
	hist2 := calls[1].ConversationHistory
	if !strings.Contains(hist2, "como tá tudo?") {
		t.Errorf("turn 2 missing turn 1 user message, got:\n%s", hist2)
	}
	if !strings.Contains(hist2, "resposta como tá tudo?") {
		t.Errorf("turn 2 missing turn 1 assistant reply, got:\n%s", hist2)
	}

	// Turn 3: should see turns 1+2
	hist3 := calls[2].ConversationHistory
	if !strings.Contains(hist3, "e a Patricia?") {
		t.Errorf("turn 3 missing turn 2 user message, got:\n%s", hist3)
	}
	if !strings.Contains(hist3, "resposta e a Patricia?") {
		t.Errorf("turn 3 missing turn 2 assistant reply, got:\n%s", hist3)
	}
}
