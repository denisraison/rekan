function GenerateContent(profile: BusinessProfile, roles: ContentRole[], previousHooks: string[]) -> Post[] {
  client GeneratorClient
  prompt #"
    Você é o(a) dono(a) do(a) {{ profile.businessName }}. Você mesmo(a) escreve os posts do Instagram do seu negócio. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Escreva 3 posts pro seu Instagram.

    Seu negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade/Bairro: {{ profile.city }}, {{ profile.neighbourhood }}
    - Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    Cada post precisa ter:
    - Legenda (máximo 700 caracteres)
    - Hashtags do nicho e da região (mínimo 5)
    - Um convite pra ação (chama no zap, link na bio, comenta, etc.)
    - Direção de foto/vídeo na nota de produção: o que montar, de que ângulo, em que momento (ex: Foto do bolo pronto na bancada, luz natural da janela)

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva cenas que dá pra montar e fotografar: preparando um produto, arrumando o espaço, um resultado de serviço. A legenda acompanha a foto.
    - Emojis só quando você usaria de verdade no WhatsApp
    - Os 3 posts devem ter papéis diferentes. Misture estes tipos (em qualquer ordem):
    {% for r in roles %}  {{ r.name }}: {{ r.description }}
    {% endfor %}
    - Varie a estrutura: um post pode ser narrativo, outro direto, outro lista. Não siga o mesmo esqueleto nos 3.
    - Não repita o mesmo convite pra ação nos 3 posts
    - Mencione o nome do negócio em pelo menos 2 posts
    - Mencione a cidade ou bairro em pelo menos 1 post
    - Abra cada post com um micro-momento concreto: uma cena específica, um número real, um detalhe sensorial. Nada de declarações genéricas. Exemplos: "Hoje de manhã queimei a terceira fornada", "R$12 de margarina e o bolo não cresceu", "Minha cliente chegou chorando na cadeira".
    - NUNCA termine um post com pergunta genérica de engajamento ("qual seu favorito?", "comenta aqui", "marca um amigo", "quem concorda?"). Se quiser provocar interação, faça isso dentro da história: conte algo que gera opinião dividida, revele um bastidor polêmico, ou deixe uma dúvida real no ar.

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
    - Cidade/Bairro: {{ profile.city }}, {{ profile.neighbourhood }}
    - Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
    - Público: {{ profile.targetAudience }}
    - Vibe: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    O post precisa ter:
    - Legenda (máximo 700 caracteres)
    - Hashtags do nicho e da região (mínimo 5)
    - Um convite pra ação (chama no zap, link na bio, comenta, etc.)
    - Direção de foto/vídeo na nota de produção: o que montar, de que ângulo, em que momento (ex: Foto do bolo pronto na bancada, luz natural da janela)

    REGRA PRINCIPAL — use os detalhes concretos da mensagem do cliente:
    - O post deve ser centrado no que o cliente descreveu: promoção, novidade, bastidor, evento, resultado de serviço
    - Extraia nomes, números, datas, detalhes específicos da mensagem e use na legenda
    - Se a mensagem mencionar preço, inclua o preço. Se mencionar data, inclua a data.

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva uma cena que dá pra montar e fotografar, ligada ao que o cliente descreveu
    - Emojis só quando você usaria de verdade no WhatsApp
    - Abra com um micro-momento concreto: uma cena específica, um número real, um detalhe sensorial. Nada de declarações genéricas.
    - NUNCA termine com pergunta genérica de engajamento ("qual seu favorito?", "comenta aqui", "marca um amigo", "quem concorda?"). Se quiser provocar interação, faça isso dentro da história.
    - Mencione o nome do negócio
    - Mencione a cidade ou bairro

    {% if previousHooks | length > 0 %}
    IMPORTANTE: Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo, tema ou cena. Crie algo completamente diferente:
    {% for hook in previousHooks %}- {{ hook }}
    {% endfor %}
    {% endif %}

    {{ ctx.output_format }}
  "#
}
