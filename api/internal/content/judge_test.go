//go:build integration

package content

import (
	"context"
	"testing"
)

var judgeProfile = BusinessProfile{
	BusinessName:   "Confeitaria da Tia Marta",
	BusinessType:   "confeitaria",
	City:           "Belo Horizonte",
	Services: []Service{
		{Name: "Bolo personalizado", PriceBRL: 180},
		{Name: "Docinhos para festa", PriceBRL: 120},
		{Name: "Cupcake", PriceBRL: 25},
	},
	TargetAudience: "famílias e mulheres 30-55",
	BrandVibe:      "acolhedor",
	Quirks:         []string{"entrega no dia", "personaliza nomes e desenhos", "opção vegana"},
}

const knownGoodContent = `POST 1:
Hoje de manhã a cozinha já tava cheirando chocolate às 6h. A Marta tava quieta, concentrada, desenhando uma Magali no bolo de uma menina que vai fazer 7 anos amanhã.

Esse é o tipo de encomenda que não aparece no cardápio. A cliente mandou foto do caderno de desenho da filha e pediu pra gente reproduzir. Ficou igualzinho.

Bolo personalizado a partir de R$180, mas o desenho feito com a referência da própria criança? Isso não tem preço 😊

Pra encomendar chama no zap. A gente entrega no dia se precisar.

📸 Foto do bolo ao lado do desenho original da criança, luz natural da janela da cozinha

#ConfeitariaDaTiaMarta #BoloBH #BoloPersonalizado #Funcionarios #ConfeitariaBH

---

POST 2:
Pergunta honesta pra quem já tentou: alguém conseguiu fazer brigadeiro vegano em casa sem gosto de leite de coco?

A gente testou 11 receitas até chegar numa que ninguém percebe a diferença. O truque é o cacau que a Marta compra direto de Ilhéus, não é o mesmo do mercado.

Bandeja com 50 docinhos por R$120. Metade vegano, metade tradicional, mistura sem drama.

Próxima festa, testa. Chama no zap e conta quantos convidados tem.

🎥 Vídeo curto: close na mão da Marta enrolando o brigadeiro, mostra a textura cremosa. Fundo desfocado da bancada.

#BrigadeiroVegano #DocinhoBH #ConfeitariaDaTiaMarta #FestaBH #DocinhosDeFesta`

const knownBadContent = `We are pleased to announce our premium confectionery services. Our establishment offers a wide range of high-quality baked goods at competitive prices. We pride ourselves on excellent customer service and timely delivery. Contact our sales team for corporate catering packages and bulk order discounts. We look forward to serving your confectionery needs.`

func TestJudgeKnownGoodPassesMost(t *testing.T) {
	results, err := RunAllJudges(context.Background(), judgeProfile, knownGoodContent)
	if err != nil {
		t.Fatal(err)
	}
	passCount := 0
	for _, r := range results {
		t.Logf("  %s: verdict=%v reasoning=%q", r.Name, r.Verdict, r.Reasoning)
		if r.Verdict {
			passCount++
		}
	}
	if passCount < 4 {
		t.Errorf("expected known-good content to pass at least 4/5 judges, got %d passes", passCount)
	}
}

func TestJudgeKnownBadFailsMost(t *testing.T) {
	results, err := RunAllJudges(context.Background(), judgeProfile, knownBadContent)
	if err != nil {
		t.Fatal(err)
	}
	failCount := 0
	for _, r := range results {
		t.Logf("  %s: verdict=%v reasoning=%q", r.Name, r.Verdict, r.Reasoning)
		if !r.Verdict {
			failCount++
		}
	}
	if failCount < 3 {
		t.Errorf("expected known-bad content to fail at least 3/5 judges, got %d failures", failCount)
	}
}

// knownNoCTAContent has no CTA, no hashtags, and a natural announcement style.
// JudgeAcionavel should pass (absence of CTA/hashtags is fine).
// JudgeEngajamento should pass (genuine voice, specific details).
const knownNoCTAContent = `POST 1:
Ontem eu fiz 14 bolos. Quatorze. Minha mão tá doendo de bater massa e eu nem liguei a batedeira porque o bolo de pote fica melhor no braço.

A Confeitaria da Tia Marta começou assim: fazendo bolo pra vizinha. Agora faço pra Belo Horizonte inteira e continuo batendo no braço. Dá trabalho, mas o cliente percebe.

Bolo de pote de ninho com Nutella a R$12. Encomenda mínima de 10 unidades, entrega no dia em BH.

📸 Foto da bancada cheia de potes prontos, com a Marta ao fundo de avental sujo de chocolate, sorrindo cansada. Luz natural da janela.`

func TestJudgeNoCTAContent(t *testing.T) {
	results, err := RunAllJudges(context.Background(), judgeProfile, knownNoCTAContent)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		t.Logf("  %s: verdict=%v reasoning=%q", r.Name, r.Verdict, r.Reasoning)
		if r.Name == "acionavel" && !r.Verdict {
			t.Errorf("JudgeAcionavel should pass content with no CTA/hashtags, got verdict=false: %s", r.Reasoning)
		}
		if r.Name == "engajamento" && !r.Verdict {
			t.Errorf("JudgeEngajamento should pass natural announcement-style content, got verdict=false: %s", r.Reasoning)
		}
	}
}
