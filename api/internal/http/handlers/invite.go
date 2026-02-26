package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	asaasclient "github.com/denisraison/rekan/api/internal/asaas"
	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/pricing"
	"github.com/denisraison/rekan/api/internal/terms"
	"github.com/pocketbase/pocketbase/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

var errAlreadyClaimed = errors.New("invite already claimed")

func InviteSend(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")
		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
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

		tier := business.GetString("tier")
		commitment := business.GetString("commitment")
		if !pricing.ValidTier(tier) || !pricing.ValidCommitment(commitment) {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "plano e compromisso devem ser definidos antes de enviar o convite"})
		}

		status := business.GetString("invite_status")
		switch status {
		case domain.InviteStatusAccepted, domain.InviteStatusActive:
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
		business.Set("invite_status", domain.InviteStatusInvited)
		business.Set("invite_sent_at", now)
		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		// Store outgoing message
		collection, _ := e.App.FindCollectionByNameOrId(domain.CollMessages)
		if collection != nil {
			record := core.NewRecord(collection)
			record.Set("business", businessID)
			record.Set("phone", phone)
			record.Set("type", domain.MsgTypeText)
			record.Set("content", text)
			record.Set("direction", domain.DirectionOutgoing)
			record.Set("wa_timestamp", now)
			e.App.Save(record)
		}

		return e.JSON(http.StatusOK, map[string]string{"invite_url": inviteURL})
	}
}

