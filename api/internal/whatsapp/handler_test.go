package whatsapp

import (
	"testing"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func newHandlerTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	businesses := core.NewBaseCollection("businesses")
	businesses.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "client_name"},
		&core.TextField{Name: "type"},
		&core.TextField{Name: "city"},
		&core.TextField{Name: "state"},
		&core.JSONField{Name: "services"},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "phone"},
	)
	if err := app.Save(businesses); err != nil {
		t.Fatalf("save businesses collection: %v", err)
	}

	messages := core.NewBaseCollection("messages")
	messages.Fields.Add(
		&core.TextField{Name: "business"},
		&core.TextField{Name: "phone", Required: true},
		&core.SelectField{Name: "type", Values: []string{"text", "audio", "image"}, Required: true, MaxSelect: 1},
		&core.TextField{Name: "content"},
		&core.SelectField{Name: "direction", Values: []string{"incoming", "outgoing"}, Required: true, MaxSelect: 1},
		&core.DateField{Name: "wa_timestamp"},
		&core.TextField{Name: "wa_message_id"},
	)
	if err := app.Save(messages); err != nil {
		t.Fatalf("save messages collection: %v", err)
	}

	return app
}

func makeDeps(t *testing.T, app *tests.TestApp) HandlerDeps {
	t.Helper()
	return HandlerDeps{App: app, Client: nil, Transcribe: nil}
}

func incomingTextEvt(msgID, senderPhone string) *events.Message {
	return incomingTextEvtWithName(msgID, senderPhone, "")
}

func incomingTextEvtWithName(msgID, senderPhone, pushName string) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				IsFromMe: false,
				IsGroup:  false,
				Sender:   types.JID{User: senderPhone, Server: "s.whatsapp.net"},
				Chat:     types.JID{User: senderPhone, Server: "s.whatsapp.net"},
			},
			ID:        types.MessageID(msgID),
			Timestamp: time.Now(),
			PushName:  pushName,
		},
		Message: &waE2E.Message{Conversation: proto.String("Olá")},
	}
}

func lidEvt(msgID string) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				IsFromMe: true,
				IsGroup:  false,
				Sender:   types.JID{User: "me", Server: "s.whatsapp.net"},
				Chat:     types.JID{User: "10235024560177", Server: "lid"},
			},
			ID:        types.MessageID(msgID),
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{Conversation: proto.String("Ola")},
	}
}

func outgoingTextEvt(msgID, recipientPhone string) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				IsFromMe: true,
				IsGroup:  false,
				Sender:   types.JID{User: "me", Server: "s.whatsapp.net"},
				Chat:     types.JID{User: recipientPhone, Server: "s.whatsapp.net"},
			},
			ID:        types.MessageID(msgID),
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{Conversation: proto.String("Resposta")},
	}
}

func countMessages(t *testing.T, app *tests.TestApp) int {
	t.Helper()
	records, err := app.FindAllRecords(domain.CollMessages)
	if err != nil {
		t.Fatalf("find messages: %v", err)
	}
	return len(records)
}

func countBusinesses(t *testing.T, app *tests.TestApp) int {
	t.Helper()
	records, err := app.FindAllRecords(domain.CollBusinesses)
	if err != nil {
		t.Fatalf("find businesses: %v", err)
	}
	return len(records)
}

// TestHandleMessageUnknownSenderCreatesPlaceholder verifies that a message from
// a phone number not in the businesses table creates a placeholder business and
// saves the message linked to it.
func TestHandleMessageUnknownSenderCreatesPlaceholder(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, incomingTextEvt("msg1", "5511999990001"))

	if got := countBusinesses(t, app); got != 1 {
		t.Fatalf("want 1 placeholder business, got %d", got)
	}
	if got := countMessages(t, app); got != 1 {
		t.Fatalf("want 1 message, got %d", got)
	}

	biz, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "phone = '5511999990001'")
	if err != nil {
		t.Fatalf("placeholder business not found: %v", err)
	}
	if biz.GetString("name") != "+5511999990001" {
		t.Errorf("placeholder name = %q, want %q", biz.GetString("name"), "+5511999990001")
	}

	msg, err := app.FindFirstRecordByFilter(domain.CollMessages, "wa_message_id = 'msg1'")
	if err != nil {
		t.Fatalf("message not found: %v", err)
	}
	if msg.GetString("business") != biz.Id {
		t.Errorf("message.business = %q, want %q", msg.GetString("business"), biz.Id)
	}
	if msg.GetString("direction") != domain.DirectionIncoming {
		t.Errorf("direction = %q, want incoming", msg.GetString("direction"))
	}
}

