package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/pocketbase/pocketbase/core"
)

// SaveProactivePost saves a proactively generated post to the posts collection.
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

		collection, err := e.App.FindCollectionByNameOrId(domain.CollPosts)
		if err != nil {
			return fmt.Errorf("find posts collection: %w", err)
		}

		record := core.NewRecord(collection)
		record.Set("business", businessID)
		record.Set("caption", body.Caption)
		record.Set("hashtags", body.Hashtags)
		record.Set("production_note", body.ProductionNote)
		record.Set("source", domain.PostSourceProactive)
		record.Set("edited", false)

		if err := e.App.Save(record); err != nil {
			return fmt.Errorf("save proactive post: %w", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":              record.Id,
			"caption":         body.Caption,
			"hashtags":        body.Hashtags,
			"production_note": body.ProductionNote,
		})
	}
}
