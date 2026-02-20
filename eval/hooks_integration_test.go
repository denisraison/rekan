//go:build integration

package eval

import (
	"context"
	"testing"
)

func TestChainGenerationProducesDifferentHooks(t *testing.T) {
	profile := BusinessProfile{
		BusinessName:   "Confeitaria da Tia Marta",
		BusinessType:   "confeitaria",
		City:           "Belo Horizonte",
		Neighbourhood:  "Funcionários",
		Services: []Service{
			{Name: "Bolo personalizado", PriceBRL: 180},
			{Name: "Docinhos para festa", PriceBRL: 120},
			{Name: "Cupcake", PriceBRL: 25},
		},
		TargetAudience: "famílias e mulheres 30-55",
		BrandVibe:      "acolhedor",
		Quirks:         []string{"entrega no dia", "personaliza nomes e desenhos", "opção vegana"},
	}

	ctx := context.Background()
	roles := PickRoles(3, nil)

	// Batch 1: no previous hooks.
	content1, err := Generate(ctx, profile, roles, nil)
	if err != nil {
		t.Fatalf("batch 1 generate: %v", err)
	}
	hooks1 := ExtractHooks(content1)
	if len(hooks1) == 0 {
		t.Fatal("batch 1 produced no hooks")
	}
	t.Logf("batch 1 hooks (%d):", len(hooks1))
	for _, h := range hooks1 {
		t.Logf("  - %s", h)
	}

	// Batch 2: pass batch 1 hooks as exclusion context.
	roles2 := PickRoles(3, nil)
	content2, err := Generate(ctx, profile, roles2, hooks1)
	if err != nil {
		t.Fatalf("batch 2 generate: %v", err)
	}
	hooks2 := ExtractHooks(content2)
	if len(hooks2) == 0 {
		t.Fatal("batch 2 produced no hooks")
	}
	t.Logf("batch 2 hooks (%d):", len(hooks2))
	for _, h := range hooks2 {
		t.Logf("  - %s", h)
	}

	// Verify no exact duplicates across batches.
	set := make(map[string]bool, len(hooks1))
	for _, h := range hooks1 {
		set[h] = true
	}
	for _, h := range hooks2 {
		if set[h] {
			t.Errorf("duplicate hook across batches: %q", h)
		}
	}

	// Both batches should pass heuristics.
	for i, content := range []string{content1, content2} {
		checks := RunChecks(content, profile)
		passed := 0
		for _, c := range checks {
			if c.Pass {
				passed++
			}
		}
		if passed < 5 {
			t.Errorf("batch %d: only %d/%d checks passed", i+1, passed, len(checks))
			for _, c := range checks {
				if !c.Pass {
					t.Logf("  FAIL %s: %s", c.Name, c.Reason)
				}
			}
		}
	}
}
