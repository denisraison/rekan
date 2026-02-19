//go:build integration

package eval

import (
	"context"
	"testing"
)

var judgeProfile = BusinessProfile{
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

const knownGoodContent = `POST 1:
Gente, quem aí já provou o bolo personalizado da Confeitaria da Tia Marta? 🎂

Aqui no bairro Funcionários, em BH, a gente faz cada bolo que é uma obra de arte! Personaliza nome, desenho, tema... tudo do jeitinho que você quer pra sua festa ficar perfeita.

E olha, tem opção vegana também, viu? Ninguém fica de fora! 🌱

Nossos docinhos pra festa (a partir de R$120) são aquele sucesso que todo mundo elogia. E o melhor: entrega no dia!

Bora encomendar? Chama no WhatsApp que a Tia Marta cuida de tudo 💕

🎥 Sugestão: filme um vídeo curto mostrando o bolo pronto com o nome personalizado, de perto

#ConfeitariaBH #BoloBH #DocinhosBH #FestaBH #ConfeitariaDaTiaMarta

---

POST 2:
3 erros que acabam com seu bolo de aniversário (e como evitar) 👇

1️⃣ Encomendar em cima da hora — aqui na Confeitaria da Tia Marta a gente pede pelo menos 48h, pra caprichar nos detalhes
2️⃣ Não combinar o tema antes — a gente personaliza tudo, mas precisa saber o que você sonha!
3️⃣ Esquecer dos convidados com restrição — temos opção vegana que é tão gostosa quanto a tradicional

Salva esse post pra não esquecer na hora de encomendar! 📌

Funcionários, BH — chama no zap e garanta o seu 🎂

📸 Sugestão: foto de antes e depois mostrando a massa crua e o bolo decorado finalizado

#ConfeitariaDaTiaMarta #DicasDeConfeitaria #BoloBH #ConfeitariaBH #BoloPersonalizado`

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
