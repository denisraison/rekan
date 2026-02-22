package eval

import (
	"context"
	"fmt"
	"strings"

	baml "github.com/denisraison/rekan/eval/baml_client/baml_client"
)

type Post struct {
	Caption        string   `json:"caption"`
	Hashtags       []string `json:"hashtags"`
	ProductionNote string   `json:"productionNote"`
}

// GenerateFunc is the signature for content generation functions.
type GenerateFunc func(ctx context.Context, profile BusinessProfile, roles []Role, previousHooks []string) ([]Post, error)

// GenerateFromMessageFunc is the signature for single-post generation from a WhatsApp message.
type GenerateFromMessageFunc func(ctx context.Context, profile BusinessProfile, message string, previousHooks []string) (Post, error)

func Generate(ctx context.Context, profile BusinessProfile, roles []Role, previousHooks []string) ([]Post, error) {
	bamlPosts, err := baml.GenerateContent(ctx, toBamlProfile(profile), toBamlRoles(roles), previousHooks)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}
	posts := make([]Post, len(bamlPosts))
	for i, p := range bamlPosts {
		posts[i] = Post{
			Caption:        p.Caption,
			Hashtags:       p.Hashtags,
			ProductionNote: p.ProductionNote,
		}
	}
	return posts, nil
}

func GenerateRekan(ctx context.Context, profile BusinessProfile, roles []Role, previousHooks []string) ([]Post, error) {
	bamlPosts, err := baml.GenerateRekanContent(ctx, toBamlProfile(profile), toBamlRoles(roles), previousHooks)
	if err != nil {
		return nil, fmt.Errorf("generate rekan content: %w", err)
	}
	posts := make([]Post, len(bamlPosts))
	for i, p := range bamlPosts {
		posts[i] = Post{
			Caption:        p.Caption,
			Hashtags:       p.Hashtags,
			ProductionNote: p.ProductionNote,
		}
	}
	return posts, nil
}

func GenerateFromMessage(ctx context.Context, profile BusinessProfile, message string, previousHooks []string) (Post, error) {
	bamlPost, err := baml.GenerateFromMessage(ctx, toBamlProfile(profile), message, previousHooks)
	if err != nil {
		return Post{}, fmt.Errorf("generate from message: %w", err)
	}
	return Post{
		Caption:        bamlPost.Caption,
		Hashtags:       bamlPost.Hashtags,
		ProductionNote: bamlPost.ProductionNote,
	}, nil
}

// RenderPosts reconstructs a human-readable text format from structured posts.
// Used for judge input and verbose display.
func RenderPosts(posts []Post) string {
	var parts []string
	for _, p := range posts {
		var b strings.Builder
		b.WriteString(p.Caption)
		if len(p.Hashtags) > 0 {
			b.WriteString("\n\n")
			for i, h := range p.Hashtags {
				if i > 0 {
					b.WriteByte(' ')
				}
				if !strings.HasPrefix(h, "#") {
					b.WriteByte('#')
				}
				b.WriteString(h)
			}
		}
		if p.ProductionNote != "" {
			b.WriteString("\n\n[")
			b.WriteString(p.ProductionNote)
			b.WriteByte(']')
		}
		parts = append(parts, b.String())
	}
	return strings.Join(parts, "\n\n---\n\n")
}
