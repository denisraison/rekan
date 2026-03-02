package handlers

import (
	"fmt"
	"net/http"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/operator"
	"github.com/denisraison/rekan/eval"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

func GeneratePosts(deps Deps) func(*core.RequestEvent) error {
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

		roles := eval.PickRoles(3, nil)

		previousHooks, err := operator.LoadPreviousHooks(e.App, businessID)
		if err != nil {
			return fmt.Errorf("load previous hooks: %w", err)
		}

		posts, err := deps.Generate(e.Request.Context(), profile, roles, previousHooks)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "erro ao gerar conteúdo. Tente novamente.",
			})
		}

		hooks := eval.ExtractHooks(posts)

		batchID := uuid.New().String()
		collection, err := e.App.FindCollectionByNameOrId(domain.CollPosts)
		if err != nil {
			return fmt.Errorf("find posts collection: %w", err)
		}

		type postResponse struct {
			ID             string   `json:"id"`
			Caption        string   `json:"caption"`
			Hashtags       []string `json:"hashtags"`
			ProductionNote string   `json:"production_note"`
			Role           string   `json:"role"`
			Hook           string   `json:"hook"`
		}

		savedPosts := make([]postResponse, 0, len(posts))
		for i, post := range posts {
			record := core.NewRecord(collection)
			record.Set("business", businessID)
			record.Set("caption", post.Caption)
			record.Set("hashtags", post.Hashtags)
			record.Set("production_note", post.ProductionNote)
			record.Set("edited", false)
			record.Set("batch_id", batchID)

			roleName := ""
			if i < len(roles) {
				roleName = roles[i].Name
				record.Set("role", roleName)
			}

			hook := ""
			if i < len(hooks) {
				hook = hooks[i]
				record.Set("hook", hook)
			}

			if err := e.App.Save(record); err != nil {
				return fmt.Errorf("save post %d: %w", i, err)
			}

			savedPosts = append(savedPosts, postResponse{
				ID:             record.Id,
				Caption:        post.Caption,
				Hashtags:       post.Hashtags,
				ProductionNote: post.ProductionNote,
				Role:           roleName,
				Hook:           hook,
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"batch_id": batchID,
			"posts":    savedPosts,
		})
	}
}

