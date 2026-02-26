package terms

import (
	"fmt"
	"strings"

	"github.com/denisraison/rekan/api/internal/pricing"
)

// Clause represents a single numbered clause of the terms of service.
type Clause struct {
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}

var pricingPlans = [3]string{
	"Básico (8 posts/mês): Mensal R$ 69,90 | Trimestral R$ 179,70 (R$ 59,90/mês)",
	"Parceiro (12 posts/mês): Mensal R$ 108,90* | Trimestral R$ 299,70 (R$ 99,90/mês)",
	"Profissional (16 posts/mês): Mensal R$ 249,90 | Trimestral R$ 599,70 (R$ 199,90/mês)",
}

const pricingFootnote = "* Preço promocional de lançamento (preço regular: R$ 149,90/mês). O preço de lançamento pode ser encerrado a qualquer momento para novas contratações. Clientes que contrataram no preço de lançamento mantêm o valor enquanto a assinatura estiver ativa."

// Clauses returns the full terms as structured data.
// Clause 3 shows the generic pricing table (no personalization).
func Clauses() []Clause {
	return []Clause{
		{
			Title:      "1. Identificação do Fornecedor.",
			Paragraphs: []string{"O Rekan é um serviço prestado por 65193835 Elenice de Souza, inscrita no CNPJ 65.193.835/0001-50, com sede em São Caetano do Sul/SP. Contato: WhatsApp (11) 94069-9184 ou e-mail chama@rekan.com.br. A encarregada de proteção de dados (DPO) é Elenice de Souza, acessível pelos mesmos canais."},
		},
		{
			Title:      "2. Descrição do Serviço.",
			Paragraphs: []string{"O Rekan é um serviço de criação de conteúdo para Instagram destinado a micro-empreendedores brasileiros. O conteúdo (legendas, hashtags, direção de foto, roteiros de reels e textos para stories) é gerado com auxílio de inteligência artificial e revisado pela equipe do Rekan antes da entrega. O serviço não garante aumento de seguidores, engajamento ou vendas. O resultado depende de fatores como frequência de postagem, qualidade das fotos enviadas e características do mercado local."},
		},
		{
			Title: "3. Planos e Preços.",
			Paragraphs: []string{
				pricingPlans[0],
				pricingPlans[1],
				pricingPlans[2],
				pricingFootnote,
			},
		},
		{
			Title:      "4. Pagamento e Pix Automático.",
			Paragraphs: []string{"O pagamento é realizado via Pix Automático. Ao escanear o QR Code de pagamento no momento da contratação, você autoriza a cobrança do valor do plano escolhido e a realização de cobranças recorrentes via débito automático na sua conta bancária a cada período de renovação (mensal ou trimestral, conforme o plano contratado). As cobranças seguintes serão debitadas automaticamente na data de vencimento, sem necessidade de nova autorização. Você pode cancelar a autorização de débito automático a qualquer momento pelo aplicativo do seu banco ou entrando em contato conosco. Em caso de alteração de preço, você será notificado com no mínimo 30 dias de antecedência, podendo cancelar antes da renovação."},
		},
		{
			Title:      "5. Garantia de 30 Dias.",
			Paragraphs: []string{"Se nos primeiros 30 dias após a contratação você não estiver satisfeito com o serviço, devolvemos o valor integral pago via Pix em até 3 dias úteis, sem perguntas. Para solicitar o reembolso, basta entrar em contato pelo WhatsApp (11) 94069-9184 dentro desse prazo."},
		},
		{
			Title:      "6. Direito de Arrependimento.",
			Paragraphs: []string{"Conforme o Art. 49 do Código de Defesa do Consumidor, você pode desistir da contratação em até 7 (sete) dias corridos após a aceitação destes Termos, com reembolso integral de qualquer valor pago. Para exercer este direito, entre em contato pelo WhatsApp (11) 94069-9184 ou e-mail chama@rekan.com.br. O reembolso será processado em até 3 dias úteis via Pix. Este direito é independente da garantia de 30 dias descrita acima."},
		},
		{
			Title:      "7. Cancelamento.",
			Paragraphs: []string{"Você pode cancelar a assinatura a qualquer momento, sem multa, entrando em contato pelo WhatsApp (11) 94069-9184. Para planos mensais, o cancelamento é efetivado imediatamente e não haverá novas cobranças. Para planos trimestrais, o serviço continua disponível até o final do período já pago, sem reembolso proporcional do período restante, salvo nos casos previstos nos itens 5 e 6 acima. A autorização de débito automático (Pix Automático) será cancelada junto com a assinatura."},
		},
		{
			Title:      "8. Vigência.",
			Paragraphs: []string{"O contrato vigora por prazo indeterminado, com renovação automática a cada período de cobrança (mensal ou trimestral, conforme o plano contratado), até que seja cancelado conforme o item 7."},
		},
		{
			Title:      "9. Obrigações do Contratante.",
			Paragraphs: []string{"Para que o Rekan possa prestar o serviço, o contratante se compromete a: (a) fornecer informações e materiais (fotos, textos, áudios) necessários para a criação do conteúdo dentro do período de cobrança; (b) utilizar o conteúdo gerado exclusivamente para o Instagram do seu negócio; (c) não revender ou redistribuir o conteúdo para terceiros."},
		},
		{
			Title:      "10. Entrega do Serviço.",
			Paragraphs: []string{"O Rekan se compromete a entregar a quantidade de posts prevista no plano contratado dentro de cada período de cobrança. Caso o Rekan não entregue a totalidade dos posts por motivo atribuível ao Rekan, os posts restantes serão entregues no período seguinte. O não envio de materiais pelo contratante dentro do período não gera direito a crédito ou compensação."},
		},
		{
			Title:      "11. Limitação de Responsabilidade.",
			Paragraphs: []string{"O Rekan não se responsabiliza por danos indiretos, lucros cessantes ou resultados comerciais decorrentes do uso do conteúdo gerado, nos limites permitidos pela legislação vigente. A responsabilidade total do Rekan é limitada ao valor pago pelo contratante no período de cobrança em que o evento causador do dano ocorreu."},
		},
		{
			Title: "12. Proteção de Dados (LGPD, Lei 13.709/2018).",
			Paragraphs: []string{
				"O controlador dos seus dados pessoais é 65193835 Elenice de Souza, CNPJ 65.193.835/0001-50. A encarregada de proteção de dados é Elenice de Souza, acessível pelo WhatsApp (11) 94069-9184 ou e-mail chama@rekan.com.br.",
				"Dados coletados: nome, nome do negócio, informações sobre o negócio e conteúdo enviado por você (fotos, textos, áudios). A base legal para o tratamento é a execução do contrato de prestação de serviço (Art. 7, inciso V da LGPD).",
				"O CPF/CNPJ informado no momento da contratação é transmitido ao processador de pagamentos Asaas Gestão Financeira S.A. para fins de cobrança e não é armazenado pelo Rekan. Para gerar o conteúdo, as informações e materiais que você envia são processados por provedores de inteligência artificial sediados no exterior. Essa transferência internacional de dados é necessária para a execução do contrato (Art. 33, inciso II da LGPD). Seus dados não são compartilhados com terceiros para fins de marketing.",
				"Seus dados são mantidos enquanto a assinatura estiver ativa e por até 5 anos após o cancelamento para cumprimento de obrigações legais e fiscais.",
				"Você pode, a qualquer momento, exercer os direitos previstos no Art. 18 da LGPD, incluindo confirmação de tratamento, acesso, correção, anonimização, portabilidade, exclusão e revogação de consentimento. Para exercer seus direitos, entre em contato pelo WhatsApp (11) 94069-9184 ou e-mail chama@rekan.com.br.",
			},
		},
		{
			Title:      "13. Uso do Conteúdo.",
			Paragraphs: []string{"Todo conteúdo gerado pelo Rekan para o seu negócio é de sua propriedade e pode ser usado livremente. O Rekan pode utilizar exemplos anonimizados (sem identificar você ou seu negócio) para demonstração do serviço, salvo se você solicitar o contrário."},
		},
		{
			Title:      "14. Alteração dos Termos.",
			Paragraphs: []string{"O Rekan pode alterar estes Termos a qualquer momento. As alterações serão comunicadas pelo WhatsApp com no mínimo 30 dias de antecedência. Caso você não concorde com as alterações, poderá cancelar a assinatura sem multa antes da entrada em vigor dos novos termos. A continuidade do uso do serviço após a entrada em vigor constitui aceitação dos novos termos."},
		},
		{
			Title:      "15. Foro.",
			Paragraphs: []string{"Fica eleito o foro da comarca de São Caetano do Sul/SP para dirimir quaisquer questões decorrentes destes Termos, sem prejuízo do foro do domicílio do consumidor previsto no Art. 101, inciso I do Código de Defesa do Consumidor."},
		},
	}
}

