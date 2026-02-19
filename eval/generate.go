package eval

import (
	"context"
	"fmt"

	baml "github.com/denisraison/rekan/eval/baml_client/baml_client"
)

func Generate(ctx context.Context, profile BusinessProfile) (string, error) {
	content, err := baml.GenerateContent(ctx, toBamlProfile(profile))
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}
	return content, nil
}
