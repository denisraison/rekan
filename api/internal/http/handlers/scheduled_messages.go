package handlers

import (
	"net/http"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func ListScheduledMessages() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		msgs, _ := service.ListScheduledMessages(e.App)
		if msgs == nil {
			return e.JSON(http.StatusOK, []any{})
		}

		type msgResponse struct {
			ID       string `json:"id"`
			Business string `json:"business"`
			Text     string `json:"text"`
		}

		result := make([]msgResponse, len(msgs))
		for i, m := range msgs {
			result[i] = msgResponse{ID: m.ID, Business: m.Business, Text: m.Text}
		}
		return e.JSON(http.StatusOK, result)
	}
}

func ApproveScheduledMessage(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if deps.WhatsApp == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"message": "WhatsApp não configurado",
			})
		}

		msgID := e.Request.PathValue("id")
		if err := service.ApproveScheduledMessage(e.Request.Context(), e.App, deps.WhatsApp, msgID); err != nil {
			if err == service.ErrNoPhone {
				return e.JSON(http.StatusBadRequest, map[string]string{"message": "cliente sem telefone cadastrado"})
			}
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}

func DismissScheduledMessage() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		msgID := e.Request.PathValue("id")
		if err := service.DismissScheduledMessage(e.App, msgID); err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "mensagem não encontrada"})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
