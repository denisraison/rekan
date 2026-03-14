package service

import (
	"context"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/pocketbase/pocketbase/core"
)

type ScheduledMessage struct {
	ID       string
	Business string
	Text     string
}

func ListScheduledMessages(app core.App) ([]ScheduledMessage, error) {
	records, err := app.FindRecordsByFilter(
		domain.CollScheduledMessages,
		"approved = false && dismissed = false",
		"-created",
		0,
		0,
	)
	if err != nil {
		return nil, nil //nolint:nilerr // no records is not an error
	}

	result := make([]ScheduledMessage, 0, len(records))
	for _, r := range records {
		result = append(result, ScheduledMessage{
			ID:       r.Id,
			Business: r.GetString("business"),
			Text:     r.GetString("text"),
		})
	}
	return result, nil
}

func ApproveScheduledMessage(ctx context.Context, app core.App, wa *whatsapp.Client, id string) error {
	record, err := app.FindRecordById(domain.CollScheduledMessages, id)
	if err != nil {
		return err
	}

	businessID := record.GetString("business")
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return err
	}

	phone := business.GetString("phone")
	if phone == "" {
		return ErrNoPhone
	}

	text := record.GetString("text")
	jid := types.NewJID(phone, types.DefaultUserServer)

	stop := whatsapp.Typing(ctx, wa, jid)
	defer stop()

	if _, err = wa.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &text,
	}); err != nil {
		return err
	}

	StoreOutgoingMessage(app, businessID, phone, domain.MsgTypeText, text, nil)

	record.Set("approved", true)
	return app.Save(record)
}

func DismissScheduledMessage(app core.App, id string) error {
	record, err := app.FindRecordById(domain.CollScheduledMessages, id)
	if err != nil {
		return err
	}

	record.Set("dismissed", true)
	return app.Save(record)
}
