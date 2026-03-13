package service

import (
	"context"
	"fmt"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/operator"
	content "github.com/denisraison/rekan/api/internal/content"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

type GeneratedPost struct {
	ID             string
	Caption        string
	Hashtags       []string
	ProductionNote string
	Role           string
	Hook           string
}

type GenerateBatchResult struct {
	BatchID string
	Posts   []GeneratedPost
}

func GeneratePosts(ctx context.Context, app core.App, generate content.GenerateFunc, businessID string) (*GenerateBatchResult, error) {
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return nil, wrapNotFound(err, "negócio não encontrado")
	}

	profile, err := operator.BusinessToProfile(business)
	if err != nil {
		return nil, fmt.Errorf("business to profile: %w", err)
	}

	roles := content.PickRoles(1, nil)

	previousHooks, err := operator.LoadPreviousHooks(app, businessID)
	if err != nil {
		return nil, fmt.Errorf("load previous hooks: %w", err)
	}

	posts, err := generate(ctx, profile, roles, previousHooks)
	if err != nil {
		return nil, err
	}

	hooks := content.ExtractHooks(posts)

	batchID := uuid.New().String()
	collection, err := app.FindCollectionByNameOrId(domain.CollPosts)
	if err != nil {
		return nil, fmt.Errorf("find posts collection: %w", err)
	}

	result := &GenerateBatchResult{
		BatchID: batchID,
		Posts:   make([]GeneratedPost, 0, len(posts)),
	}

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

		if err := app.Save(record); err != nil {
			return nil, fmt.Errorf("save post %d: %w", i, err)
		}

		result.Posts = append(result.Posts, GeneratedPost{
			ID:             record.Id,
			Caption:        post.Caption,
			Hashtags:        post.Hashtags,
			ProductionNote: post.ProductionNote,
			Role:           roleName,
			Hook:           hook,
		})
	}

	return result, nil
}

func GenerateFromMessage(ctx context.Context, app core.App, genFn content.GenerateFromMessageFunc, businessID, message, messageID string) (*GeneratedPost, error) {
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return nil, wrapNotFound(err, "negócio não encontrado")
	}

	profile, err := operator.BusinessToProfile(business)
	if err != nil {
		return nil, fmt.Errorf("business to profile: %w", err)
	}

	previousHooks, err := operator.LoadPreviousHooks(app, businessID)
	if err != nil {
		return nil, fmt.Errorf("load previous hooks: %w", err)
	}

	post, err := genFn(ctx, profile, message, previousHooks)
	if err != nil {
		return nil, err
	}

	hook := content.ExtractHooks([]content.Post{post})
	hookStr := ""
	if len(hook) > 0 {
		hookStr = hook[0]
	}

	collection, err := app.FindCollectionByNameOrId(domain.CollPosts)
	if err != nil {
		return nil, fmt.Errorf("find posts collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("business", businessID)
	record.Set("caption", post.Caption)
	record.Set("hashtags", post.Hashtags)
	record.Set("production_note", post.ProductionNote)
	record.Set("hook", hookStr)
	record.Set("source", domain.PostSourceOperator)
	record.Set("edited", false)
	if messageID != "" {
		record.Set("message", messageID)
	}

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("save operator post: %w", err)
	}

	return &GeneratedPost{
		ID:             record.Id,
		Caption:        post.Caption,
		Hashtags:        post.Hashtags,
		ProductionNote: post.ProductionNote,
		Hook:           hookStr,
	}, nil
}

func GenerateIdeas(ctx context.Context, app core.App, generate content.GenerateFunc, businessID string) ([]content.Post, error) {
	business, err := app.FindRecordById(domain.CollBusinesses, businessID)
	if err != nil {
		return nil, wrapNotFound(err, "negócio não encontrado")
	}

	profile, err := operator.BusinessToProfile(business)
	if err != nil {
		return nil, fmt.Errorf("business to profile: %w", err)
	}

	previousHooks, err := operator.LoadPreviousHooks(app, businessID)
	if err != nil {
		return nil, fmt.Errorf("load previous hooks: %w", err)
	}

	return generate(ctx, profile, content.PickRoles(1, nil), previousHooks)
}

type SaveProactiveParams struct {
	BusinessID     string
	Caption        string
	Hashtags       []string
	ProductionNote string
}

func SaveProactivePost(app core.App, params SaveProactiveParams) (string, error) {
	collection, err := app.FindCollectionByNameOrId(domain.CollPosts)
	if err != nil {
		return "", fmt.Errorf("find posts collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("business", params.BusinessID)
	record.Set("caption", params.Caption)
	record.Set("hashtags", params.Hashtags)
	record.Set("production_note", params.ProductionNote)
	record.Set("source", domain.PostSourceProactive)
	record.Set("edited", false)

	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("save proactive post: %w", err)
	}

	return record.Id, nil
}
