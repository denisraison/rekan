package service_test

import (
	"testing"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/denisraison/rekan/api/migrations"
)

func createScheduledMessage(t testing.TB, app core.App, bizID, text string) string {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(domain.CollScheduledMessages)
	if err != nil {
		t.Fatalf("find scheduled_messages collection: %v", err)
	}
	record := core.NewRecord(coll)
	record.Set("business", bizID)
	record.Set("text", text)
	record.Set("approved", false)
	record.Set("dismissed", false)
	if err := app.Save(record); err != nil {
		t.Fatalf("save scheduled message: %v", err)
	}
	return record.Id
}

func TestListScheduledMessages(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	createScheduledMessage(t, app, bizID, "Mensagem 1")
	createScheduledMessage(t, app, bizID, "Mensagem 2")

	msgs, err := service.ListScheduledMessages(app)
	if err != nil {
		t.Fatalf("ListScheduledMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestDismissScheduledMessage(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	msgID := createScheduledMessage(t, app, bizID, "Para dispensar")

	err := service.DismissScheduledMessage(app, msgID)
	if err != nil {
		t.Fatalf("DismissScheduledMessage: %v", err)
	}

	// Verify dismissed=true
	record, err := app.FindRecordById(domain.CollScheduledMessages, msgID)
	if err != nil {
		t.Fatalf("find record: %v", err)
	}
	if !record.GetBool("dismissed") {
		t.Error("dismissed should be true")
	}

	// Should no longer appear in list
	msgs, _ := service.ListScheduledMessages(app)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after dismiss, got %d", len(msgs))
	}
}
