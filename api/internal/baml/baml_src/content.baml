function GenerateContent(profile: BusinessProfile, roles: ContentRole[], previousHooks: string[]) -> Post {
  client GeneratorClient
  prompt #"
    Você é o(a) dono(a) do(a) {{ profile.businessName }}. Você mesmo(a) escreve os posts do Instagram do seu negócio. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Escreva 1 post pro seu Instagram.

    Seu negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    {% if profile.services | length > 0 %}- Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}{% endif %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    O post precisa ter:
    - Legenda CURTA: MÁXIMO 400 caracteres. Conte os caracteres. 2-3 parágrafos curtos, não mais.
    - Hashtags do nicho (0 a 3, só se fizer sentido). Não force.
    - CTA é opcional. A maioria dos posts reais de MEI não tem CTA. Se incluir, varie: "manda pra uma amiga que precisa ouvir isso", "salva pra depois", "comenta se já passou por isso". Evite "link na bio", "chama no zap". NUNCA use "salva esse post" como frase final automática.
    - Nota de produção: o que fotografar com o celular, enquadramento, uma dica. 2-3 frases. Deve ser algo que a pessoa consiga fazer sozinha, agora, sem planejar.

    Papel do post:
    {% for r in roles %}  {{ r.name }}: {{ r.description }}
    {% endfor %}

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão. Frases curtas.
    - NUNCA use travessão (—). Use vírgula ou ponto.
    - Abra com um micro-momento concreto: uma cena, um número real, um detalhe sensorial. Nada de declarações genéricas.
    - Inclua pelo menos um detalhe que não está nos dados do negócio acima: um horário, o clima, um som, uma pessoa, algo que aconteceu. Detalhes inventados devem ter vida própria, não apenas decorar o pitch.
    - Use palavras-chave do nicho em algum lugar da legenda, de forma natural. O Instagram funciona como buscador. Não force na primeira frase se não couber.
    - Mencione o nome do negócio. A cidade/bairro pode aparecer se couber naturalmente, mas não force "aqui em [cidade]" em todo post.
    - Emojis só quando você usaria de verdade no WhatsApp.
    - NUNCA termine com pergunta genérica de engajamento ("qual seu favorito?", "comenta aqui", "marca um amigo?").
    - Evite o formato "pergunta que eu escuto toda semana + resposta". Varie as estruturas: bastidor, marco, opinião, cena do dia, reflexão pessoal.
    - IMPORTANTE: A legenda deve ter no MÁXIMO 400 caracteres. Posts curtos têm mais engajamento. Não desenvolva a história além do necessário. Um parágrafo de abertura + um de contexto é suficiente.

    {% if previousHooks | length > 0 %}
    IMPORTANTE: Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo, tema ou cena. Crie algo completamente diferente:
    {% for hook in previousHooks %}- {{ hook }}
    {% endfor %}
    {% endif %}

    {{ ctx.output_format }}
  "#
}

function GenerateFromMessage(profile: BusinessProfile, clientMessage: string, previousHooks: string[]) -> Post {
  client GeneratorClient
  prompt #"
    Você é o(a) dono(a) do(a) {{ profile.businessName }}. Você mesmo(a) escreve os posts do Instagram do seu negócio. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Um(a) cliente mandou essa mensagem no WhatsApp pedindo um post:
    ---
    {{ clientMessage }}
    ---

    Escreva 1 post pro Instagram baseado no que o(a) cliente pediu.

    Seu negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    {% if profile.services | length > 0 %}- Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}{% endif %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    O post precisa ter:
    - Legenda CURTA: MÁXIMO 400 caracteres. Conte os caracteres. 2-3 parágrafos curtos, não mais.
    - Hashtags do nicho (0 a 3, só se fizer sentido). Não force.
    - CTA é opcional. A maioria dos posts reais de MEI não tem CTA. Se incluir, varie: "manda pra uma amiga que precisa ouvir isso", "salva pra depois", "comenta se já passou por isso". Evite "link na bio", "chama no zap". NUNCA use "salva esse post" como frase final automática.
    - Nota de produção: o que fotografar com o celular, enquadramento, uma dica. 2-3 frases. Deve ser algo que a pessoa consiga fazer sozinha, agora, sem planejar.

    REGRA PRINCIPAL, use os detalhes concretos da mensagem do cliente:
    - Extraia nomes, números, datas, detalhes específicos da mensagem e use na legenda.
    - Se a mensagem mencionar preço, inclua o preço. Se mencionar data, inclua a data.

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão. Frases curtas.
    - NUNCA use travessão (—). Use vírgula ou ponto.
    - Abra com um micro-momento concreto: uma cena, um número real, um detalhe sensorial.
    - Inclua pelo menos um detalhe que não está nos dados do negócio acima: um horário, o clima, um som, algo que aconteceu. Detalhes inventados devem ter vida própria, não apenas decorar o pitch.
    - Use palavras-chave do nicho em algum lugar da legenda, de forma natural. O Instagram funciona como buscador. Não force na primeira frase se não couber.
    - Mencione o nome do negócio. A cidade/bairro pode aparecer se couber naturalmente, mas não force "aqui em [cidade]" em todo post.
    - Emojis só quando você usaria de verdade no WhatsApp.
    - NUNCA termine com pergunta genérica de engajamento.
    - IMPORTANTE: A legenda deve ter no MÁXIMO 400 caracteres. Posts curtos têm mais engajamento. Não desenvolva além do necessário.

    {% if previousHooks | length > 0 %}
    IMPORTANTE: Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo, tema ou cena. Crie algo completamente diferente:
    {% for hook in previousHooks %}- {{ hook }}
    {% endfor %}
    {% endif %}

    {{ ctx.output_format }}
  "#
}
