package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

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

		err := service.SendTextMessage(e.Request.Context(), e.App, deps.WhatsApp, service.SendTextParams{
			BusinessID:     body.BusinessID,
			Caption:        body.Caption,
			Hashtags:       body.Hashtags,
			ProductionNote: body.ProductionNote,
		})
		if err != nil {
			if err == service.ErrNoPhone {
				return e.JSON(http.StatusBadRequest, map[string]string{"message": "Cliente sem telefone cadastrado"})
			}
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mensagem. Tente novamente."})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
