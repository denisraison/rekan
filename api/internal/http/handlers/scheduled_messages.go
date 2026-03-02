package handlers

import (
	"math/rand/v2"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// ListScheduledMessages returns all pending (not approved, not dismissed) scheduled messages.
func ListScheduledMessages() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		records, err := e.App.FindRecordsByFilter(
			domain.CollScheduledMessages,
			"approved = false && dismissed = false",
			"-created",
			0,
			0,
		)
		if err != nil {
			return e.JSON(http.StatusOK, []any{})
		}

		type msgResponse struct {
			ID       string `json:"id"`
			Business string `json:"business"`
			Text     string `json:"text"`
		}

		result := make([]msgResponse, 0, len(records))
		for _, r := range records {
			result = append(result, msgResponse{
				ID:       r.Id,
				Business: r.GetString("business"),
				Text:     r.GetString("text"),
			})
		}

		return e.JSON(http.StatusOK, result)
	}
}

// ApproveScheduledMessage sends the scheduled message via WhatsApp and marks it approved.
func ApproveScheduledMessage(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}

		msgID := e.Request.PathValue("id")

		record, err := e.App.FindRecordById(domain.CollScheduledMessages, msgID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "mensagem não encontrada"})
		}

		businessID := record.GetString("business")
		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}

		phone := business.GetString("phone")
		if phone == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "cliente sem telefone cadastrado"})
		}

		text := record.GetString("text")
		jid := types.NewJID(phone, types.DefaultUserServer)
		ctx := e.Request.Context()

		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresenceComposing, "")
		delay := time.Duration(1000+rand.IntN(2000)) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return e.JSON(http.StatusRequestTimeout, map[string]string{"message": "cancelado"})
		}

		_, err = deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &text,
		})
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}
		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")

		// Store outgoing message
		msgCollection, _ := e.App.FindCollectionByNameOrId(domain.CollMessages)
		if msgCollection != nil {
			msgRecord := core.NewRecord(msgCollection)
			msgRecord.Set("business", businessID)
			msgRecord.Set("phone", phone)
			msgRecord.Set("type", domain.MsgTypeText)
			msgRecord.Set("content", text)
			msgRecord.Set("direction", domain.DirectionOutgoing)
			msgRecord.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))
			if err := e.App.Save(msgRecord); err != nil {
				e.App.Logger().Error("scheduled_messages: failed to save outgoing message", "error", err)
			}
		}

		record.Set("approved", true)
		if err := e.App.Save(record); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro ao salvar aprovação"})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}

// DismissScheduledMessage marks a scheduled message as dismissed.
func DismissScheduledMessage() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		msgID := e.Request.PathValue("id")

		record, err := e.App.FindRecordById(domain.CollScheduledMessages, msgID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "mensagem não encontrada"})
		}

		record.Set("dismissed", true)
		if err := e.App.Save(record); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro ao salvar"})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
