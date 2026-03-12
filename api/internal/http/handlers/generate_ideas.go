package handlers

import (
	"net/http"

	"github.com/denisraison/rekan/api/internal/service"
	"github.com/pocketbase/pocketbase/core"
)

func GenerateIdeas(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		posts, err := service.GenerateIdeas(e.Request.Context(), e.App, deps.Generate, businessID)
		if err != nil {
			e.App.Logger().Error("generate ideas failed", "business", businessID, "error", err)
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Erro ao gerar conteúdo. Tente novamente."})
		}

		type postResponse struct {
			Caption        string   `json:"caption"`
			Hashtags       []string `json:"hashtags"`
			ProductionNote string   `json:"production_note"`
		}

		result := make([]postResponse, len(posts))
		for i, p := range posts {
			result[i] = postResponse{
				Caption:        p.Caption,
				Hashtags:       p.Hashtags,
				ProductionNote: p.ProductionNote,
			}
		}

		return e.JSON(http.StatusOK, result)
	}
}
