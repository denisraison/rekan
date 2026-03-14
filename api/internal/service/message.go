package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
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

// WAClient is the subset of whatsmeow used for sending messages.
type WAClient interface {
	SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message) (whatsmeow.SendResponse, error)
	SendChatPresence(ctx context.Context, jid types.JID, state types.ChatPresence, media types.ChatPresenceMedia) error
}

// wrapNotFound returns ErrNotFound if err is sql.ErrNoRows (PocketBase's
// not-found signal), otherwise returns the original error unchanged.
func wrapNotFound(err error, msg string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	}
	return err
}

// StoreOutgoingMessage saves an outgoing message record. Errors are logged but
// not returned since message storage is best-effort.
func StoreOutgoingMessage(app core.App, businessID, phone, msgType, content string, media *filesystem.File) {
	collection, err := app.FindCollectionByNameOrId(domain.CollMessages)
	if err != nil {
		app.Logger().Error("storeOutgoingMessage: collection not found", "error", err)
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

func SendTextMessage(ctx context.Context, app core.App, waClient WAClient, params SendTextParams) error {
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

	stop := wa.Typing(ctx, waClient, jid)
	if _, err = waClient.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: &text,
	}); err != nil {
		stop()
		return err
	}
	stop()

	StoreOutgoingMessage(app, params.BusinessID, phone, domain.MsgTypeText, text, nil)

	if strings.TrimSpace(params.ProductionNote) != "" {
		time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)

		stop = wa.Typing(ctx, waClient, jid)
		noteText := "*Dica de foto:* " + params.ProductionNote
		if _, err := waClient.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &noteText,
		}); err != nil {
			app.Logger().Error("sendTextMessage: production note", "error", err)
		}
		stop()
		StoreOutgoingMessage(app, params.BusinessID, phone, domain.MsgTypeText, noteText, nil)

		// Posting time tip
		time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)
		stop = wa.Typing(ctx, waClient, jid)
		tipText := postingtime.Tip(business.GetString("type"))
		if _, err := waClient.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &tipText,
		}); err != nil {
			app.Logger().Error("sendTextMessage: posting time tip", "error", err)
		}
		stop()
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
				Caption:       new(params.Caption),
				Mimetype:      new(params.ContentType),
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
				Caption:       new(params.Caption),
				Mimetype:      new(params.ContentType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			},
		}
	}

	stop := wa.Typing(ctx, waClient, jid)
	defer stop()

	if _, err = waClient.SendMessage(ctx, jid, msg); err != nil {
		return err
	}

	msgType := domain.MsgTypeImage
	if isVideo {
		msgType = domain.MsgTypeVideo
	}
	mediaFile, err := filesystem.NewFileFromBytes(params.Data, params.Filename)
	if err != nil {
		app.Logger().Error("sendMediaMessage: create file from bytes", "error", err)
	}
	StoreOutgoingMessage(app, params.BusinessID, phone, msgType, params.Caption, mediaFile)

	return nil
}
