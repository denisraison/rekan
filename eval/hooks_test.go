package eval

import (
	"testing"
)

func TestExtractHooks(t *testing.T) {
	content := `O barulho da máquina ligando logo cedo é o que me move. Tem gente que acha que vida de barbeiro é só cortar cabelo.

#barbearia #floripa

---

O cara chegou com o cabelo parecendo um capacete! Foram 50 minutos de concentração.

#antesedepois

---

Vou falar uma coisa que muita gente no meio não gosta de ouvir: não adianta ter salão bonito se o barbeiro não sabe o básico.

#opinião`

	hooks := ExtractHooks(content)
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
	hooks := ExtractHooks("")
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks for empty content, got %d", len(hooks))
	}
}

func TestHookAccumulation(t *testing.T) {
	batch1 := `Hoje de manhã queimei a terceira fornada. Mas o resultado ficou perfeito.

#confeitaria

---

R$12 de margarina e o bolo não cresceu. Acontece com todo mundo.

#realidade`

	batch2 := `Minha cliente chegou chorando na cadeira. Saiu sorrindo.

#transformação

---

O segredo do meu brigadeiro é o cacau de Ilhéus! Ninguém acredita que não é importado.

#bastidor`

	// Simulate chain: extract from batch 1, accumulate, extract from batch 2.
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