func InviteGet(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")

		business, err := e.App.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "convite não encontrado"})
		}

		sentAt := business.GetDateTime("invite_sent_at").Time()
		if time.Since(sentAt) > 7*24*time.Hour {
			return e.JSON(http.StatusGone, map[string]string{"message": "convite expirado. Peça um novo ao seu gestor de conteúdo."})
		}

		tier := pricing.Tier(business.GetString("tier"))
		commitment := pricing.Commitment(business.GetString("commitment"))
		price, _ := pricing.Price(tier, commitment)
		months := pricing.Months[commitment]

		resp := map[string]any{
			"business_name":    business.GetString("name"),
			"client_name":     business.GetString("client_name"),
			"invite_status":   business.GetString("invite_status"),
			"tier":            string(tier),
			"commitment":      string(commitment),
			"price":           price,
			"commitment_months": months,
		}

		if business.GetString("invite_status") == domain.InviteStatusAccepted {
			resp["qr_payload"] = business.GetString("qr_payload")
		}

		return e.JSON(http.StatusOK, resp)
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

		business, err := e.App.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "convite não encontrado"})
		}

		sentAt := business.GetDateTime("invite_sent_at").Time()
		if time.Since(sentAt) > 7*24*time.Hour {
			return e.JSON(http.StatusGone, map[string]string{"message": "convite expirado. Peça um novo ao seu gestor de conteúdo."})
		}

		status := business.GetString("invite_status")

		// Idempotency: already accepted with authorization, return stored qr_payload
		if status == domain.InviteStatusAccepted && business.GetString("authorization_id") != "" {
			return e.JSON(http.StatusOK, map[string]string{"qr_payload": business.GetString("qr_payload")})
		}

		if status == domain.InviteStatusActive {
			return e.JSON(http.StatusConflict, map[string]string{"message": "assinatura já está ativa"})
		}

		if status != domain.InviteStatusInvited {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "convite não pode ser aceito neste estado"})
		}

		tier := pricing.Tier(business.GetString("tier"))
		commitment := pricing.Commitment(business.GetString("commitment"))
		price, ok := pricing.Price(tier, commitment)
		if !ok {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "plano ou compromisso inválido"})
		}

		var body struct {
			CpfCnpj string `json:"cpf_cnpj"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "corpo inválido"})
		}

		// Atomically claim this invite: re-read inside the transaction and
		// transition "invited" -> "accepted" to prevent concurrent requests
		// from both calling Asaas and creating duplicate authorizations.
		err = e.App.RunInTransaction(func(txApp core.App) error {
			fresh, err := txApp.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
			if err != nil {
				return err
			}
			if fresh.GetString("invite_status") != domain.InviteStatusInvited {
				return errAlreadyClaimed
			}
			fresh.Set("invite_status", domain.InviteStatusAccepted)
			fresh.Set("terms_accepted_at", time.Now().UTC().Format(time.RFC3339))
			fresh.Set("terms_accepted_text", terms.Snapshot(tier, commitment, price))
			return txApp.Save(fresh)
		})
		if errors.Is(err, errAlreadyClaimed) {
			// Concurrent request claimed it. Check if it finished.
			business, _ = e.App.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
			if business != nil && business.GetString("authorization_id") != "" {
				return e.JSON(http.StatusOK, map[string]string{"qr_payload": business.GetString("qr_payload")})
			}
			return e.JSON(http.StatusConflict, map[string]string{"message": "convite está sendo processado"})
		}
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		// Status is "accepted" in DB. Concurrent requests are blocked.

		// Reuse existing Asaas customer on retry (CreateAuthorization may
		// have failed after CreateCustomer succeeded on a previous attempt).
		customerID := business.GetString("customer_id")
		if customerID == "" {
			clientName := business.GetString("client_name")
			clientEmail := business.GetString("client_email")
			customer, err := deps.Asaas.CreateCustomer(e.Request.Context(), clientName, clientEmail, body.CpfCnpj)
			if err != nil {
				e.App.Logger().Error("asaas create customer", "error", err)
				revertToInvited(e.App, token)
				return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao criar conta de pagamento"})
			}
			customerID = customer.ID
			// Persist customer_id immediately so retries reuse it
			if fresh, err := e.App.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token}); err == nil {
				fresh.Set("customer_id", customerID)
				_ = e.App.Save(fresh)
			}
		}

		dueDate := time.Now().Format("2006-01-02")
		auth, err := deps.Asaas.CreateAuthorization(e.Request.Context(), asaasclient.CreateAuthorizationReq{
			CustomerID:  customerID,
			Description: "Rekan - " + string(tier),
			Frequency:   pricing.AsaasFrequency[commitment],
			ContractID:  business.Id,
			StartDate:   dueDate,
			ImmediateQrCode: asaasclient.ImmediateQrCodeReq{
				Value:             price,
				OriginalValue:     price,
				DueDate:           dueDate,
				ExpirationSeconds: 86400, // 24h
			},
		})
		if err != nil {
			e.App.Logger().Error("asaas create authorization", "error", err)
			revertToInvited(e.App, token)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao criar autorização de pagamento"})
		}

		qrPayload := auth.Payload

		// Re-read fresh record (the transaction saved a different in-memory copy).
		business, err = e.App.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}
		business.Set("authorization_id", auth.ID)
		business.Set("customer_id", customerID)
		business.Set("qr_payload", qrPayload)
		if err := e.App.Save(business); err != nil {
			e.App.Logger().Error("save authorization details", "error", err)
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		return e.JSON(http.StatusOK, map[string]string{"qr_payload": qrPayload})
	}
}

// revertToInvited is a best-effort rollback when Asaas calls fail after
// the status was atomically set to "accepted". Allows the user to retry.
func revertToInvited(app core.App, token string) {
	biz, err := app.FindFirstRecordByFilter(domain.CollBusinesses, "invite_token = {:token}", map[string]any{"token": token})
	if err != nil {
		return
	}
	biz.Set("invite_status", domain.InviteStatusInvited)
	_ = app.Save(biz)
}

func AuthorizationCancel(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados",
			})
		}

		businessID := e.Request.PathValue("id")
		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}
		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		authID := business.GetString("authorization_id")
		if authID == "" || business.GetString("invite_status") != domain.InviteStatusActive {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "nenhuma assinatura ativa"})
		}

		if err := deps.Asaas.CancelAuthorization(e.Request.Context(), authID); err != nil {
			e.App.Logger().Error("asaas cancel authorization", "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao cancelar assinatura"})
		}

		business.Set("invite_status", domain.InviteStatusCancelled)
		if err := e.App.Save(business); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro interno"})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "assinatura cancelada"})
	}
}
