package handlers

import (
	"encoding/json"
	"log"
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

		// Typing indicator (ban mitigation)
		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresenceComposing, "")

		// Random delay 1-3s to simulate human behavior
		delay := time.Duration(1000+rand.IntN(2000)) * time.Millisecond
		time.Sleep(delay)

		// Build message: caption + hashtags
		text := body.Caption
		if strings.TrimSpace(body.Hashtags) != "" {
			text += "\n\n" + body.Hashtags
		}

		// Send main message
		_, err = deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &text,
		})
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		// Store outgoing message
		collection, _ := e.App.FindCollectionByNameOrId(domain.CollMessages)
		if collection != nil {
			record := core.NewRecord(collection)
			record.Set("business", body.BusinessID)
			record.Set("phone", phone)
			record.Set("type", domain.MsgTypeText)
			record.Set("content", text)
			record.Set("direction", domain.DirectionOutgoing)
			record.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))
			if err := e.App.Save(record); err != nil {
				log.Printf("send_message: failed to save outgoing message: %v", err)
			}
		}

		// Send production note as separate message if present
		if strings.TrimSpace(body.ProductionNote) != "" {
			time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)

			noteText := "*Dica de foto:* " + body.ProductionNote
			deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: &noteText,
			})

			// Store production note as outgoing message
			if collection != nil {
				noteRecord := core.NewRecord(collection)
				noteRecord.Set("business", body.BusinessID)
				noteRecord.Set("phone", phone)
				noteRecord.Set("type", domain.MsgTypeText)
				noteRecord.Set("content", noteText)
				noteRecord.Set("direction", domain.DirectionOutgoing)
				noteRecord.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))
				if err := e.App.Save(noteRecord); err != nil {
					log.Printf("send_message: failed to save production note message: %v", err)
				}
			}
		}

		// Send posting time tip
		time.Sleep(time.Duration(500+rand.IntN(1000)) * time.Millisecond)
		tipText := postingtime.Tip(business.GetString("type"))
		deps.WhatsApp.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: &tipText,
		})
		if collection != nil {
			tipRecord := core.NewRecord(collection)
			tipRecord.Set("business", body.BusinessID)
			tipRecord.Set("phone", phone)
			tipRecord.Set("type", domain.MsgTypeText)
			tipRecord.Set("content", tipText)
			tipRecord.Set("direction", domain.DirectionOutgoing)
			tipRecord.Set("wa_timestamp", time.Now().UTC().Format(time.RFC3339))
			if err := e.App.Save(tipRecord); err != nil {
				log.Printf("send_message: failed to save tip message: %v", err)
			}
		}

		// Clear typing indicator
		deps.WhatsApp.SendChatPresence(ctx, jid, types.ChatPresencePaused, "")

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
