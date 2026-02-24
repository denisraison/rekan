package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	asaasclient "github.com/denisraison/rekan/api/internal/asaas"
	"github.com/pocketbase/pocketbase/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

const PriceParceiro = 108.90

func InviteSend(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")
		business, err := e.App.FindRecordById("businesses", businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}
		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		phone := business.GetString("phone")
		if phone == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "cliente sem telefone cadastrado"})
		}

		status := business.GetString("invite_status")
		switch status {
		case "accepted", "active":
			return e.JSON(http.StatusConflict, map[string]string{"message": "convite já aceito ou assinatura ativa"})
		}

		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados",
			})
		}

		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}
		token := hex.EncodeToString(tokenBytes)

		inviteURL := deps.AppURL + "/convite/" + token
		clientName := business.GetString("client_name")
		text := "Oi " + clientName + "! Segue o link pra ativar seu acesso ao Rekan: " + inviteURL

		jid := types.NewJID(phone, types.DefaultUserServer)
		_, err = deps.WhatsApp.SendMessage(e.Request.Context(), jid, &waE2E.Message{
			Conversation: &text,
		})
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao enviar mensagem"})
		}

		now := time.Now().UTC().Format(time.RFC3339)
		business.Set("invite_token", token)
		business.Set("invite_status", "invited")
		business.Set("invite_sent_at", now)
		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		// Store outgoing message
		collection, _ := e.App.FindCollectionByNameOrId("messages")
		if collection != nil {
			record := core.NewRecord(collection)
			record.Set("business", businessID)
			record.Set("phone", phone)
			record.Set("type", "text")
			record.Set("content", text)
			record.Set("direction", "outgoing")
			record.Set("wa_timestamp", now)
			e.App.Save(record)
		}

		return e.JSON(http.StatusOK, map[string]string{"invite_url": inviteURL})
	}
}

func InviteGet(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")

		business, err := e.App.FindFirstRecordByFilter("businesses", "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "convite não encontrado"})
		}

		sentAt := business.GetDateTime("invite_sent_at").Time()
		if time.Since(sentAt) > 7*24*time.Hour {
			return e.JSON(http.StatusGone, map[string]string{"message": "convite expirado. Peça um novo ao seu gestor de conteúdo."})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"business_name": business.GetString("name"),
			"client_name":   business.GetString("client_name"),
			"invite_status": business.GetString("invite_status"),
			"price_monthly":  PriceParceiro,
		})
	}
}

func InviteAccept(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados",
			})
		}

		token := e.Request.PathValue("token")

		business, err := e.App.FindFirstRecordByFilter("businesses", "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "convite não encontrado"})
		}

		sentAt := business.GetDateTime("invite_sent_at").Time()
		if time.Since(sentAt) > 7*24*time.Hour {
			return e.JSON(http.StatusGone, map[string]string{"message": "convite expirado. Peça um novo ao seu gestor de conteúdo."})
		}

		status := business.GetString("invite_status")

		// Idempotency: already accepted with subscription, return existing payment link
		if status == "accepted" && business.GetString("subscription_id") != "" {
			sub, err := deps.Asaas.GetSubscription(e.Request.Context(), business.GetString("subscription_id"))
			if err != nil {
				return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao buscar assinatura"})
			}
			return e.JSON(http.StatusOK, map[string]string{"payment_url": sub.PaymentLink})
		}

		if status == "active" {
			return e.JSON(http.StatusConflict, map[string]string{"message": "assinatura já está ativa"})
		}

		if status != "invited" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "convite não pode ser aceito neste estado"})
		}

		var body struct {
			CpfCnpj string `json:"cpf_cnpj"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "corpo inválido"})
		}

		// Save terms acceptance timestamp before Asaas calls.
		// Status stays "invited" until Asaas succeeds, so retries work.
		business.Set("terms_accepted_at", time.Now().UTC().Format(time.RFC3339))
		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		clientName := business.GetString("client_name")
		clientEmail := business.GetString("client_email")

		customer, err := deps.Asaas.CreateCustomer(e.Request.Context(), clientName, clientEmail, body.CpfCnpj)
		if err != nil {
			e.App.Logger().Error("asaas create customer", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao criar conta de pagamento"})
		}

		sub, err := deps.Asaas.CreateSubscription(e.Request.Context(), asaasclient.CreateSubscriptionReq{
			Customer:          customer.ID,
			BillingType:       "PIX",
			Value:             PriceParceiro,
			NextDueDate:       time.Now().Format("2006-01-02"),
			Cycle:             "MONTHLY",
			Description:       "Rekan - Parceiro",
			ExternalReference: business.Id,
			Callback: &asaasclient.Callback{
				SuccessURL:   deps.AppURL + "/convite/" + token + "/confirmacao",
				AutoRedirect: true,
			},
		})
		if err != nil {
			e.App.Logger().Error("asaas create subscription", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao criar assinatura"})
		}

		// Both Asaas calls succeeded. Set accepted + subscription_id atomically.
		business.Set("invite_status", "accepted")
		business.Set("subscription_id", sub.ID)
		if err := e.App.Save(business); err != nil {
			e.App.Logger().Error("save accepted status", "error", err)
		}

		return e.JSON(http.StatusOK, map[string]string{"payment_url": sub.PaymentLink})
	}
}

func SubscriptionCancel(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados",
			})
		}

		businessID := e.Request.PathValue("id")
		business, err := e.App.FindRecordById("businesses", businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}
		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		subID := business.GetString("subscription_id")
		if subID == "" || business.GetString("invite_status") != "active" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "nenhuma assinatura ativa"})
		}

		if err := deps.Asaas.CancelSubscription(e.Request.Context(), subID); err != nil {
			e.App.Logger().Error("asaas cancel subscription", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao cancelar assinatura"})
		}

		business.Set("invite_status", "cancelled")
		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "assinatura cancelada"})
	}
}
