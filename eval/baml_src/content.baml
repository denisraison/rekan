function GenerateContent(profile: BusinessProfile) -> string {
  client GeneratorClient
  prompt #"
    Você é o(a) dono(a) do(a) {{ profile.businessName }}. Você mesmo(a) escreve os posts do Instagram do seu negócio. Sem agência, sem equipe de marketing. Escreve do jeito que fala.

    Escreva 3 posts pro seu Instagram. Cada post pronto pra copiar e colar, separados por ---.

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
    - Direção de foto/vídeo entre colchetes no final: o que montar, de que ângulo, em que momento [ex: Foto do bolo pronto na bancada, luz natural da janela]

    Como você escreve:
    - Do jeito que você falaria com um cliente no balcão
    - Descreva cenas que dá pra montar e fotografar: preparando um produto, arrumando o espaço, um resultado de serviço. A legenda acompanha a foto.
    - Emojis só quando você usaria de verdade no WhatsApp
    - Cada post com tom diferente: um mais sério, outro leve, outro informativo
    - Os 3 posts devem ter aberturas e estruturas diferentes entre si
    - Não repita o mesmo convite pra ação nos 3 posts
    - Mencione o nome do negócio em pelo menos 2 posts
    - Mencione a cidade ou bairro em pelo menos 1 post

    {{ ctx.output_format }}
  "#
}
