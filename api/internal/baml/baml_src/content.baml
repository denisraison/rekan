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
    - Legenda (200 a 400 caracteres). Pode passar um pouco se a história pedir, mas nunca passe de 500.
    - Hashtags do nicho (0 a 3, só se fizer sentido para o post). Não force hashtags genéricas.
    - CTA é opcional. Se incluir, prefira CTAs que mantêm a pessoa no Instagram: "salva esse post", "manda pra uma amiga que precisa", "comenta aqui". Evite CTAs que tiram a pessoa do Instagram: "link na bio", "chama no zap". O algoritmo penaliza saídas da plataforma.
    - Nota de produção: o que fotografar, enquadramento básico, uma dica prática. 2-3 frases.

    Papel do post:
    {% for r in roles %}  {{ r.name }}: {{ r.description }}
    {% endfor %}

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva uma cena que dá pra montar e fotografar: preparando um produto, arrumando o espaço, um resultado de serviço. A legenda acompanha a foto.
    - NUNCA use travessão (—). Se precisar separar uma ideia, use vírgula ou ponto.
    - Separe ideias em parágrafos curtos com linha em branco entre eles. Evite blocos de texto corrido.
    - Use palavras-chave do nicho naturalmente na legenda, especialmente nas duas primeiras frases. Ex: "bolo de aniversario personalizado" em vez de só "bolo lindo". O Instagram funciona como buscador e lê o texto da legenda pra distribuir o conteúdo.
    - Emojis só quando você usaria de verdade no WhatsApp
    - Mencione o nome do negócio
    - Mencione a cidade ou bairro
    - Abra com um micro-momento concreto: uma cena específica, um número real, um detalhe sensorial. Nada de declarações genéricas. Exemplos: "Hoje de manhã queimei a terceira fornada", "R$12 de margarina e o bolo não cresceu", "Minha cliente chegou chorando na cadeira".
    - NUNCA termine com pergunta genérica de engajamento ("qual seu favorito?", "comenta aqui", "marca um amigo", "quem concorda?"). Se quiser provocar interação, faça isso dentro da história: conte algo que gera opinião dividida, revele um bastidor polêmico, ou deixe uma dúvida real no ar.

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
    - Legenda (200 a 400 caracteres). Pode passar um pouco se a história pedir, mas nunca passe de 500.
    - Hashtags do nicho (0 a 3, só se fizer sentido para o post). Não force hashtags genéricas.
    - CTA é opcional. Se incluir, prefira CTAs que mantêm a pessoa no Instagram: "salva esse post", "manda pra uma amiga que precisa", "comenta aqui". Evite CTAs que tiram a pessoa do Instagram: "link na bio", "chama no zap". O algoritmo penaliza saídas da plataforma.
    - Nota de produção: o que fotografar, enquadramento básico, uma dica prática. 2-3 frases.

    REGRA PRINCIPAL, use os detalhes concretos da mensagem do cliente:
    - O post deve ser centrado no que o cliente descreveu: promoção, novidade, bastidor, evento, resultado de serviço
    - Extraia nomes, números, datas, detalhes específicos da mensagem e use na legenda
    - Se a mensagem mencionar preço, inclua o preço. Se mencionar data, inclua a data.

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva uma cena que dá pra montar e fotografar, ligada ao que o cliente descreveu
    - NUNCA use travessão (—). Se precisar separar uma ideia, use vírgula ou ponto.
    - Separe ideias em parágrafos curtos com linha em branco entre eles. Evite blocos de texto corrido.
    - Use palavras-chave do nicho naturalmente na legenda, especialmente nas duas primeiras frases. Ex: "bolo de aniversario personalizado" em vez de só "bolo lindo". O Instagram funciona como buscador e lê o texto da legenda pra distribuir o conteúdo.
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
