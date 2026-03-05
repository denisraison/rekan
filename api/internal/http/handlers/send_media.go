package handlers

import (
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// SendMedia sends a WhatsApp media message (image or video) to a business's phone number.
func SendMedia(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}

		businessID := strings.TrimSpace(e.Request.FormValue("business_id"))
		caption := strings.TrimSpace(e.Request.FormValue("caption"))
		if businessID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "business_id é obrigatório"})
		}

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Arquivo é obrigatório"})
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Erro ao ler arquivo"})
		}

		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "Negócio não encontrado"})
		}

		phone := business.GetString("phone")
		if phone == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Cliente sem telefone cadastrado"})
		}

		jid := types.NewJID(phone, types.DefaultUserServer)
		ctx := e.Request.Context()

		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		isVideo := strings.HasPrefix(contentType, "video/")

		var waMediaType whatsmeow.MediaType
		if isVideo {
			waMediaType = whatsmeow.MediaVideo
		} else {
			waMediaType = whatsmeow.MediaImage
		}

		// Upload to WhatsApp servers
		resp, err := deps.WhatsApp.Upload(ctx, data, waMediaType)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mídia. Tente novamente."})
		}

		// Typing indicator
		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresenceComposing, "")
		delay := time.Duration(1000+rand.IntN(2000)) * time.Millisecond
		time.Sleep(delay)

		// Build and send message
		var msg *waE2E.Message
		if isVideo {
			msg = &waE2E.Message{
				VideoMessage: &waE2E.VideoMessage{
					Caption:       proto.String(caption),
					Mimetype:      proto.String(contentType),
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
					Caption:       proto.String(caption),
					Mimetype:      proto.String(contentType),
					URL:           &resp.URL,
					DirectPath:    &resp.DirectPath,
					MediaKey:      resp.MediaKey,
					FileEncSHA256: resp.FileEncSHA256,
					FileSHA256:    resp.FileSHA256,
					FileLength:    &resp.FileLength,
				},
			}
		}

		_, err = deps.WhatsApp.SendMessage(ctx, jid, msg)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		// Store outgoing message
		msgType := domain.MsgTypeImage
		if isVideo {
			msgType = domain.MsgTypeVideo
		}

		collection, _ := e.App.FindCollectionByNameOrId(domain.CollMessages)
		if collection != nil {
			record := core.NewRecord(collection)
			record.Set("business", businessID)
			record.Set("phone", phone)
			record.Set("type", msgType)
			record.Set("content", caption)
			record.Set("direction", domain.DirectionOutgoing)
			record.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))

			mediaFile, err := filesystem.NewFileFromBytes(data, header.Filename)
			if err == nil {
				record.Set("media", mediaFile)
			}

			if err := e.App.Save(record); err != nil {
				e.App.Logger().Error("send_media: failed to save outgoing message", "error", err)
			}
		}

		// Clear typing indicator
		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
