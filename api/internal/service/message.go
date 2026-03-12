package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/postingtime"
	wa "github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

var (
	ErrNoPhone  = errors.New("cliente sem telefone cadastrado")
	ErrNotFound = errors.New("não encontrado")
	ErrConflict = errors.New("conflito")
)

// SimulateTyping shows a typing indicator, waits 1-3s, runs fn, then clears
// the indicator. The typing indicator is best-effort (errors are ignored).
func SimulateTyping(ctx context.Context, waClient *wa.Client, jid types.JID, fn func() error) error {
	waClient.SendChatPresence(ctx, jid, types.ChatPresenceComposing, "")
	delay := time.Duration(1000+rand.IntN(2000)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		waClient.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")
		return ctx.Err()
	}
	err := fn()
	waClient.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")
	return err
}

// StoreOutgoingMessage saves an outgoing message record. Errors are logged but
// not returned since message storage is best-effort.
func StoreOutgoingMessage(app core.App, businessID, phone, msgType, content string, media *filesystem.File) {
	collection, _ := app.FindCollectionByNameOrId(domain.CollMessages)
	if collection == nil {
		return
	}
	record := core.NewRecord(collection)
	record.Set("business", businessID)
	record.Set("phone", phone)
	record.Set("type", msgType)
	record.Set("content", content)
	record.Set("direction", domain.DirectionOutgoing)
	record.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))
	if media != nil {
		record.Set("media", media)
	}
	if err := app.Save(record); err != nil {
		app.Logger().Error("storeOutgoingMessage: failed", "type", msgType, "error", err)
	}
}

type SendTextParams struct {
	BusinessID     string
	Caption        string
	Hashtags       string
	ProductionNote string
}

func SendTextMessage(ctx context.Context, app core.App, waClient *wa.Client, params SendTextParams) error {
	business, err := app.FindRecordById(domain.CollBusinesses, params.BusinessID)
	if err != nil {
		return err
	}

	phone := business.GetString("phone")
	if phone == "" {
		return ErrNoPhone
	}

	jid := types.NewJID(phone, types.DefaultUserServer)

	text := params.Caption
	if strings.TrimSpace(params.Hashtags) != "" {
		text += "\n\n" + params.Hashtags
	}

	err = SimulateTyping(ctx, waClient, jid, func() error {
		_, err := waClient.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &text,
		})
		return err
	})
	if err != nil {
		return err
	}

	StoreOutgoingMessage(app, params.BusinessID, phone, domain.MsgTypeText, text, nil)

	if strings.TrimSpace(params.ProductionNote) != "" {
		time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)

		noteText := "*Dica de foto:* " + params.ProductionNote
		waClient.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &noteText,
		})
		StoreOutgoingMessage(app, params.BusinessID, phone, domain.MsgTypeText, noteText, nil)

		// Posting time tip
		time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)
		tipText := postingtime.Tip(business.GetString("type"))
		waClient.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &tipText,
		})
		StoreOutgoingMessage(app, params.BusinessID, phone, domain.MsgTypeText, tipText, nil)
	}

	return nil
}

type SendMediaParams struct {
	BusinessID  string
	Caption     string
	Data        []byte
	ContentType string
	Filename    string
}

func SendMediaMessage(ctx context.Context, app core.App, waClient *wa.Client, params SendMediaParams) error {
	business, err := app.FindRecordById(domain.CollBusinesses, params.BusinessID)
	if err != nil {
		return err
	}

	phone := business.GetString("phone")
	if phone == "" {
		return ErrNoPhone
	}

	jid := types.NewJID(phone, types.DefaultUserServer)
	isVideo := strings.HasPrefix(params.ContentType, "video/")

	var waMediaType whatsmeow.MediaType
	if isVideo {
		waMediaType = whatsmeow.MediaVideo
	} else {
		waMediaType = whatsmeow.MediaImage
	}

	resp, err := waClient.Upload(ctx, params.Data, waMediaType)
	if err != nil {
		return err
	}

	var msg *waE2E.Message
	if isVideo {
		msg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(params.Caption),
				Mimetype:      proto.String(params.ContentType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			},
		}
	} else {
		msg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(params.Caption),
				Mimetype:      proto.String(params.ContentType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			},
		}
	}

	err = SimulateTyping(ctx, waClient, jid, func() error {
		_, err := waClient.SendMessage(ctx, jid, msg)
		return err
	})
	if err != nil {
		return err
	}

	msgType := domain.MsgTypeImage
	if isVideo {
		msgType = domain.MsgTypeVideo
	}
	mediaFile, _ := filesystem.NewFileFromBytes(params.Data, params.Filename)
	StoreOutgoingMessage(app, params.BusinessID, phone, msgType, params.Caption, mediaFile)

	return nil
}
