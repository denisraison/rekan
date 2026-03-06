# Content Prompt Redesign

## Problem

The current `content.baml` prompt produces content that fails the naturalidade judge 0/12. Two optimization cycles confirmed that surface-level prompt tweaks don't fix this. The root cause is structural: the prompt frames the model as a "social media manager," gives a rigid output template (POST 1 / Legenda / Hashtags / CTA: / Nota de producao:), and the model fills template slots. The result reads like AI performing informality.

## Design

### Persona: business owner, not social media manager

Current: "Voce e um social media manager especializado em Instagram para micro-empreendedores brasileiros."

New: "Voce e o(a) dono(a) do(a) {{ profile.businessName }}. Voce mesmo(a) escreve os posts do Instagram do seu negocio. Sem agencia, sem equipe de marketing. Escreve do jeito que fala."

### Structure: requirements, not template

Drop the rigid POST 1 / Legenda / Hashtags / CTA: / Nota de producao: template. Instead, describe what each post needs:

- Legenda (max 700 chars)
- Hashtags do nicho e da regiao (min 5)
- Um convite pra acao (chama no zap, link na bio, comenta, etc.)
- Direcao de foto/video entre colchetes no final: o que montar, de que angulo, em que momento

Posts separated by `---` (required for heuristic parsing).

### Voice: stageable scenes, not fiction

Instead of "conte cenas reais" (which produces invented narratives the owner can't photograph), instruct the model to describe scenes the owner can stage and photograph. The caption accompanies the photo, it doesn't narrate something that already happened.

"Descreva cenas que da pra montar e fotografar: preparando um produto, arrumando o espaco, um resultado de servico. A legenda acompanha a foto."

### Variety: structural, not thematic

Instead of listing format options (bastidores, antes/depois, dica), describe structural variety:

- Aberturas diferentes (um comeca contando algo, outro com pergunta, outro direto ao ponto)
- Estruturas diferentes (um narrativo, outro lista curta, outro conversa com o seguidor)
- Tom diferente por post (um mais serio, outro leve, outro informativo)
- Nao repetir o mesmo CTA nos 3 posts

### What we're dropping

- "Social media manager" persona
- Explicit pt-BR grammar instructions ("use gente, bora, ne, ta, pra")
- "NUNCA use portugues de Portugal" rule
- Rigid output template
- Format menu (bastidores, antes/depois, dica, etc.)
- "Use emojis de forma natural" (replaced by "emojis so quando voce usaria de verdade no WhatsApp")

### Full prompt

```
Voce e o(a) dono(a) do(a) {{ profile.businessName }}. Voce mesmo(a) escreve os posts do Instagram do seu negocio. Sem agencia, sem equipe de marketing. Escreve do jeito que fala.

Escreva 3 posts pro seu Instagram. Cada post pronto pra copiar e colar, separados por ---.

Seu negocio:
- Nome: {{ profile.businessName }}
- Tipo: {{ profile.businessType }}
- Cidade/Bairro: {{ profile.city }}, {{ profile.neighbourhood }}
- Servicos: {% for s in profile.services %}{{ s.name }} (R${{ s.priceBRL }}){% if not loop.last %}, {% endif %}{% endfor %}
- Publico: {{ profile.targetAudience }}
- Vibe: {{ profile.brandVibe }}
- Diferenciais: {% for q in profile.quirks %}{{ q }}{% if not loop.last %}, {% endif %}{% endfor %}

Cada post precisa ter:
- Legenda (maximo 700 caracteres)
- Hashtags do nicho e da regiao (minimo 5)
- Um convite pra acao (chama no zap, link na bio, comenta, etc.)
- Direcao de foto/video entre colchetes no final: o que montar, de que angulo, em que momento [ex: Foto do bolo pronto na bancada, luz natural da janela]

Como voce escreve:
- Do jeito que voce falaria com um cliente no balcao
- Descreva cenas que da pra montar e fotografar: preparando um produto, arrumando o espaco, um resultado de servico. A legenda acompanha a foto.
- Emojis so quando voce usaria de verdade no WhatsApp
- Cada post com tom diferente: um mais serio, outro leve, outro informativo
- Os 3 posts devem ter aberturas e estruturas diferentes entre si
- Nao repita o mesmo convite pra acao nos 3 posts
- Mencione o nome do negocio em pelo menos 2 posts
- Mencione a cidade ou bairro em pelo menos 1 post
```

### Heuristic impact

All 7 heuristics should work as-is:

- `business_name`: keyword search, format-agnostic
- `location`: keyword search, format-agnostic
- `hashtags`: regex for `#word`, needs 3+
- `cta`: pattern list (chama no zap, link na bio, etc.), format-agnostic
- `brazilian_portuguese`: looks for pt-BR markers. Risk: we no longer explicitly instruct "use gente, bora, ne, ta". The model writes in pt-BR naturally but may not hit the exact markers. Monitor after first run.
- `caption_length`: splits on `---`, which we still require
- `production_note`: looks for keywords (foto, video, reels, etc.), bracket format still contains these

### Success criteria

Run `make eval-judges` after implementing. Compare to baseline:

| Criterion | Baseline | Target |
|-----------|----------|--------|
| naturalidade | 0/12 | 6+ |
| variedade | 1/12 | 4+ |
| engajamento | 3/12 | 4+ |
| especificidade | 8/12 | 8+ (no regression) |
| acionavel | 12/12 | 11+ (no regression) |

If `brazilian_portuguese` heuristic fails on multiple profiles, add a light nudge ("escreva em portugues brasileiro informal") without the explicit marker list.
