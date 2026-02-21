package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/denisraison/rekan/eval"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// trialGenerationLimit is the number of free generations before a subscription is required.
const trialGenerationLimit = 3

// storedService matches the shape the frontend writes to PocketBase (snake_case).
type storedService struct {
	Name     string  `json:"name"`
	PriceBRL float64 `json:"price_brl"`
}

func GeneratePosts(deps Deps) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		businessID := e.Request.PathValue("id")

		business, err := e.App.FindRecordById("businesses", businessID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"message": "negócio não encontrado"})
		}

		if business.GetString("user") != e.Auth.Id {
			return e.JSON(http.StatusForbidden, map[string]string{"message": "acesso negado"})
		}

		status := e.Auth.GetString("subscription_status")
		generationsUsed := e.Auth.GetInt("generations_used")

		switch {
		case status == "trial" && generationsUsed >= trialGenerationLimit:
			return e.JSON(http.StatusPaymentRequired, map[string]string{
				"message": "período de teste encerrado. Assine para continuar gerando conteúdo.",
			})
		case status != "trial" && status != "active":
			return e.JSON(http.StatusPaymentRequired, map[string]string{
				"message": "assinatura inativa.",
			})
		}

		profile, err := businessToProfile(business)
		if err != nil {
			return fmt.Errorf("business to profile: %w", err)
		}

		roles := eval.PickRoles(3, nil)

		previousHooks, err := loadPreviousHooks(e.App, businessID)
		if err != nil {
			return fmt.Errorf("load previous hooks: %w", err)
		}

		posts, err := eval.Generate(e.Request.Context(), profile, roles, previousHooks)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{
				"message": "erro ao gerar conteúdo. Tente novamente.",
			})
		}

		hooks := eval.ExtractHooks(posts)

		batchID := uuid.New().String()
		collection, err := e.App.FindCollectionByNameOrId("posts")
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

		if status == "trial" {
			e.Auth.Set("generations_used", generationsUsed+1)
			if err := e.App.Save(e.Auth); err != nil {
				e.App.Logger().Error("failed to increment generations_used", "error", err)
			}
		}

		return e.JSON(http.StatusOK, map[string]any{
			"batch_id": batchID,
			"posts":    savedPosts,
		})
	}
}

func businessToProfile(record *core.Record) (eval.BusinessProfile, error) {
	raw := record.Get("services")
	b, err := json.Marshal(raw)
	if err != nil {
		return eval.BusinessProfile{}, fmt.Errorf("marshal services: %w", err)
	}
	var stored []storedService
	if err := json.Unmarshal(b, &stored); err != nil {
		return eval.BusinessProfile{}, fmt.Errorf("unmarshal services: %w", err)
	}
	services := make([]eval.Service, len(stored))
	for i, s := range stored {
		services[i] = eval.Service{Name: s.Name, PriceBRL: s.PriceBRL}
	}

	var quirks []string
	for _, line := range strings.Split(record.GetString("quirks"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			quirks = append(quirks, line)
		}
	}

	return eval.BusinessProfile{
		BusinessName:   record.GetString("name"),
		BusinessType:   record.GetString("type"),
		City:           record.GetString("city"),
		Neighbourhood:  "", // collection has state, not neighbourhood; city is sufficient for generation
		Services:       services,
		TargetAudience: record.GetString("target_audience"),
		BrandVibe:      record.GetString("brand_vibe"),
		Quirks:         quirks,
	}, nil
}

func loadPreviousHooks(app core.App, businessID string) ([]string, error) {
	records, err := app.FindAllRecords("posts", dbx.HashExp{"business": businessID})
	if err != nil {
		return nil, err
	}
	hooks := make([]string, 0, len(records))
	for _, r := range records {
		if h := r.GetString("hook"); h != "" {
			hooks = append(hooks, h)
		}
	}
	return hooks, nil
}
