package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func OperatorGenerate(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		var body struct {
			Message   string `json:"message"`
			MessageID string `json:"message_id"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "mensagem é obrigatória"})
		}

		post, err := service.GenerateFromMessage(e.Request.Context(), e.App, deps.GenerateFromMessage, businessID, body.Message, body.MessageID)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "Erro ao gerar conteúdo. Tente novamente.",
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":              post.ID,
			"caption":         post.Caption,
			"hashtags":        post.Hashtags,
			"production_note": post.ProductionNote,
			"role":            "",
			"hook":            post.Hook,
		})
	}
}
