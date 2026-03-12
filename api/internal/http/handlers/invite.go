package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func InviteSend(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		businessID := e.Request.PathValue("id")
		result, err := service.SendInvite(e.Request.Context(), e.App, deps.WhatsApp, deps.Asaas, businessID, deps.AppURL)
		if err != nil {
			if errors.Is(err, service.ErrNoPhone) {
				return e.JSON(http.StatusBadRequest, map[string]string{"message": "cliente sem telefone cadastrado"})
			}
			if errors.Is(err, service.ErrConflict) {
				return e.JSON(http.StatusConflict, map[string]string{"message": err.Error()})
			}
			return e.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"invite_url": result.InviteURL})
	}
}

func InviteGet() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")

		info, err := service.GetInviteInfo(e.App, token)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "convite não encontrado"})
		}

		if time.Since(info.SentAt) > 7*24*time.Hour {
			return e.JSON(http.StatusGone, map[string]string{"message": "convite expirado. Peça um novo ao seu gestor de conteúdo."})
		}

		resp := map[string]any{
			"business_name":     info.BusinessName,
			"client_name":      info.ClientName,
			"invite_status":    info.InviteStatus,
			"tier":             info.Tier,
			"commitment":       info.Commitment,
			"price":            info.Price,
			"commitment_months": info.CommitmentMonths,
		}

		if info.InviteStatus == domain.InviteStatusAccepted {
			resp["qr_payload"] = info.QRPayload
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

		var body struct {
			CpfCnpj string `json:"cpf_cnpj"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "corpo inválido"})
		}

		qrPayload, err := service.AcceptInvite(e.Request.Context(), e.App, deps.Asaas, token, body.CpfCnpj)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return e.JSON(http.StatusNotFound, map[string]string{"message": err.Error()})
			}
			if errors.Is(err, service.ErrConflict) {
				return e.JSON(http.StatusConflict, map[string]string{"message": err.Error()})
			}
			return e.JSON(http.StatusBadGateway, map[string]string{"message": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"qr_payload": qrPayload})
	}
}

func AuthorizationCancel(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.Asaas == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "pagamentos não configurados",
			})
		}

		businessID := e.Request.PathValue("id")
		if err := service.CancelAuthorization(e.Request.Context(), e.App, deps.Asaas, businessID); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return e.JSON(http.StatusNotFound, map[string]string{"message": err.Error()})
			}
			return e.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "assinatura cancelada"})
	}
}