// Snapshot returns a personalized plain-text snapshot for storage at acceptance time.
// Clause 3 is personalized with the client's tier, commitment, and price.
func Snapshot(tier pricing.Tier, commitment pricing.Commitment, price float64) string {
	tierNames := map[pricing.Tier]string{
		pricing.Basico:       "Básico",
		pricing.Parceiro:     "Parceiro",
		pricing.Profissional: "Profissional",
	}
	commitmentNames := map[pricing.Commitment]string{
		pricing.Mensal:     "Mensal",
		pricing.Trimestral: "Trimestral",
	}

	months := pricing.Months[commitment]
	var priceDesc string
	if months == 1 {
		priceDesc = fmt.Sprintf("R$ %.2f/mês", price)
	} else {
		monthly := price / float64(months)
		priceDesc = fmt.Sprintf("R$ %.2f (%dx de R$ %.2f)", price, months, monthly)
	}
	priceDesc = strings.ReplaceAll(priceDesc, ".", ",")

	clauses := Clauses()

	var b strings.Builder
	b.WriteString("TERMOS DE USO DO SERVIÇO REKAN\n\n")

	for i, c := range clauses {
		if i == 2 {
			// Personalized clause 3
			fmt.Fprintf(&b, "3. Plano Contratado. Plano %s (%s): %s.\n\n", tierNames[tier], commitmentNames[commitment], priceDesc)
			b.WriteString("Tabela de preços vigente:\n")
			for _, p := range pricingPlans {
				b.WriteString("- ")
				b.WriteString(p)
				b.WriteByte('\n')
			}
			b.WriteString(pricingFootnote)
		} else {
			b.WriteString(c.Title)
			b.WriteByte(' ')
			b.WriteString(strings.Join(c.Paragraphs, " "))
		}
		if i < len(clauses)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}
