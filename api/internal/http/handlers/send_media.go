package handlers

import (
	"net/http"
	"strings"

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

		data, contentType, header, err := formFileData(e.Request, "file")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Arquivo é obrigatório"})
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

		isVideo := strings.HasPrefix(contentType, "video/")

		var waMediaType whatsmeow.MediaType
		if isVideo {
			waMediaType = whatsmeow.MediaVideo
		} else {
			waMediaType = whatsmeow.MediaImage
		}

		resp, err := deps.WhatsApp.Upload(ctx, data, waMediaType)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mídia. Tente novamente."})
		}

		// Build message
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

		err = simulateTyping(ctx, deps.WhatsApp, jid, func() error {
			_, err := deps.WhatsApp.SendMessage(ctx, jid, msg)
			return err
		})
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		// Store outgoing message
		msgType := domain.MsgTypeImage
		if isVideo {
			msgType = domain.MsgTypeVideo
		}
		mediaFile, _ := filesystem.NewFileFromBytes(data, header.Filename)
		storeOutgoingMessage(e.App, businessID, phone, msgType, caption, mediaFile)

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
