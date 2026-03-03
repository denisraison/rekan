package handlers

import (
	"fmt"
	"net/http"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/operator"
	"github.com/denisraison/rekan/eval"
	"github.com/pocketbase/pocketbase/core"
)

// GenerateIdeas generates 3 post ideas for a business without saving them.
func GenerateIdeas(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		business, err := e.App.FindRecordById(domain.CollBusinesses, businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}

		profile, err := operator.BusinessToProfile(business)
		if err != nil {
			return fmt.Errorf("business to profile: %w", err)
		}

		previousHooks, err := operator.LoadPreviousHooks(e.App, businessID)
		if err != nil {
			return fmt.Errorf("load previous hooks: %w", err)
		}

		posts, err := deps.Generate(e.Request.Context(), profile, eval.PickRoles(3, nil), previousHooks)
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
