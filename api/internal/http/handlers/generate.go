package handlers

import (
	"errors"
	"net/http"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func GeneratePosts(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		result, err := service.GeneratePosts(e.Request.Context(), e.App, deps.Generate, businessID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
			}
			e.App.Logger().Error("generate posts failed", "business", businessID, "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "erro ao gerar conteúdo. Tente novamente."})
		}

		type postResponse struct {
			ID             string   `json:"id"`
			Caption        string   `json:"caption"`
			Hashtags       []string `json:"hashtags"`
			ProductionNote string   `json:"production_note"`
			Role           string   `json:"role"`
			Hook           string   `json:"hook"`
		}

		posts := make([]postResponse, len(result.Posts))
		for i, p := range result.Posts {
			posts[i] = postResponse{
				ID:             p.ID,
				Caption:        p.Caption,
				Hashtags:       p.Hashtags,
				ProductionNote: p.ProductionNote,
				Role:           p.Role,
				Hook:           p.Hook,
			}
		}

		return e.JSON(http.StatusOK, map[string]any{
			"batch_id": result.BatchID,
			"posts":    posts,
		})
	}
}
