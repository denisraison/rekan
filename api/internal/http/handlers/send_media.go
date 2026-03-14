package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

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

		err = service.SendMediaMessage(e.Request.Context(), e.App, deps.WhatsApp, service.SendMediaParams{
			BusinessID:  businessID,
			Caption:     caption,
			Data:        data,
			ContentType: contentType,
			Filename:    header.Filename,
		})
		if err != nil {
			if errors.Is(err, service.ErrNoPhone) {
				return e.JSON(http.StatusBadRequest, map[string]string{"message": "Cliente sem telefone cadastrado"})
			}
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao enviar mídia. Tente novamente."})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "sent"})
	}
}