// TestHandleMessageKnownSenderLinksExistingBusiness verifies that a message from
// a known phone number links to the existing business without creating a new one.
func TestHandleMessageKnownSenderLinksExistingBusiness(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	businesses, _ := app.FindCollectionByNameOrId(domain.CollBusinesses)
	biz := core.NewRecord(businesses)
	biz.Set("name", "Padaria da Ana")
	biz.Set("type", "padaria")
	biz.Set("phone", "5511888880001")
	if err := app.Save(biz); err != nil {
		t.Fatalf("save business: %v", err)
	}

	handleMessage(deps, incomingTextEvt("msg2", "5511888880001"))

	if got := countBusinesses(t, app); got != 1 {
		t.Errorf("want 1 business, got %d (should not create placeholder)", got)
	}

	msg, _ := app.FindFirstRecordByFilter(domain.CollMessages, "wa_message_id = 'msg2'")
	if msg.GetString("business") != biz.Id {
		t.Errorf("message.business = %q, want %q", msg.GetString("business"), biz.Id)
	}
}

// TestHandleMessageFromMeIsOutgoing verifies that messages sent from the connected
// device are stored with direction=outgoing and the recipient's phone number.
func TestHandleMessageFromMeIsOutgoing(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, outgoingTextEvt("msg3", "5511777770001"))

	msg, err := app.FindFirstRecordByFilter(domain.CollMessages, "wa_message_id = 'msg3'")
	if err != nil {
		t.Fatalf("message not found: %v", err)
	}
	if msg.GetString("direction") != domain.DirectionOutgoing {
		t.Errorf("direction = %q, want outgoing", msg.GetString("direction"))
	}
	if msg.GetString("phone") != "5511777770001" {
		t.Errorf("phone = %q, want recipient phone", msg.GetString("phone"))
	}
}

// TestHandleMessageGroupIsIgnored verifies that group messages are silently dropped.
func TestHandleMessageGroupIsIgnored(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	evt := incomingTextEvt("msg4", "5511666660001")
	evt.Info.IsGroup = true

	handleMessage(deps, evt)

	if got := countMessages(t, app); got != 0 {
		t.Errorf("want 0 messages for group, got %d", got)
	}
}

// TestHandleMessageDeduplication verifies that a second message with the same
// wa_message_id is silently dropped.
func TestHandleMessageDeduplication(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, incomingTextEvt("dup1", "5511555550001"))
	handleMessage(deps, incomingTextEvt("dup1", "5511555550001"))

	if got := countMessages(t, app); got != 1 {
		t.Errorf("want 1 message after dedup, got %d", got)
	}
}

// TestHandleMessageSecondMessageReusesPlaceholder verifies that two messages from
// the same unknown number share one placeholder business.
func TestHandleMessageSecondMessageReusesPlaceholder(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, incomingTextEvt("m1", "5511444440001"))
	handleMessage(deps, incomingTextEvt("m2", "5511444440001"))

	if got := countBusinesses(t, app); got != 1 {
		t.Errorf("want 1 placeholder business, got %d", got)
	}
	if got := countMessages(t, app); got != 2 {
		t.Errorf("want 2 messages, got %d", got)
	}
}

// TestHandleMessagePushNameUsedAsPlaceholderName verifies that an incoming message
// with a push name creates a placeholder business using that name instead of the phone.
func TestHandleMessagePushNameUsedAsPlaceholderName(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, incomingTextEvtWithName("msg-name1", "5511999990002", "Maria Silva"))

	biz, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "phone = '5511999990002'")
	if err != nil {
		t.Fatalf("placeholder business not found: %v", err)
	}
	if biz.GetString("name") != "Maria Silva" {
		t.Errorf("name = %q, want %q", biz.GetString("name"), "Maria Silva")
	}
	if biz.GetString("client_name") != "Maria Silva" {
		t.Errorf("client_name = %q, want %q", biz.GetString("client_name"), "Maria Silva")
	}
}

// TestHandleMessagePushNameUpdatesExistingPlaceholder verifies that a second message
// with a push name updates a placeholder whose name was still the raw phone number.
func TestHandleMessagePushNameUpdatesExistingPlaceholder(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, incomingTextEvt("msg-name2", "5511999990003"))
	handleMessage(deps, incomingTextEvtWithName("msg-name3", "5511999990003", "João Costa"))

	biz, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "phone = '5511999990003'")
	if err != nil {
		t.Fatalf("placeholder business not found: %v", err)
	}
	if biz.GetString("name") != "João Costa" {
		t.Errorf("name = %q, want %q", biz.GetString("name"), "João Costa")
	}
	if got := countBusinesses(t, app); got != 1 {
		t.Errorf("want 1 business, got %d", got)
	}
}

// TestHandleMessageLIDJIDIsIgnored verifies that messages with LID JIDs (WhatsApp
// internal IDs) are silently dropped rather than creating a garbage placeholder.
func TestHandleMessageLIDJIDIsIgnored(t *testing.T) {
	app := newHandlerTestApp(t)
	deps := makeDeps(t, app)

	handleMessage(deps, lidEvt("msg-lid1"))

	if got := countMessages(t, app); got != 0 {
		t.Errorf("want 0 messages for LID JID, got %d", got)
	}
	if got := countBusinesses(t, app); got != 0 {
		t.Errorf("want 0 businesses for LID JID, got %d", got)
	}
}
