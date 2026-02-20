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
