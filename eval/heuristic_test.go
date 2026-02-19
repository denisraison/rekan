package eval

import (
	"strings"
	"testing"
)

var testProfile = BusinessProfile{
	BusinessName:   "Studio Nails da Jéssica",
	BusinessType:   "nail designer",
	City:           "Manaus",
	Neighbourhood:  "Adrianópolis",
	Services: []Service{
		{Name: "Alongamento em gel", PriceBRL: 120},
		{Name: "Esmaltação em gel", PriceBRL: 60},
	},
	TargetAudience: "mulheres 20-35",
	BrandVibe:      "trendy",
	Quirks:         []string{"atende com hora marcada"},
}

const passingSample = `Gente, olha que perfeição essas unhas da Studio Nails da Jéssica! 💅

Aqui em Manaus a gente transforma suas mãos com muito carinho e estilo. Bora agendar seu horário pra ficar com as unhas dos sonhos?

📸 Registre suas unhas novas e marque a gente!

Chama no WhatsApp pra agendar ✨

#NailsManaus #UnhasDeGel #StudioNails`

const failingSample = `We are pleased to announce our new service offerings. Our company provides excellent quality at competitive prices. Contact our sales department for more information about our premium packages. We look forward to serving you.`

func TestPassingSamplePassesAll(t *testing.T) {
	results := RunChecks(passingSample, testProfile)
	for _, r := range results {
		if !r.Pass {
			t.Errorf("check %q should pass, got fail: %s", r.Name, r.Reason)
		}
	}
}

func TestFailingSampleFailsMost(t *testing.T) {
	results := RunChecks(failingSample, testProfile)
	failCount := 0
	for _, r := range results {
		if !r.Pass {
			failCount++
		}
	}
	if failCount < 5 {
		t.Errorf("expected at least 5 failures, got %d", failCount)
		for _, r := range results {
			t.Logf("  %s: pass=%v reason=%q", r.Name, r.Pass, r.Reason)
		}
	}
}

func TestCheckBusinessName(t *testing.T) {
	t.Run("case insensitive", func(t *testing.T) {
		r := checkBusinessName("venha pro studio nails da jéssica!", testProfile)
		if !r.Pass {
			t.Error("should match case-insensitively")
		}
	})
	t.Run("accent insensitive", func(t *testing.T) {
		r := checkBusinessName("venha pro Studio Nails da Jessica!", testProfile)
		if !r.Pass {
			t.Error("should match without accents")
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := checkBusinessName("venha pro nosso salão!", testProfile)
		if r.Pass {
			t.Error("should fail when business name absent")
		}
	})
}

func TestCheckLocation(t *testing.T) {
	t.Run("city match", func(t *testing.T) {
		r := checkLocation("Aqui em Manaus", testProfile)
		if !r.Pass {
			t.Error("should match city")
		}
	})
	t.Run("neighbourhood match", func(t *testing.T) {
		r := checkLocation("Lá em Adrianópolis", testProfile)
		if !r.Pass {
			t.Error("should match neighbourhood")
		}
	})
	t.Run("neither", func(t *testing.T) {
		r := checkLocation("Aqui no bairro", testProfile)
		if r.Pass {
			t.Error("should fail when neither city nor neighbourhood present")
		}
	})
}

func TestCheckHashtags(t *testing.T) {
	t.Run("exactly 3", func(t *testing.T) {
		r := checkHashtags("texto #one #two #three")
		if !r.Pass {
			t.Error("exactly 3 hashtags should pass")
		}
	})
	t.Run("two fails", func(t *testing.T) {
		r := checkHashtags("texto #one #two")
		if r.Pass {
			t.Error("2 hashtags should fail")
		}
	})
	t.Run("accented", func(t *testing.T) {
		r := checkHashtags("#CafézinhoBH #PãoDeQueijo #Açaí")
		if !r.Pass {
			t.Error("accented hashtags should be counted")
		}
	})
	t.Run("none", func(t *testing.T) {
		r := checkHashtags("texto sem hashtag")
		if r.Pass {
			t.Error("no hashtags should fail")
		}
	})
}

func TestCheckCTA(t *testing.T) {
	t.Run("whatsapp", func(t *testing.T) {
		r := checkCTA("Chama no WhatsApp!")
		if !r.Pass {
			t.Error("should match WhatsApp CTA")
		}
	})
	t.Run("agende", func(t *testing.T) {
		r := checkCTA("Agende seu horário")
		if !r.Pass {
			t.Error("should match agende")
		}
	})
	t.Run("none", func(t *testing.T) {
		r := checkCTA("Nossas unhas são lindas.")
		if r.Pass {
			t.Error("should fail without CTA")
		}
	})
}

func TestCheckBrazilianPortuguese(t *testing.T) {
	t.Run("has marker", func(t *testing.T) {
		r := checkBrazilianPortuguese("Bora gente, vamos lá!")
		if !r.Pass {
			t.Error("should pass with pt-BR markers")
		}
	})
	t.Run("no markers", func(t *testing.T) {
		r := checkBrazilianPortuguese("We offer excellent services.")
		if r.Pass {
			t.Error("should fail without pt-BR markers")
		}
	})
	t.Run("portugal marker", func(t *testing.T) {
		r := checkBrazilianPortuguese("Bora pegar o autocarro gente!")
		if r.Pass {
			t.Error("should fail with Portugal Portuguese marker")
		}
	})
}

func TestCheckCaptionLength(t *testing.T) {
	t.Run("exactly 2200", func(t *testing.T) {
		content := strings.Repeat("a", 2200)
		r := checkCaptionLength(content)
		if !r.Pass {
			t.Error("exactly 2200 chars should pass")
		}
	})
	t.Run("2201 fails", func(t *testing.T) {
		content := strings.Repeat("a", 2201)
		r := checkCaptionLength(content)
		if r.Pass {
			t.Error("2201 chars should fail")
		}
	})
}

func TestCheckProductionNote(t *testing.T) {
	t.Run("foto", func(t *testing.T) {
		r := checkProductionNote("Tire uma foto incrível!")
		if !r.Pass {
			t.Error("should match 'foto'")
		}
	})
	t.Run("stories", func(t *testing.T) {
		r := checkProductionNote("Poste nos stories")
		if !r.Pass {
			t.Error("should match 'stories'")
		}
	})
	t.Run("none", func(t *testing.T) {
		r := checkProductionNote("Texto sem sugestão de mídia")
		if r.Pass {
			t.Error("should fail without production keywords")
		}
	})
}
