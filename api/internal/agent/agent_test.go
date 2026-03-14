package agent_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/agent"
	"github.com/denisraison/rekan/api/internal/domain"
	_ "github.com/denisraison/rekan/api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

var testGroupJID = types.JID{User: "120363000000000000", Server: "g.us"}

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

func (f *fakeWAClient) SendChatPresence(_ context.Context, _ types.JID, _ types.ChatPresence, _ types.ChatPresenceMedia) error {
	return nil
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

func newAgent(t *testing.T, app core.App, wa *fakeWAClient) *agent.Agent {
	t.Helper()
	return agent.New(app, wa, slog.Default(), nil, nil)
}

// send is a test helper that calls ProcessMessage with test defaults.
func send(a *agent.Agent, message, operatorName, operatorJID string) {
	senderJID := types.JID{User: operatorJID, Server: "s.whatsapp.net"}
	a.ProcessMessage(testGroupJID, "test-msg-id", senderJID, message, operatorName, operatorJID)
}

// setConfirmingState puts an operator into the confirming state with given params.
func setConfirmingState(t *testing.T, app core.App, operatorJID, actionType string, params any) {
	t.Helper()
	state, err := agent.LoadState(app, operatorJID)
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.SetConfirming(app, state, operatorJID, actionType, params); err != nil {
		t.Fatal(err)
	}
}

// TestCancellationFlow tests that "não" cancels a pending action cleanly.
func TestCancellationFlow(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, &agent.CustomerCreateParams{
		Name: "Ana",
		Type: "Manicure",
		City: "Goiania",
	})

	a := newAgent(t, app, wa)

	send(a, "não", "Elenice", "5511999990000")

	// Verify state cleared
	state, err := agent.LoadState(app, "5511999990000")
	if err != nil {
		t.Fatal(err)
	}
	if state.State != agent.StateIdle {
		t.Errorf("expected idle after cancellation, got %s", state.State)
	}

	// Verify no business created
	allBiz, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatal(err)
	}
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

	// Set up a confirmation that will produce a reply when confirmed
	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, &agent.CustomerCreateParams{
		Name: "Ana",
		Type: "Manicure",
		City: "Goiania",
	})

	a := newAgent(t, app, wa)

	send(a, "sim", "Elenice", "5511999990000")

	msgs, err := agent.LoadRecent(app, 15)
	if err != nil {
		t.Fatal(err)
	}

	foundAssistant := false
	for _, m := range msgs {
		if m.Role == "assistant" && strings.Contains(m.Content, "cadastrada") {
			foundAssistant = true
		}
	}

	if !foundAssistant {
		t.Error("assistant reply not found in conversation buffer")
	}
}

// TestActionLog_Recorded verifies that actions are logged to agent_action_log.
func TestActionLog_Recorded(t *testing.T) {
	app := newTestApp(t)
	wa := &fakeWAClient{}

	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, &agent.CustomerCreateParams{
		Name: "Ana",
		Type: "Manicure",
		City: "Goiania",
	})

	a := newAgent(t, app, wa)

	send(a, "sim", "Elenice", "5511999990000")

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

// TestCollectedFieldsRoundTrip verifies params survive JSON roundtrip in state.
func TestCollectedFieldsRoundTrip(t *testing.T) {
	app := newTestApp(t)

	audience := "mulheres do bairro"
	params := &agent.CustomerCreateParams{
		Name:           "Ana",
		Type:           "Manicure",
		City:           "Goiania",
		TargetAudience: &audience,
	}

	setConfirmingState(t, app, "5511999990000", agent.ActionCustomerCreate, params)

	state, err := agent.LoadState(app, "5511999990000")
	if err != nil {
		t.Fatal(err)
	}

	var recovered agent.CustomerCreateParams
	if err := json.Unmarshal(state.CollectedFields, &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered.Name != "Ana" {
		t.Errorf("name: got %q, want %q", recovered.Name, "Ana")
	}
	if recovered.TargetAudience == nil || *recovered.TargetAudience != "mulheres do bairro" {
		t.Errorf("target_audience: got %v, want %q", recovered.TargetAudience, "mulheres do bairro")
	}
}
