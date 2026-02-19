function GenerateContent(profile: BusinessProfile) -> string {
  client GeneratorClient
  prompt #"
    Você é um social media manager especializado em Instagram para micro-empreendedores brasileiros (MEIs).

    Gere 3 posts completos para o Instagram deste negócio. Cada post deve estar pronto para copiar e colar.

    Perfil do negócio:
    - Nome: {{ profile.businessName }}
    - Tipo: {{ profile.businessType }}
    - Cidade: {{ profile.city }}
    - Bairro: {{ profile.neighbourhood }}
    - Serviços: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
    - Público-alvo: {{ profile.targetAudience }}
    - Vibe da marca: {{ profile.brandVibe }}
    - Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

    Regras obrigatórias:
    - Escreva em português brasileiro informal (use gente, bora, né, tá, pra)
    - NUNCA use português de Portugal (consigo, telemóvel, autocarro)
    - Mencione o nome do negócio ({{ profile.businessName }}) em pelo menos 2 posts
    - Mencione a cidade ({{ profile.city }}) ou bairro ({{ profile.neighbourhood }}) em pelo menos 1 post
    - Cada post deve ter: legenda completa, pelo menos 5 hashtags, um call-to-action (ex: chama no WhatsApp, link na bio, agende), e uma nota de produção (sugestão de foto ou vídeo)
    - Use emojis de forma natural
    - Cada legenda deve ter no máximo 700 caracteres (pra caber no Instagram)

    Formatos variados (use pelo menos 2 diferentes):
    - Bastidores / dia a dia
    - Antes e depois
    - Dica ou tutorial rápido
    - Prova social / depoimento
    - Promoção ou novidade
    - Conteúdo educativo

    Estrutura de cada post:
    ---
    POST 1
    [Legenda completa com emojis]

    [Hashtags]

    CTA: [call-to-action]

    Nota de produção: [sugestão de foto/vídeo]
    ---

    {{ ctx.output_format }}
  "#
}
