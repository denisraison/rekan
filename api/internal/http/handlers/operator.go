package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

func OperatorGenerate(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}

		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		var body struct {
			Message   string `json:"message"`
			MessageID string `json:"message_id"` // optional: link to source message record
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "mensagem é obrigatória"})
		}

		profile, err := businessToProfile(business)
		if err != nil {
			return fmt.Errorf("business to profile: %w", err)
		}

		previousHooks, err := loadPreviousHooks(e.App, businessID)
		if err != nil {
			return fmt.Errorf("load previous hooks: %w", err)
		}

		post, err := deps.GenerateFromMessage(e.Request.Context(), profile, body.Message, previousHooks)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "Erro ao gerar conteúdo. Tente novamente.",
			})
		}

		// Save to posts collection
		hook := eval.ExtractHooks([]eval.Post{post})
		hookStr := ""
		if len(hook) > 0 {
			hookStr = hook[0]
		}

		collection, err := e.App.FindCollectionByNameOrId(domain.CollPosts)
		if err != nil {
			return fmt.Errorf("find posts collection: %w", err)
		}

		record := core.NewRecord(collection)
		record.Set("business", businessID)
		record.Set("caption", post.Caption)
		record.Set("hashtags", post.Hashtags)
		record.Set("production_note", post.ProductionNote)
		record.Set("hook", hookStr)
		record.Set("source", domain.PostSourceOperator)
		record.Set("edited", false)
		if body.MessageID != "" {
			record.Set("message", body.MessageID)
		}

		if err := e.App.Save(record); err != nil {
			return fmt.Errorf("save operator post: %w", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":              record.Id,
			"caption":         post.Caption,
			"hashtags":        post.Hashtags,
			"production_note": post.ProductionNote,
			"role":            "",
			"hook":            hookStr,
		})
	}
}
