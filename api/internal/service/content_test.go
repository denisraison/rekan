package service_test

import (
	"context"
	"testing"

	"github.com/denisraison/rekan/api/internal/domain"
	"github.com/denisraison/rekan/api/internal/service"
	"github.com/denisraison/rekan/eval"
	_ "github.com/denisraison/rekan/api/migrations"
)

func stubGenerate(_ context.Context, _ eval.BusinessProfile, _ []eval.Role, _ []string) ([]eval.Post, error) {
	return []eval.Post{
		{
			Caption:        "Legenda de teste",
			Hashtags:       []string{"#teste", "#rekan"},
			ProductionNote: "Nota de teste",
		},
	}, nil
}

func stubGenerateFromMessage(_ context.Context, _ eval.BusinessProfile, _ string, _ []string) (eval.Post, error) {
	return eval.Post{
		Caption:        "Legenda do operador",
		Hashtags:       []string{"#operador"},
		ProductionNote: "Nota operador",
	}, nil
}

func TestGeneratePosts(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	result, err := service.GeneratePosts(context.Background(), app, stubGenerate, bizID)
	if err != nil {
		t.Fatalf("GeneratePosts: %v", err)
	}

	if result.BatchID == "" {
		t.Error("batch_id should not be empty")
	}
	if len(result.Posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result.Posts))
	}

	// Verify persisted
	posts, err := app.FindAllRecords(domain.CollPosts)
	if err != nil {
		t.Fatalf("find posts: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post in DB, got %d", len(posts))
	}
	if got := posts[0].GetString("batch_id"); got != result.BatchID {
		t.Errorf("batch_id: got %q, want %q", got, result.BatchID)
	}
	if got := posts[0].GetString("caption"); got != "Legenda de teste" {
		t.Errorf("caption: got %q, want %q", got, "Legenda de teste")
	}
}

func TestGenerateFromMessage(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	result, err := service.GenerateFromMessage(context.Background(), app, stubGenerateFromMessage, bizID, "Fiz um bolo hoje", "")
	if err != nil {
		t.Fatalf("GenerateFromMessage: %v", err)
	}

	if result.Caption != "Legenda do operador" {
		t.Errorf("caption: got %q, want %q", result.Caption, "Legenda do operador")
	}

	// Verify persisted with source=operator
	posts, err := app.FindAllRecords(domain.CollPosts)
	if err != nil {
		t.Fatalf("find posts: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post in DB, got %d", len(posts))
	}
	if got := posts[0].GetString("source"); got != "operator" {
		t.Errorf("source: got %q, want %q", got, "operator")
	}
}

func TestGenerateIdeas(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	posts, err := service.GenerateIdeas(context.Background(), app, stubGenerate, bizID)
	if err != nil {
		t.Fatalf("GenerateIdeas: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 idea, got %d", len(posts))
	}
	if posts[0].Caption != "Legenda de teste" {
		t.Errorf("caption: got %q, want %q", posts[0].Caption, "Legenda de teste")
	}

	// GenerateIdeas should NOT persist to DB
	dbPosts, _ := app.FindAllRecords(domain.CollPosts)
	if len(dbPosts) != 0 {
		t.Errorf("expected 0 posts in DB (ideas are not saved), got %d", len(dbPosts))
	}
}

func TestSaveProactivePost(t *testing.T) {
	app, _, bizID := newTestApp(t)
	defer app.Cleanup()

	id, err := service.SaveProactivePost(app, service.SaveProactiveParams{
		BusinessID:     bizID,
		Caption:        "Post proativo",
		Hashtags:       []string{"#proativo"},
		ProductionNote: "Nota proativa",
	})
	if err != nil {
		t.Fatalf("SaveProactivePost: %v", err)
	}
	if id == "" {
		t.Error("returned ID should not be empty")
	}

	// Verify persisted
	record, err := app.FindRecordById(domain.CollPosts, id)
	if err != nil {
		t.Fatalf("find post: %v", err)
	}
	if got := record.GetString("source"); got != "proactive" {
		t.Errorf("source: got %q, want %q", got, "proactive")
	}
	if got := record.GetString("caption"); got != "Post proativo" {
		t.Errorf("caption: got %q, want %q", got, "Post proativo")
	}
}
