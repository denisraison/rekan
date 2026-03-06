package handlers

import (
	"context"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/whatsapp"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// formFileData reads the named form file field, returning its bytes, detected
// content type, and original header. The caller must not close the file.
func formFileData(r *http.Request, field string) ([]byte, string, *multipart.FileHeader, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return nil, "", nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", nil, err
	}

	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return data, ct, header, nil
}

// simulateTyping shows a typing indicator, waits 1-3s, runs fn, then clears
// the indicator. The typing indicator is best-effort (errors are ignored).
func simulateTyping(ctx context.Context, wa *whatsapp.Client, jid types.JID, fn func() error) error {
	wa.SendChatPresence(ctx, jid, types.ChatPresenceComposing, "")
	delay := time.Duration(1000+rand.IntN(2000)) * time.Millisecond
	time.Sleep(delay)
	err := fn()
	wa.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")
	return err
}

// storeOutgoingMessage saves an outgoing message record. Errors are logged but
// not returned since message storage is best-effort.
func storeOutgoingMessage(app core.App, businessID, phone, msgType, content string, media *filesystem.File) {
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
