package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func SaveProactivePost() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		var body struct {
			Caption        string   `json:"caption"`
			Hashtags       []string `json:"hashtags"`
			ProductionNote string   `json:"production_note"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "corpo inválido"})
		}

		id, err := service.SaveProactivePost(e.App, service.SaveProactiveParams{
			BusinessID:     businessID,
			Caption:        body.Caption,
			Hashtags:       body.Hashtags,
			ProductionNote: body.ProductionNote,
		})
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": "erro ao salvar"})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":              id,
			"caption":         body.Caption,
			"hashtags":        body.Hashtags,
			"production_note": body.ProductionNote,
		})
	}
}
