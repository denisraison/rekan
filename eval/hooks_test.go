package eval

import (
	"testing"
)

func TestExtractHooks(t *testing.T) {
	posts := []Post{
		{Caption: "O barulho da máquina ligando logo cedo é o que me move. Tem gente que acha que vida de barbeiro é só cortar cabelo."},
		{Caption: "O cara chegou com o cabelo parecendo um capacete! Foram 50 minutos de concentração."},
		{Caption: "Vou falar uma coisa que muita gente no meio não gosta de ouvir: não adianta ter salão bonito se o barbeiro não sabe o básico."},
	}

	hooks := ExtractHooks(posts)
	if len(hooks) != 3 {
		t.Fatalf("expected 3 hooks, got %d: %v", len(hooks), hooks)
	}

	expected := []string{
		"O barulho da máquina ligando logo cedo é o que me move.",
		"O cara chegou com o cabelo parecendo um capacete!",
		"Vou falar uma coisa que muita gente no meio não gosta de ouvir: não adianta ter salão bonito se o barbeiro não sabe o básico.",
	}
	for i, want := range expected {
		if hooks[i] != want {
			t.Errorf("hook[%d]:\n  got:  %q\n  want: %q", i, hooks[i], want)
		}
	}
}

func TestExtractHooksEmpty(t *testing.T) {
	hooks := ExtractHooks(nil)
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks for empty posts, got %d", len(hooks))
	}
}

func TestHookAccumulation(t *testing.T) {
	batch1 := []Post{
		{Caption: "Hoje de manhã queimei a terceira fornada. Mas o resultado ficou perfeito."},
		{Caption: "R$12 de margarina e o bolo não cresceu. Acontece com todo mundo."},
	}

	batch2 := []Post{
		{Caption: "Minha cliente chegou chorando na cadeira. Saiu sorrindo."},
		{Caption: "O segredo do meu brigadeiro é o cacau de Ilhéus! Ninguém acredita que não é importado."},
	}

	hooks1 := ExtractHooks(batch1)
	if len(hooks1) != 2 {
		t.Fatalf("batch 1: expected 2 hooks, got %d", len(hooks1))
	}

	allHooks := hooks1
	hooks2 := ExtractHooks(batch2)
	if len(hooks2) != 2 {
		t.Fatalf("batch 2: expected 2 hooks, got %d", len(hooks2))
	}
	allHooks = append(allHooks, hooks2...)

	if len(allHooks) != 4 {
		t.Fatalf("expected 4 accumulated hooks, got %d", len(allHooks))
	}

	// Verify no duplicates across batches.
	seen := make(map[string]bool, len(allHooks))
	for _, h := range allHooks {
		if seen[h] {
			t.Errorf("duplicate hook: %q", h)
		}
		seen[h] = true
	}
}
