package handlers

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/postingtime"
	"github.com/pocketbase/pocketbase/core"
)

// SendMessage sends a WhatsApp message to a business's phone number
// and stores it as an outgoing message.
func SendMessage(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}

		var body struct {
			BusinessID     string `json:"business_id"`
			Caption        string `json:"caption"`
			Hashtags       string `json:"hashtags"`
			ProductionNote string `json:"production_note"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Corpo inválido"})
		}

		if strings.TrimSpace(body.Caption) == "" || strings.TrimSpace(body.BusinessID) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Negócio e legenda são obrigatórios"})
		}

		business, err := e.App.FindRecordById(domain.CollBusinesses, body.BusinessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "Negócio não encontrado"})
		}

		phone := business.GetString("phone")
		if phone == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Cliente sem telefone cadastrado"})
		}

		jid := types.NewJID(phone, types.DefaultUserServer)
		ctx := e.Request.Context()

		// Build message: caption + hashtags
		text := body.Caption
		if strings.TrimSpace(body.Hashtags) != "" {
			text += "\n\n" + body.Hashtags
		}

		// Send with typing indicator
		err = simulateTyping(ctx, deps.WhatsApp, jid, func() error {
			_, err := deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: &text,
			})
			return err
		})
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		storeOutgoingMessage(e.App, body.BusinessID, phone, domain.MsgTypeText, text, nil)

		// Send production note as separate message if present
		if strings.TrimSpace(body.ProductionNote) != "" {
			time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)

			noteText := "*Dica de foto:* " + body.ProductionNote
			deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: &noteText,
			})
			storeOutgoingMessage(e.App, body.BusinessID, phone, domain.MsgTypeText, noteText, nil)
		}

		// Send posting time tip only when delivering a generated post (has production note)
		if strings.TrimSpace(body.ProductionNote) != "" {
			time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)
			tipText := postingtime.Tip(business.GetString("type"))
			deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: &tipText,
			})
			storeOutgoingMessage(e.App, body.BusinessID, phone, domain.MsgTypeText, tipText, nil)
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
