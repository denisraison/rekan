package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

func OperatorGenerate(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		business, err := e.App.FindRecordById("businesses", businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}

		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		var body struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "mensagem é obrigatória"})
		}

		profile, err := businessToProfile(business)
		if err != nil {
			return fmt.Errorf("business to profile: %w", err)
		}

		post, err := eval.GenerateFromMessage(e.Request.Context(), profile, body.Message, nil)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "erro ao gerar conteúdo. Tente novamente.",
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"caption":         post.Caption,
			"hashtags":        post.Hashtags,
			"production_note": post.ProductionNote,
		})
	}
}
