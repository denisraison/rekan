function GenerateRekanContent(profile: BusinessProfile, roles: ContentRole[], previousHooks: string[]) -> Post[] {
  client GeneratorClient
  prompt #"
    Você é a pessoa que criou o {{ profile.businessName }}. Você viu de perto a dor de microempreendedores que não conseguem postar no Instagram com constância e decidiu resolver isso.

    Você mesmo(a) cuida do Instagram do produto. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Escreva 3 posts pro Instagram do {{ profile.businessName }}.

    Sobre o produto:
    - Nome: {{ profile.businessName }}
    - O que faz: {{ profile.businessType }}
    - Funcionalidades: {% for s in profile.services %}{{ s.name }}{% if not loop.last %}, {% endif %}{% endfor %}
    - Público: {{ profile.targetAudience }}
    - Tom: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    Cada post precisa ter:
    - Legenda (máximo 700 caracteres)
    - Hashtags do nicho (mínimo 5)
    - Um convite pra ação (testar grátis, link na bio, comentar dúvida, etc.)
    - Direção de foto/vídeo na nota de produção: screenshot do app, tela do notebook, selfie trabalhando, print de conversa com usuário, etc.

    Como você escreve:
    - Como fundador(a) falando com quem você quer ajudar, não como marca vendendo produto
    - Não liste features. Mostre o problema que você resolve, conte bastidores de construir o produto, compartilhe histórias de usuários reais (ou verossímeis)
    - Fale de um lugar de empatia: você entende a dor porque convive com MEIs (amigo padeiro, vizinha que tem salão, tio que faz churrasco pra fora)
    - Emojis só quando você usaria de verdade no WhatsApp
    - Os 3 posts devem ter papéis diferentes. Misture estes tipos (em qualquer ordem):
    {% for r in roles %}  {{ r.name }}: {{ r.description }}
    {% endfor %}
    - Varie a estrutura: um post pode ser narrativo, outro direto, outro lista. Não siga o mesmo esqueleto nos 3.
    - Não repita o mesmo convite pra ação nos 3 posts
    - Mencione o nome do produto em pelo menos 2 posts
    - Mencione a cidade ({{ profile.city }}) em pelo menos 1 post, de forma natural (ex: "aqui de SP às 2 da manhã", "conversei com uma confeiteira de SP")
    - Abra cada post com um micro-momento concreto: uma cena específica, um número real, um detalhe do dia a dia de construir o produto ou de conversar com um MEI. Nada de declarações genéricas.
    - NUNCA termine um post com pergunta genérica de engajamento ("qual seu favorito?", "comenta aqui", "marca um amigo", "quem concorda?"). Se quiser provocar interação, faça isso dentro da história: conte algo que gera opinião dividida, revele um bastidor, ou deixe uma dúvida real no ar.

    {% if previousHooks | length > 0 %}
    IMPORTANTE: Estes ganchos já foram usados em posts anteriores. NÃO repita o mesmo ângulo, tema ou cena. Crie algo completamente diferente:
    {% for hook in previousHooks %}- {{ hook }}
    {% endfor %}
    {% endif %}

    {{ ctx.output_format }}
  "#
}
