package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// resolvedMessage holds the resolved sender info for a direct message.
type resolvedMessage struct {
	direction string
	phone     string
	pushName  string
}

// resolveDirectSender resolves the phone number and direction from a direct message event.
// Returns false if the LID cannot be resolved and the message should be skipped.
func resolveDirectSender(ctx context.Context, deps HandlerDeps, evt *events.Message) (resolvedMessage, bool) {
	if evt.Info.IsFromMe {
		jid := deps.Client.ResolveLID(ctx, evt.Info.Chat)
		if jid.IsEmpty() {
			deps.Logger.Warn("whatsapp: skipping unresolvable LID outgoing event", "chat", evt.Info.Chat)
			return resolvedMessage{}, false
		}
		return resolvedMessage{direction: domain.DirectionOutgoing, phone: jid.User}, true
	}

	jid := deps.Client.ResolveLID(ctx, evt.Info.Sender)
	if jid.IsEmpty() {
		deps.Logger.Warn("whatsapp: skipping unresolvable LID incoming event", "sender", evt.Info.Sender)
		return resolvedMessage{}, false
	}
	return resolvedMessage{direction: domain.DirectionIncoming, phone: jid.User, pushName: evt.Info.PushName}, true
}

// parsedContent holds the extracted content from a WhatsApp message.
type parsedContent struct {
	msgType      string
	content      string
	extraCaption string
	mediaFile    *filesystem.File
}

// extractContent extracts the message type, text content, and optional media from an event.
// Returns false if the message type is unsupported.
func extractContent(ctx context.Context, deps HandlerDeps, evt *events.Message) (parsedContent, bool) {
	msg := evt.Message
	var p parsedContent

	switch {
	case msg.GetConversation() != "":
		p.msgType = domain.MsgTypeText
		p.content = msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		p.msgType = domain.MsgTypeText
		p.content = msg.GetExtendedTextMessage().GetText()
	case msg.GetAudioMessage() != nil:
		p.msgType = domain.MsgTypeAudio
		p.content = transcribeAudio(ctx, deps, evt)
	case msg.GetImageMessage() != nil:
		p.msgType = domain.MsgTypeImage
		p.content, p.extraCaption, p.mediaFile = processImage(ctx, deps, evt)
	case msg.GetVideoMessage() != nil:
		p.msgType = domain.MsgTypeVideo
		p.content, p.extraCaption, p.mediaFile = processVideo(ctx, deps, evt)
	default:
		return parsedContent{}, false
	}

	return p, true
}

// isDuplicate returns true if a message with the given wa_message_id already exists.
func isDuplicate(deps HandlerDeps, waMessageID string) bool {
	existing, _ := deps.App.FindFirstRecordByFilter(domain.CollMessages, "wa_message_id = {:id}", map[string]any{"id": waMessageID})
	return existing != nil
}

// saveMessageRecord creates and saves a message record. Returns the saved record or nil on error.
func saveMessageRecord(deps HandlerDeps, businessID, phone, direction, msgType, content string, waTimestamp time.Time, waMessageID string, mediaFile *filesystem.File) *core.Record {
	collection, err := deps.App.FindCachedCollectionByNameOrId(domain.CollMessages)
	if err != nil {
		deps.Logger.Error("whatsapp: messages collection not found", "error", err)
		return nil
	}

	record := core.NewRecord(collection)
	if businessID != "" {
		record.Set("business", businessID)
	}
	record.Set("phone", phone)
	record.Set("type", msgType)
	record.Set("content", content)
	record.Set("direction", direction)
	record.Set("wa_timestamp", waTimestamp.UTC().Format(time.RFC3339))
	record.Set("wa_message_id", waMessageID)

	if mediaFile != nil {
		record.Set("media", mediaFile)
	}

	if err := deps.App.Save(record); err != nil {
		deps.Logger.Error("whatsapp: failed to save message", "error", err)
		return nil
	}

	return record
}
