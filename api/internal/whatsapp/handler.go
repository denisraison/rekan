package whatsapp

import (
	"context"
	"log/slog"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/transcribe"
	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

// HandlerDeps holds dependencies for the message event handler.
type HandlerDeps struct {
	Client        *Client
	App           core.App
	Logger        *slog.Logger
	Transcribe    *transcribe.Client     // nil if GEMINI_API_KEY not set
	ExtractSignal eval.ExtractSignalFunc // nil if GEMINI_API_KEY not set
}

// RegisterMessageHandler wires incoming WhatsApp messages to PocketBase storage.
func RegisterMessageHandler(deps HandlerDeps) {
	deps.Client.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.Message:
			if v.Info.IsGroup {
				handleGroupMessage(deps, v)
			} else {
				handleDirectMessage(deps, v)
			}
		}
	})
}

func handleDirectMessage(deps HandlerDeps, evt *events.Message) {
	if evt.Message == nil {
		return
	}

	waMessageID := string(evt.Info.ID)
	if isDuplicate(deps, waMessageID) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resolved, ok := resolveDirectSender(ctx, deps, evt)
	if !ok {
		return
	}

	parsed, ok := extractContent(ctx, deps, evt)
	if !ok {
		return
	}

	businessID, inviteStatus, businessType := findOrCreateBusiness(deps, resolved.phone, resolved.pushName)

	if resolved.direction == domain.DirectionIncoming && deps.Client != nil {
		senderJID := types.JID{User: resolved.phone, Server: "s.whatsapp.net"}
		go refreshProfilePicture(deps, businessID, senderJID)
	}

	saveMessageRecord(deps, businessID, resolved.phone, resolved.direction, parsed.msgType, parsed.content, evt.Info.Timestamp, waMessageID, parsed.mediaFile)

	if resolved.direction == domain.DirectionIncoming && len(parsed.content) >= 20 && businessID != "" && deps.ExtractSignal != nil && inviteStatus == domain.InviteStatusActive {
		go extractAndSaveSignal(deps, businessID, businessType, parsed.content)
	}

	if parsed.extraCaption != "" {
		saveMessageRecord(deps, businessID, resolved.phone, resolved.direction, domain.MsgTypeText, parsed.extraCaption, evt.Info.Timestamp, waMessageID+"_caption", nil)
	}
}
