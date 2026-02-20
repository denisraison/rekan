package eval

import (
	"context"
	"fmt"

	baml "github.com/denisraison/rekan/eval/baml_client/baml_client"
)

func Generate(ctx context.Context, profile BusinessProfile, roles []Role, previousHooks []string) (string, error) {
	content, err := baml.GenerateContent(ctx, toBamlProfile(profile), toBamlRoles(roles), previousHooks)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}
	return content, nil
}
